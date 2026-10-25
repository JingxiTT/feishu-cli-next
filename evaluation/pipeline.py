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
from live_batch import load_runner_settings

ROOT = Path(__file__).resolve().parents[1]
WORKLOADS = {
    'paired': 'workflow',
    'docs-hard': 'diagnostic',
    'read': 'diagnostic',
    'scenario': 'workflow',
    'scaling': 'diagnostic',
    'business': 'workflow',
    'foundation': 'diagnostic',
    'contact': 'diagnostic',
    'e2e': 'workflow',
}
SUITES = tuple(WORKLOADS)
WORKLOADS_ALLOWED = ('workflow', 'diagnostic')
WORKLOAD_LABELS = {'workflow': 'Realistic workflows', 'diagnostic': 'Diagnostics'}


class PipelineError(ValueError):
    """Operator guidance safe to print without exposing config contents."""


def selected_manifests(config):
    workloads = config.get('workloads', list(WORKLOADS_ALLOWED))
    if (not isinstance(workloads, list) or not workloads or not all(isinstance(c, str) for c in workloads)
            or len(set(workloads)) != len(workloads) or set(workloads) - set(WORKLOADS_ALLOWED)):
        raise PipelineError('Select workflow and/or diagnostic workloads with --workload.')
    allowed = {s for s, workload in WORKLOADS.items() if workload in workloads}
    bindings = config.get('live_manifests')
    if not isinstance(bindings, dict):
        raise PipelineError('Configure live_manifests for real Feishu tasks; simulated backends are not supported.')
    suites = config.get('suites', [s for s in bindings if s in allowed])
    if (not isinstance(suites, list) or not suites or not all(isinstance(s, str) for s in suites)
            or len(set(suites)) != len(suites) or set(suites) - set(SUITES)):
        raise PipelineError('Select unique development suites with --suite: ' + ', '.join(SUITES))
    if set(suites) - allowed:
        raise PipelineError('Suite selection conflicts with workloads; select matching --suite and --workload values.')
    manifests = {}
    for suite in suites:
        path = Path(bindings.get(suite, ''))
        if not path.is_absolute() or not path.is_file():
            raise PipelineError(f'Live suite {suite} is blocked: set live_manifests.{suite} to an absolute manifest path.')
        if path.name == 'tasks.held-out.json':
            raise PipelineError('Held-out tasks are quarantined and cannot be configured for development evaluation.')
        manifest = tasks_from(path)
        if manifest.get('backend') != 'live-feishu-v1' or manifest.get('split') != 'development':
            raise PipelineError('New evaluations require live-feishu-v1 development manifests; synthetic backends are not evaluation.')
        from codex_runner import validate_output_schema
        for task in manifest['tasks']:
            if not isinstance(task['section'], str) or not re.fullmatch(r'[a-zA-Z0-9][a-zA-Z0-9_-]*', task['section']):
                raise PipelineError('Live task sections must be directory-safe names containing letters, digits, underscores or hyphens.')
            validate_output_schema(task['fixture']['answer_schema'])
        manifests[suite] = manifest
    return manifests


def list_sections(config):
    entries = {}
    if 'live_manifests' not in config:
        config = {'suites': ['contact'], **config, 'live_manifests': {
            'contact': str(ROOT / 'evaluation/tasks.live-contact.json')}}
    manifests = selected_manifests(config)
    for suite, manifest in manifests.items():
        counts = {}
        for task in manifest['tasks']:
            counts[task['section']] = counts.get(task['section'], 0) + 1
        for section, count in counts.items():
            entries.setdefault(section, []).append(f"{suite} ({WORKLOADS[suite]}, {count} task{'s' if count != 1 else ''})")
    for section, suites in sorted(entries.items()):
        print(section + ': ' + '; '.join(suites))


def source_digest(directory):
    ignored = {'.git', '.tmp', '__pycache__', 'evidence', 'node_modules', '.venv'}
    return json_digest([(str(p.relative_to(directory)), digest(p))
                        for p in sorted(Path(directory).rglob('*'))
                        if p.is_file() and not ignored.intersection(p.relative_to(directory).parts)
                        and p.name != 'tasks.held-out.json'])


def prepare(config):
    """Validate the entire matrix before any model process can start."""
    if config.get('schema_version') != 1:
        raise PipelineError('schema_version must be 1; start from evaluation/pipeline.example.json.')
    for key in ('official_source', 'next_source', 'official_cli', 'next_cli'):
        path = Path(config.get(key, ''))
        if not path.is_absolute() or not (path.is_dir() if key.endswith('_source') else path.is_file()):
            raise PipelineError(f'Set {key} to an absolute existing ' + ('source directory.' if key.endswith('_source') else 'CLI file.'))
    repeats = config.get('repeats', 3)
    if type(repeats) is not int or repeats < 2:
        raise PipelineError('At least two repeats required; use --repeats 2 or greater.')
    if type(config.get('seed', 20261012)) is not int:
        raise PipelineError('Set seed to an integer.')
    if type(config.get('budget_seconds', 180)) is not int or config.get('budget_seconds', 180) <= 0:
        raise PipelineError('Use a positive --budget-seconds value.')
    if config.get('guidance_delivery', 'file') not in ('file', 'inline'):
        raise PipelineError('Set guidance_delivery to file or inline.')
    sections = config.get('sections', [])
    if not isinstance(sections, list) or not all(isinstance(s, str) for s in sections):
        raise PipelineError('sections must be an array of names; use --list-sections to discover names.')
    manifests = selected_manifests(config)
    available = {t['section'] for m in manifests.values() for t in m['tasks']}
    if set(sections) - available:
        raise PipelineError('Section unavailable in selected suites/workloads. Available sections: '
                            + ', '.join(sorted(available)) + '. Use --list-sections or change --suite/--workload.')
    from live_fixture import validate_live_config
    validate_live_config(config.get('live'))
    cells = config.get('experiments')
    if not isinstance(cells, list) or not cells:
        raise PipelineError('experiments must be a nonempty array; start from evaluation/pipeline.example.json.')
    ids = set()
    jobs = []
    for cell in cells:
        name = cell['id']
        if not isinstance(name, str) or not re.fullmatch(r'[a-zA-Z0-9][a-zA-Z0-9_-]*', name) or name in ids:
            raise PipelineError('Use unique experiment IDs containing only letters, digits, underscores or hyphens.')
        ids.add(name)
        if not all(isinstance(cell.get(k), str) and cell[k] for k in ('harness', 'model', 'reasoning')):
            raise PipelineError('Pin harness, model and reasoning for every experiment.')
        if 'runner_config' in cell:
            adapter = Path(cell['runner_config'])
            if not adapter.is_absolute() or not adapter.is_file():
                raise PipelineError('Set runner_config to an absolute existing adapter settings file.')
            settings = load_runner_settings(adapter)
            if settings['mode'] != 'real':
                raise PipelineError('Evaluation requires a real model runner; controlled adapters belong in tooling tests.')
            backends = settings['supported_backends']
        else:
            for key in ('codex', 'auth_home'):
                if not Path(cell.get(key, '')).is_absolute() or not Path(cell[key]).exists():
                    raise PipelineError(f'Set {key} to an absolute existing native adapter path.')
            backends = [m['backend'] for m in manifests.values()]
        for suite, manifest in manifests.items():
            tasks = [t for t in manifest['tasks'] if not sections or t['section'] in sections]
            if not tasks:
                continue
            if manifest['backend'] not in backends:
                raise PipelineError('Adapter does not support every selected backend; narrow --suite or --section.')
            jobs.append({'experiment': name, 'suite': suite, 'workload': WORKLOADS[suite],
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
                'live_manifests': {s: digest(config['live_manifests'][s]) for s in manifests},
                'live_driver': [digest(arg) for arg in config['live']['driver']
                                if Path(arg).is_absolute() and Path(arg).is_file()],
                'adapters': {c['id']: {'settings': digest(c['runner_config']),
                    'command_files': [digest(arg) for arg in load_runner_settings(Path(c['runner_config']))['runner']
                                      if Path(arg).is_absolute() and Path(arg).is_file()]}
                    if 'runner_config' in c else digest(c['codex']) for c in cells}}
    return {'schema_version': 1, 'identity_sha256': json_digest(identity), 'jobs': jobs,
            'planned_attempts': sum(j['planned_attempts'] for j in jobs),
            'repeats': repeats, 'seed': config.get('seed', 20261012),
            'budget_seconds': config.get('budget_seconds', 180),
            'workloads': [w for w in WORKLOADS_ALLOWED if any(j['workload'] == w for j in jobs)],
            'comparison': ['official', 'next'],
            'measurement': 'descriptive official/Next comparison against real Feishu; task-scoped live verification'}


def command(config, job, output, prepare_only=False):
    # Missing live bindings must never fall back to a simulated API.
    return [sys.executable, str(ROOT / 'evaluation/live_batch.py'),
            '--config', str(output / 'config.local.json'),
            '--experiment', job['experiment'], '--suite', job['suite'],
            '--output-root', str(output / job['experiment'] / job['suite']),
            *[arg for section in job['sections'] for arg in ('--section', section)],
            *(['--prepare-only'] if prepare_only else ['--execute', '--progress'])]


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
            if (job['backend'] == 'live-feishu-v1'
                    and batch.get('environment', {}).get('backend') != 'live-feishu-v1'):
                raise PipelineError('Live report contains a different backend; synthetic results cannot be scored as live.')
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
                rows.append({'experiment': job['experiment'], 'workload': job['workload'],
                             'harness': job['harness'], 'model': job['model'],
                             'suite': job['suite'], 'section': section, 'scope': scope, 'name': name,
                             'official': official, 'next': candidate,
                             'observed_token_difference': saving,
                             'next_verified_outcome_change': (candidate['success_rate'] - official['success_rate'])
                             if comparable else None})
            matched.append({'experiment': job['experiment'], 'workload': job['workload'],
                            'suite': job['suite'], 'section': section,
                            'diagnostic_only': True, 'matched_success': batch['matched_success']})
        coverage.append({**job, 'completed_attempts': count,
                         'complete': bool(complete and count == job['planned_attempts'])})
    result = {'schema_version': 1, 'measurement': plan['measurement'], 'coverage': coverage,
              'workloads': {w: {'planned_attempts': sum(j['planned_attempts'] for j in coverage if j['workload'] == w),
                                'completed_attempts': sum(j['completed_attempts'] for j in coverage if j['workload'] == w)}
                            for w in plan['workloads']},
              'comparisons': rows, 'matched_success': matched,
              'complete': all(j['complete'] for j in coverage)}
    atomic_json(output / 'summary.json', result)
    stream = io.StringIO()
    writer = csv.writer(stream)
    writer.writerow(['experiment', 'workload', 'harness', 'model', 'suite', 'section', 'scope', 'name',
                     'official_successes', 'next_successes', 'official_known_tokens', 'next_known_tokens',
                     'official_input_tokens', 'official_cached_input_tokens', 'official_output_tokens',
                     'next_input_tokens', 'next_cached_input_tokens', 'next_output_tokens',
                     'official_latency_median_seconds', 'next_latency_median_seconds',
                     'official_api_calls', 'next_api_calls', 'official_retries', 'next_retries',
                     'observed_token_difference', 'next_verified_outcome_change'])
    lines = ['# Official CLI / Next evaluation', '', plan['measurement'], '',
             'These are descriptive comparisons for users and developers, not acceptance criteria.',
             'Tokens per verified completion use all attempts, including failures and recovery.',
             'Cached input is included in input tokens. Unknown usage, incomplete schedules, zero successes, controlled runs and inconclusive outcomes have no finite comparison.',
             'Models, harnesses, workloads and task categories are never pooled.']
    for row in rows:
        a, b = row['official'] or {}, row['next'] or {}
        writer.writerow([row[k] for k in ('experiment', 'workload', 'harness', 'model', 'suite', 'section', 'scope', 'name')] +
                        [a.get('successes'), b.get('successes'), a.get('known_total_tokens'),
                         b.get('known_total_tokens')] +
                        [g.get('known_tokens', {}).get(k) for g in (a, b)
                         for k in ('input_tokens', 'cached_input_tokens', 'output_tokens')] +
                        [g.get('latency_median_seconds') for g in (a, b)] +
                        [g.get('api_calls') for g in (a, b)] +
                        [g.get('retries') for g in (a, b)] +
                        [row['observed_token_difference'], row['next_verified_outcome_change']])
    for workload in plan['workloads']:
        counts = result['workloads'][workload]
        lines.extend(['', '## ' + WORKLOAD_LABELS[workload], '',
                      f"Attempts: {counts['completed_attempts']}/{counts['planned_attempts']} completed.", '',
                      '| Experiment | Suite | Task | Official success | Next success | Token difference | Latency O/N | API calls O/N | Retries O/N |',
                      '|---|---|---|---|---|---|---|---|---|'])
        for row in (r for r in rows if r['workload'] == workload and r['scope'] == 'task_id'):
            a, b = row['official'] or {}, row['next'] or {}
            difference = 'unavailable' if row['observed_token_difference'] is None else f"{row['observed_token_difference']:.1%}"
            lines.append(f"| {row['experiment']} | {row['suite']} | {row['name']} | "
                         f"{a.get('successes', 0)}/{a.get('attempts', 0)} | {b.get('successes', 0)}/{b.get('attempts', 0)} | {difference} | "
                         f"{a.get('latency_median_seconds')} / {b.get('latency_median_seconds')} | "
                         f"{a.get('api_calls')} / {b.get('api_calls')} | {a.get('retries')} / {b.get('retries')} |")
    lines.extend(['', f"Completed jobs: {sum(j['complete'] for j in coverage)}/{len(coverage)}.",
                  'Small repeated samples describe observed variability; they do not establish population success parity or release readiness.'])
    (output / 'summary.csv').write_text(stream.getvalue())
    (output / 'REPORT.md').write_text('\n'.join(lines) + '\n')
    return result


def progress(message):
    print(message, file=sys.stderr, flush=True)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, epilog=
        'Start with --list-sections. Without --execute, preparation never bills models. '
        'CLI selections replace the corresponding config arrays; repeat them or supply multiple names.')
    parser.add_argument('--config', type=Path, help='Experiment JSON; see evaluation/pipeline.example.json')
    parser.add_argument('--output', type=Path, help='Run directory outside both frozen source trees')
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument('--execute', action='store_true', help='Bill model calls after validating all jobs')
    mode.add_argument('--report-only', action='store_true', help='Render retained plan/results; only --output required')
    mode.add_argument('--dry-run', action='store_true', help='Print selected plan JSON; no preflight, run files or model calls')
    mode.add_argument('--list-sections', action='store_true', help='List configured live sections, or the bundled live Contact task without config')
    for name, choices in (('section', None), ('suite', tuple(SUITES)),
                          ('workload', WORKLOADS_ALLOWED), ('experiment', None)):
        parser.add_argument('--' + name, action='extend', nargs='+', choices=choices, metavar=name.upper(),
                            help=f'Select {name} names (replaces config selection)')
    parser.add_argument('--repeats', type=int, help='Trials per task per arm, at least 2')
    parser.add_argument('--budget-seconds', type=int, help='Time budget per attempt, applied to both arms')
    args = parser.parse_args(argv)
    filters = (args.section, args.suite, args.workload, args.experiment, args.repeats, args.budget_seconds)
    if args.report_only and any(value is not None for value in filters):
        parser.error('--report-only uses the retained plan; selection and budget overrides are unavailable')
    if args.list_sections and any(value is not None for value in
                                  (args.section, args.experiment, args.repeats, args.budget_seconds)):
        parser.error('--list-sections accepts only --config, --suite and --workload filters')
    if not args.list_sections and not args.report_only and not args.config:
        parser.error('--config is required; start from evaluation/pipeline.example.json')
    if not args.list_sections and not args.dry_run and not args.output:
        parser.error('--output is required for preparation, execution and reports')
    config = json.loads(args.config.read_text()) if args.config and not args.report_only else {}
    if not isinstance(config, dict):
        raise PipelineError('Config must be a JSON object; start from evaluation/pipeline.example.json.')
    for flag, key in (('section', 'sections'), ('suite', 'suites'), ('workload', 'workloads'),
                      ('repeats', 'repeats'), ('budget_seconds', 'budget_seconds')):
        value = getattr(args, flag)
        if value is not None:
            config[key] = list(dict.fromkeys(value)) if isinstance(value, list) else value
    if args.list_sections:
        list_sections(config)
        return 0
    if args.experiment:
        cells = config.get('experiments', [])
        if set(args.experiment) - {c['id'] for c in cells}:
            raise PipelineError('Unknown experiment; choose an ID from the config experiments array.')
        config['experiments'] = [c for c in cells if c['id'] in args.experiment]
    if args.dry_run:
        print(json.dumps(prepare(config), indent=2))
        return 0
    output = args.output.resolve()
    if not args.report_only:
        plan = prepare(config)
        if any(output == Path(config[k]).resolve() or Path(config[k]).resolve() in output.parents
               for k in ('official_source', 'next_source')):
            raise PipelineError('Output must be outside measured source trees; choose another --output directory.')
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    with output_lock(output / 'pipeline'):
        if args.report_only:
            summary = report(json.loads((output / 'plan.json').read_text()), output)
            print(json.dumps({'status': 'report_rendered', 'complete': summary['complete'],
                              'output': str(output), 'report': str(output / 'REPORT.md')}))
            return 0
        path = output / 'plan.json'
        if path.exists() and json.loads(path.read_text()) != plan:
            raise PipelineError('Measured inputs or selection changed; use a new --output directory. '
                                'To resume, repeat the same config and CLI filters.')
        atomic_json(path, plan)
        atomic_json(output / 'config.local.json', config)
        progress(f"Plan: {len(plan['jobs'])} jobs, {plan['planned_attempts']} attempts, "
                 f"{plan['repeats']} repeats per arm; budget {plan['budget_seconds']}s per attempt.")
        progress(f"Reports: {output / 'REPORT.md'}; create {output / 'STOP'} to stop between attempts.")
        # Local adapter and guidance validation only; no tenant calls before --execute.
        for index, job in enumerate(plan['jobs'], 1):
            progress(f"Preflight {index}/{len(plan['jobs'])}: {job['experiment']}/{job['suite']} "
                     f"sections={','.join(job['sections'])}, {job['planned_attempts']} attempts")
            subprocess.run(command(config, job, output, True), check=True)
        failed = False
        if args.execute:
            for index, job in enumerate(plan['jobs'], 1):
                if (output / 'STOP').exists():
                    progress('STOP requested; completed attempts are retained. Remove STOP and repeat the command to resume.')
                    break
                progress(f"Running {index}/{len(plan['jobs'])}: {job['experiment']}/{job['suite']}")
                # The shared stop path is also checked between attempts by each harness.
                root = output / job['experiment'] / job['suite']
                root.mkdir(parents=True, exist_ok=True)
                stop = root / 'STOP'
                if not stop.exists() and not stop.is_symlink():
                    stop.symlink_to(output / 'STOP')
                code = subprocess.run(command(config, job, output), check=False).returncode
                summary = report(plan, output)
                progress(f"Completed {sum(j['completed_attempts'] for j in summary['coverage'])}/"
                         f"{plan['planned_attempts']} attempts; reports updated.")
                if code not in (0, 1):
                    raise PipelineError('Job interrupted or invalid; inspect preceding adapter diagnostics and '
                                        'reconcile in-flight evidence before resuming. No automatic replay occurred.')
                failed |= code != 0
        summary = report(plan, output)
        print(json.dumps({'status': 'prepared' if not args.execute else 'complete' if summary['complete'] else 'stopped',
                          'planned_attempts': plan['planned_attempts'],
                          'completed_attempts': sum(j['completed_attempts'] for j in summary['coverage']),
                          'output': str(output), 'report': str(output / 'REPORT.md')}))
        return int(failed or args.execute and not summary['complete'])


if __name__ == '__main__':
    try:
        sys.exit(main())
    except PipelineError as error:
        print(json.dumps({'status': 'error', 'category': 'invalid_or_changed_pipeline_inputs',
                          'message': str(error)}), file=sys.stderr)
        sys.exit(2)
    except json.JSONDecodeError as error:
        print(json.dumps({'status': 'error', 'category': 'invalid_json',
                          'message': f'Check JSON syntax at line {error.lineno}, column {error.colno}.'}), file=sys.stderr)
        sys.exit(2)
    except (OSError, ValueError, KeyError, TypeError, subprocess.CalledProcessError):
        print(json.dumps({'status': 'error', 'category': 'invalid_or_changed_pipeline_inputs',
                          'message': 'Check config paths and adapter settings against evaluation/PIPELINE.md; '
                                     'inspect preceding adapter diagnostics if preflight failed.'}), file=sys.stderr)
        sys.exit(2)
