#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Plan, run and report independent official/Next harness/model comparisons."""
import argparse
import csv
import io
import json
from pathlib import Path
import re
import subprocess
import sys

from harness import atomic_json, digest, json_digest, output_lock, tasks_from
from run_read_batch import load_runner_settings

ROOT = Path(__file__).resolve().parents[1]
SUITES = {'paired': 'paired', 'docs-hard': 'docs-hard', 'read': 'read-sections',
          'scenario': 'scenarios', 'scaling': 'scaling', 'business': 'business',
          'foundation': 'foundation', 'contact': 'contact', 'e2e': 'e2e'}
CATEGORIES = {'subsystem': tuple(s for s in SUITES if s != 'e2e'), 'e2e': ('e2e',)}


def source_digest(directory):
    ignored = {'.git', '.tmp', '__pycache__', 'evidence', 'node_modules', '.venv'}
    return json_digest([(str(p.relative_to(directory)), digest(p))
                        for p in sorted(Path(directory).rglob('*'))
                        if p.is_file() and not ignored.intersection(p.relative_to(directory).parts)
                        and p.name != 'tasks.held-out.json'])


def prepare(config):
    """Validate the entire matrix before any model process can start."""
    if config.get('schema_version') != 1:
        raise ValueError('schema_version must be 1')
    for key in ('official_source', 'next_source', 'official_cli', 'next_cli'):
        path = Path(config[key])
        if not path.is_absolute() or not path.exists():
            raise ValueError('absolute existing product paths required')
    repeats = config.get('repeats', 3)
    if type(repeats) is not int or repeats < 2:
        raise ValueError('at least two repeats required')
    if type(config.get('seed', 20261012)) is not int:
        raise ValueError('integer seed required')
    if type(config.get('budget_seconds', 180)) is not int or config.get('budget_seconds', 180) <= 0:
        raise ValueError('positive budget_seconds required')
    if config.get('guidance_delivery', 'file') not in ('file', 'inline'):
        raise ValueError('invalid guidance delivery')
    categories = config.get('categories', list(CATEGORIES))
    if (not isinstance(categories, list) or not categories or not all(isinstance(c, str) for c in categories)
            or len(set(categories)) != len(categories) or set(categories) - set(CATEGORIES)):
        raise ValueError('select subsystem and/or e2e categories')
    allowed = {s for c in categories for s in CATEGORIES[c]}
    suites = config.get('suites', [s for s in SUITES if s in allowed])
    if not isinstance(suites, list) or not suites or len(set(suites)) != len(suites) or set(suites) - set(SUITES):
        raise ValueError('select unique supported development suites')
    if set(suites) - allowed:
        raise ValueError('suite selection conflicts with categories')
    sections = config.get('sections', [])
    if not isinstance(sections, list) or not all(isinstance(s, str) for s in sections):
        raise ValueError('sections must be an array of names')
    manifests = {suite: tasks_from(ROOT / f'evaluation/tasks.{SUITES[suite]}.json') for suite in suites}
    available = {t['section'] for m in manifests.values() for t in m['tasks']}
    if set(sections) - available:
        raise ValueError('unknown section selection')
    cells = config['experiments']
    if not isinstance(cells, list) or not cells:
        raise ValueError('experiments must be a nonempty array')
    ids = set()
    jobs = []
    for cell in cells:
        name = cell['id']
        if not isinstance(name, str) or not re.fullmatch(r'[a-zA-Z0-9][a-zA-Z0-9_-]*', name) or name in ids:
            raise ValueError('unique safe experiment IDs required')
        ids.add(name)
        if not all(isinstance(cell.get(k), str) and cell[k] for k in ('harness', 'model', 'reasoning')):
            raise ValueError('pin harness, model and reasoning for every experiment')
        if 'runner_config' in cell:
            adapter = Path(cell['runner_config'])
            if not adapter.is_absolute() or not adapter.is_file():
                raise ValueError('absolute runner_config required')
            settings = load_runner_settings(adapter)
            backends = settings['supported_backends']
        else:
            for key in ('codex', 'auth_home'):
                if not Path(cell[key]).is_absolute() or not Path(cell[key]).exists():
                    raise ValueError('absolute native adapter paths required')
            backends = [m['backend'] for m in manifests.values()]
        for suite, manifest in manifests.items():
            tasks = [t for t in manifest['tasks'] if not sections or t['section'] in sections]
            if not tasks:
                continue
            if manifest['backend'] not in backends:
                raise ValueError('adapter does not support every selected backend; narrow the suite selection')
            jobs.append({'experiment': name, 'suite': suite, 'category': 'e2e' if suite == 'e2e' else 'subsystem',
                         'harness': cell['harness'],
                         'model': cell['model'], 'reasoning': cell['reasoning'],
                         'backend': manifest['backend'], 'task_ids': [t['id'] for t in tasks],
                         'sections': list(dict.fromkeys(t['section'] for t in tasks)),
                         'section_tasks': {s: [t['id'] for t in tasks if t['section'] == s]
                                           for s in dict.fromkeys(t['section'] for t in tasks)},
                         'planned_attempts': len(tasks) * repeats * 2})
    # Hash public inputs; never read auth homes or credential contents.
    identity = {'config': config, 'python': {'executable': sys.executable, 'version': sys.version,
                                           'sha256': digest(sys.executable)},
                'products': {key: source_digest(config[key]) if key.endswith('_source')
                 else digest(config[key]) for key in ('official_source', 'next_source', 'official_cli', 'next_cli')},
                'evaluator': {str(p.relative_to(ROOT)): digest(p) for p in sorted((ROOT / 'evaluation').rglob('*'))
                              if p.is_file() and p.suffix in ('.py', '.json', '.md')
                              and p.name != 'tasks.held-out.json' and 'reports' not in p.parts},
                'adapters': {c['id']: {'settings': digest(c['runner_config']),
                    'command_files': [digest(arg) for arg in load_runner_settings(Path(c['runner_config']))['runner']
                                      if Path(arg).is_absolute() and Path(arg).is_file()]}
                    if 'runner_config' in c else digest(c['codex']) for c in cells}}
    return {'schema_version': 1, 'identity_sha256': json_digest(identity), 'jobs': jobs,
            'planned_attempts': sum(j['planned_attempts'] for j in jobs),
            'repeats': repeats, 'seed': config.get('seed', 20261012),
            'categories': [c for c in CATEGORIES if any(j['category'] == c for j in jobs)],
            'comparison': ['official', 'next'], 'measurement': 'development fixtures; not live tenant parity'}


def command(config, job, output, prepare_only=False):
    cell = next(c for c in config['experiments'] if c['id'] == job['experiment'])
    argv = [sys.executable, str(ROOT / 'evaluation/run_read_batch.py'), '--original-comparison',
            '--task-kind', job['suite'], '--output-root', str(output / job['experiment'] / job['suite']),
            '--model', cell['model'], '--reasoning', cell['reasoning'],
            '--guidance-delivery', config.get('guidance_delivery', 'file'),
            '--repeats', str(config.get('repeats', 3)), '--seed', str(config.get('seed', 20261012)),
            '--budget-seconds', str(config.get('budget_seconds', 180))]
    for key in ('official_source', 'next_source', 'official_cli', 'next_cli'):
        argv.extend(['--' + key.replace('_', '-'), config[key]])
    for section in job['sections']:
        argv.extend(['--section', section])
    if 'runner_config' in cell:
        argv.extend(['--runner-config', cell['runner_config']])
    else:
        argv.extend(['--codex', cell['codex'], '--auth-home', cell['auth_home']])
    argv.append('--prepare-only' if prepare_only else '--resume')
    return argv


def report(plan, output):
    rows, matched, coverage = [], [], []
    for job in plan['jobs']:
        complete = True
        count = 0
        for section in job['sections']:
            path = output / job['experiment'] / job['suite'] / section / 'batch-01.json'
            if not path.exists():
                complete = False
                continue
            batch = json.loads(path.read_text())
            count += len(batch['runs'])
            expected = {(task, trial, arm) for task in job['section_tasks'][section]
                        for trial in range(plan['repeats']) for arm in ('official', 'next')}
            observed = {(r['task_id'], r['trial'], r['arm']) for r in batch['runs']}
            balanced = observed == expected and len(batch['runs']) == len(expected)
            finished = batch['completed'] and balanced and batch['mode'] == 'real'
            complete &= batch['completed'] and balanced
            groups = {(g['scope'], g['name'], g['arm']): g for g in batch['groups']}
            for (scope, name, arm), group in groups.items():
                attempts = [r for r in batch['runs'] if r['arm'] == arm
                            and (scope == 'all' or r[scope] == name)]
                known = [r['usage'] or r.get('known_usage') for r in attempts]
                group['known_tokens'] = {key: sum(u.get(key, 0) for u in known if u)
                                         for key in ('input_tokens', 'cached_input_tokens', 'output_tokens')}
            for scope, name in sorted({(g['scope'], g['name']) for g in batch['groups']}):
                official = groups.get((scope, name, 'official'))
                candidate = groups.get((scope, name, 'next'))
                comparable = (finished and official and candidate and official['usage_complete']
                              and candidate['usage_complete'] and not official['inconclusive']
                              and not candidate['inconclusive'] and official['tokens_per_success']
                              and candidate['tokens_per_success'] is not None)
                saving = 1 - candidate['tokens_per_success'] / official['tokens_per_success'] if comparable else None
                rows.append({'experiment': job['experiment'], 'category': job['category'],
                             'harness': job['harness'], 'model': job['model'],
                             'suite': job['suite'], 'section': section, 'scope': scope, 'name': name,
                             'official': official, 'next': candidate, 'saving_fraction': saving,
                             'observed_success_nonregression': candidate['success_rate'] >= official['success_rate']
                             if comparable else None})
            matched.append({'experiment': job['experiment'], 'category': job['category'],
                            'suite': job['suite'], 'section': section,
                            'diagnostic_only': True, 'matched_success': batch['matched_success']})
        coverage.append({**job, 'completed_attempts': count,
                         'complete': bool(complete and count == job['planned_attempts'])})
    result = {'schema_version': 1, 'measurement': plan['measurement'], 'coverage': coverage,
              'categories': {c: {'planned_attempts': sum(j['planned_attempts'] for j in coverage if j['category'] == c),
                                 'completed_attempts': sum(j['completed_attempts'] for j in coverage if j['category'] == c)}
                             for c in plan['categories']},
              'comparisons': rows, 'matched_success': matched,
              'complete': all(j['complete'] for j in coverage)}
    atomic_json(output / 'summary.json', result)
    stream = io.StringIO()
    writer = csv.writer(stream)
    writer.writerow(['experiment', 'category', 'harness', 'model', 'suite', 'section', 'scope', 'name',
                     'official_successes', 'next_successes', 'official_known_tokens', 'next_known_tokens',
                     'official_input_tokens', 'official_cached_input_tokens', 'official_output_tokens',
                     'next_input_tokens', 'next_cached_input_tokens', 'next_output_tokens', 'saving_fraction'])
    lines = ['# Official CLI / Next evaluation', '', plan['measurement'], '',
             'Savings use all-attempt tokens / verified successes, including failures and recovery.',
             'Missing usage, incomplete schedules, zero successes, controlled runs and inconclusive outcomes have no savings score.',
             'Cached input is included in input tokens. Models/harnesses and evaluation categories are never pooled.']
    for row in rows:
        a, b = row['official'] or {}, row['next'] or {}
        writer.writerow([row[k] for k in ('experiment', 'category', 'harness', 'model', 'suite', 'section', 'scope', 'name')] +
                        [a.get('successes'), b.get('successes'), a.get('known_total_tokens'),
                         b.get('known_total_tokens')] +
                        [g.get('known_tokens', {}).get(k) for g in (a, b)
                         for k in ('input_tokens', 'cached_input_tokens', 'output_tokens')] + [row['saving_fraction']])
    for category in plan['categories']:
        title = 'Subsystem function evaluation' if category == 'subsystem' else 'E2E task evaluation'
        counts = result['categories'][category]
        lines.extend(['', '## ' + title, '',
                      f"Attempts: {counts['completed_attempts']}/{counts['planned_attempts']} completed.", '',
                      '| Experiment | Suite | Task | Official success | Next success | Token saving |',
                      '|---|---|---|---|---|---|'])
        for row in (r for r in rows if r['category'] == category and r['scope'] == 'task_id'):
            a, b = row['official'] or {}, row['next'] or {}
            saving = 'unavailable' if row['saving_fraction'] is None else f"{row['saving_fraction']:.1%}"
            lines.append(f"| {row['experiment']} | {row['suite']} | {row['name']} | "
                         f"{a.get('successes', 0)}/{a.get('attempts', 0)} | {b.get('successes', 0)}/{b.get('attempts', 0)} | {saving} |")
    lines.extend(['', f"Completed jobs: {sum(j['complete'] for j in coverage)}/{len(coverage)}.",
                  'Small repeated samples describe observed variability; they do not establish population success parity.'])
    (output / 'summary.csv').write_text(stream.getvalue())
    (output / 'REPORT.md').write_text('\n'.join(lines) + '\n')
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--execute', action='store_true', help='Bill model calls after validating all jobs')
    parser.add_argument('--report-only', action='store_true', help='Render retained plan/results without launching processes')
    args = parser.parse_args()
    if args.execute and args.report_only:
        parser.error('execute and report-only are mutually exclusive')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    with output_lock(output / 'pipeline'):
        if args.report_only:
            report(json.loads((output / 'plan.json').read_text()), output)
            return 0
        config = json.loads(args.config.read_text())
        if any(output == Path(config[k]).resolve() or Path(config[k]).resolve() in output.parents
               for k in ('official_source', 'next_source')):
            raise ValueError('output must be outside measured source trees')
        plan = prepare(config)
        path = output / 'plan.json'
        if path.exists() and json.loads(path.read_text()) != plan:
            raise ValueError('measured inputs changed; use a new output directory')
        atomic_json(path, plan)
        # All adapters, guidance and extension builds must preflight before billing.
        for job in plan['jobs']:
            subprocess.run(command(config, job, output, True), check=True)
        failed = False
        if args.execute:
            for job in plan['jobs']:
                if (output / 'STOP').exists():
                    break
                # The shared stop path is also checked between attempts by each harness.
                root = output / job['experiment'] / job['suite']
                root.mkdir(parents=True, exist_ok=True)
                stop = root / 'STOP'
                if not stop.exists() and not stop.is_symlink():
                    stop.symlink_to(output / 'STOP')
                code = subprocess.run(command(config, job, output), check=False).returncode
                if code not in (0, 1):
                    report(plan, output)
                    raise ValueError('job interrupted or invalid; reconcile before resuming')
                failed |= code != 0
        summary = report(plan, output)
        print(json.dumps({'status': 'prepared' if not args.execute else 'complete' if summary['complete'] else 'stopped',
                          'planned_attempts': plan['planned_attempts'], 'output': str(output)}))
        return int(failed or args.execute and not summary['complete'])


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError, KeyError, TypeError, subprocess.CalledProcessError):
        print(json.dumps({'status': 'error', 'category': 'invalid_or_changed_pipeline_inputs'}), file=sys.stderr)
        sys.exit(2)
