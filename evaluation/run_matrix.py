#!/usr/bin/env python3
"""Plan, resume and report the complete implemented original/Next task matrix.

Configuration contains local paths only; public evidence contains content hashes.
Completed failures are cache hits too. No automatic statistical reruns.
"""
import argparse
import ast
from concurrent.futures import ThreadPoolExecutor, as_completed
import fcntl
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

from harness import ORIGINAL_ARMS, aggregate, schedule

ROOT = Path(__file__).resolve().parents[1]
SUITES = {'paired': ('tasks.paired.json', 20260908),
          'docs-hard': ('tasks.docs-hard.json', 20260909),
          'read': ('tasks.read-sections.json', 20260910),
          'scenario': ('tasks.scenarios.json', 20260930),
          'scaling': ('tasks.scaling.json', 20261020),
          'business': ('tasks.business.json', 20261101),
          'foundation': ('tasks.foundation.json', 20260907),
          'contact': ('tasks.contact.json', 20260907)}
PATH_KEYS = ('official_source', 'official_cli', 'next_source', 'next_cli', 'codex', 'auth_home')
# These functions define the measured trial and scoring contract. Harness main
# owns scheduling/checkpoint transport, which is independently bound per new job.
TRIAL_FUNCTIONS = ('run_one', 'process', 'token_usage', 'guidance_delivery',
                   'sanitized_accounting_diagnostics', 'sanitized_verification_failures',
                   'sanitized_api_trace', 'sanitized_trace', 'aggregate', 'schedule')


def read(path):
    return json.loads(Path(path).read_text())


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True).encode()).hexdigest()


def write(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(path.suffix + '.tmp')
    with temporary.open('w') as stream:
        stream.write(json.dumps(value, indent=2) + '\n')
        stream.flush()
        os.fsync(stream.fileno())
    temporary.replace(path)


def seal_batch(folder, path):
    """A terminal batch's bytes must remain unchanged on every cache read."""
    receipt = folder / 'completed.json'
    value = {'batch_sha256': sha(path)}
    if receipt.exists():
        if read(receipt) != value:
            raise ValueError('Completed evidence changed')
    else:
        write(receipt, value)


def tree(path):
    return {str(p.relative_to(path)): sha(p) for p in sorted(path.rglob('*'))
            if p.is_file() and '__pycache__' not in p.parts}


def trial_contract(path):
    body = ast.parse(path.read_text()).body
    nodes = {n.name: ast.dump(n, include_attributes=False) for n in body if isinstance(n, ast.FunctionDef)}
    return digest({'functions': {name: nodes[name] for name in TRIAL_FUNCTIONS},
                   'constants': [ast.dump(n, include_attributes=False) for n in body if isinstance(n, ast.Assign)],
                   'imports': [ast.dump(n, include_attributes=False) for n in body
                               if isinstance(n, (ast.Import, ast.ImportFrom))
                               and not (isinstance(n, ast.Import) and all(a.name == 'fcntl' for a in n.names))
                               and not (isinstance(n, ast.ImportFrom) and n.module == 'contextlib')]})


def runtime_sources(root):
    # Launcher/report edits do not change model observations. All adapter code,
    # fixture contracts and runner resources do; keep a conservative dependency set.
    values = {}
    for p in sorted((root / 'evaluation').rglob('*')):
        rel = p.relative_to(root / 'evaluation')
        if not p.is_file() or '__pycache__' in rel.parts or rel.parts[0] in ('evidence',):
            continue
        if p.suffix == '.py' and p.name not in ('harness.py', 'run_read_batch.py', 'run_matrix.py', 'plot_scaling.py'):
            values[str(rel)] = sha(p)
        elif rel.parts[0] in ('runtime', 'contracts'):
            values[str(rel)] = sha(p)
    values['harness_trial_contract'] = trial_contract(root / 'evaluation/harness.py')
    for name in ('docs/reference/registrations.json.gz', 'scripts/next_test.py'):
        values[name] = sha(root / name)
    return values


def source_identity(root):
    # Frozen archives include Go embed inputs and extension build resources.
    return digest({k: v for k, v in tree(root).items() if not any(p in ('.git', '.tmp') for p in Path(k).parts)})


def environment(config):
    paths = {key: Path(config[key]).resolve(strict=True) for key in PATH_KEYS}
    version = subprocess.check_output([str(paths['codex']), '--version'], text=True).strip()
    if version != 'codex-cli 0.153.4':
        raise ValueError('This matrix requires the pinned Codex CLI 0.153.4')
    values = {'schema': 1, 'model': 'gpt-5.6-luna', 'reasoning': 'medium',
              'codex_version': version, 'codex_sha256': sha(paths['codex']),
              'guidance_delivery': 'file', 'repeats': 3, 'agent_seconds': 180,
              'outer_seconds': 210, 'arms': list(ORIGINAL_ARMS),
              'python_version': sys.version, 'python_sha256': sha(Path(sys.executable).resolve()),
              'launcher_sha256': sha(ROOT / 'evaluation/run_read_batch.py'),
              'section_workers': 3, 'runtime_sources': runtime_sources(ROOT)}
    for arm, prefix in (('official', 'official'), ('next', 'next')):
        source = paths[prefix + '_source']
        values[arm] = {'cli_sha256': sha(paths[prefix + '_cli']),
                       'skills_tree_sha256': digest(tree(source / 'skills')),
                       'source_sha256': source_identity(source)}
    values['next']['entry_tree_sha256'] = digest(tree(paths['next_source'] / 'agent-skills/feishu-cli-next'))
    values['next']['entry_sha256'] = sha(paths['next_source'] / 'agent-skills/feishu-cli-next/SKILL.md')
    return values, paths


def jobs(env):
    result = []
    for kind, (name, seed) in SUITES.items():
        manifest = read(ROOT / 'evaluation' / name)
        for section in dict.fromkeys(t['section'] for t in manifest['tasks']):
            tasks = [t for t in manifest['tasks'] if t['section'] == section]
            index = next(i for i, t in enumerate(manifest['tasks']) if t['section'] == section)
            contract = {'suite': kind, 'section': section, 'seed': seed + index,
                        'manifest_sha256': sha(ROOT / 'evaluation' / name),
                        'environment_sha256': digest(env), 'tasks': [t['id'] for t in tasks],
                        'planned_attempts': len(tasks) * 6}
            result.append({**contract, 'key': digest(contract)})
    return result


def bind_jobs(matrix_jobs, env, paths, output):
    from run_read_batch import extension_host
    for job in matrix_jobs:
        section = job['section']
        skill = 'lark-' + {'docs': 'doc', 'events': 'event', 'packaging': 'shared',
                           'business': 'shared', 'foundation': 'shared'}.get(section, section)
        if job['suite'] == 'paired' and section == 'markdown':
            skill = 'lark-drive'
        entry = paths['official_source'] / 'skills' / skill / 'SKILL.md'
        if not entry.exists():
            if section == 'extensions':
                documents = ('extension/README.md', 'extension/platform/README.md', 'extension/command/examples/chat-brief/main.go')
                content = '\n\n'.join((paths['official_source'] / name).read_text() for name in documents)
                official_skill = hashlib.sha256(content.encode()).hexdigest()
            else:
                official_skill = sha(paths['official_source'] / 'affordance' / (section + '.md'))
        else:
            official_skill = sha(entry)
        arms = {arm: {'cli_sha256': env[arm]['cli_sha256'], 'skill_sha256': official_skill if arm == 'official' else env['next']['entry_sha256']}
                for arm in ORIGINAL_ARMS}
        if section == 'extensions':
            for arm in ORIGINAL_ARMS:
                source = paths['official_source' if arm == 'official' else 'next_source']
                arms[arm]['cli_sha256'] = sha(extension_host(source, output / 'jobs' / job['key'], arm == 'next'))
        job['expected_arms'] = arms


def validate_batch(batch, job, env, complete=True):
    manifest = read(ROOT / 'evaluation' / SUITES[job['suite']][0])
    tasks = [t for t in manifest['tasks'] if t['section'] == job['section']]
    expected = [(t['id'], arm, trial) for t, trial, arm in schedule(tasks, 3, job['seed'], ORIGINAL_ARMS)]
    actual = [(r['task_id'], r['arm'], r['trial']) for r in batch['runs']]
    if (actual != expected[:len(actual)] or len(actual) > len(expected)
            or complete and (actual != expected or not batch.get('completed'))):
        raise ValueError('Batch is not the exact scheduled prefix/completion')
    if (batch['mode'] != 'real' or batch['seed'] != job['seed'] or batch['repeats'] != 3
            or batch['planned_runs'] != len(expected)
            or batch['task_manifest_sha256'] != job['manifest_sha256']
            or set(batch['arms']) != set(ORIGINAL_ARMS)
            or batch.get('arm_selection', {}).get('mode') != 'original_comparison'):
        raise ValueError('Batch differs from frozen job')
    for field in ('model', 'reasoning', 'guidance_delivery'):
        if batch['environment'][field] != env[field]:
            raise ValueError('Batch settings changed')
    if batch['environment']['budget']['timeout_seconds'] != env['agent_seconds']:
        raise ValueError('Batch budget changed')
    if batch['arms'] != job['expected_arms']:
        raise ValueError('Batch binary or selected guidance changed')
    for run in batch['runs']:
        value = run.get('usage') or run.get('known_usage')
        if value and (value['total_tokens'] != value['input_tokens'] + value['output_tokens']
                      or not 0 <= value['cached_input_tokens'] <= value['input_tokens']):
            raise ValueError('Invalid provider accounting')


def import_condition(directory, old_source, env, matrix_jobs, output, reuse_review):
    """Import reviewed condition09 as whole comparisons, never cherry-pick rows."""
    migration = read(reuse_review)
    if migration.get('decision') != 'accepted':
        raise ValueError('Historical migration lacks accepted equivalence review')
    for name, expected in migration['current_sources_sha256'].items():
        if sha(ROOT / 'evaluation' / name) != expected:
            raise ValueError('Historical migration review does not bind current code')
    for name, expected in migration['historical_sources_sha256'].items():
        if sha(directory / name) != expected:
            raise ValueError('Historical migration review does not bind retained evidence')
    if digest(env['runtime_sources']) != migration['measured_equivalence']['runtime_sources_sha256']:
        raise ValueError('Historical migration review does not bind runtime inputs')
    old = read(directory / 'environment.json')
    review = read(directory / 'result-review.json')
    if review.get('decision') != 'accepted':
        raise ValueError('Historical reuse requires accepted source audit')
    for name, expected in review['source_sha256'].items():
        if sha(directory / name) != expected:
            raise ValueError('Historical evidence changed')
    for name, expected in old['execution_sources_sha256'].items():
        if sha(old_source / name) != expected:
            raise ValueError('Historical evaluator source changed')
    checks = {'model': env['model'], 'reasoning': env['reasoning'],
              'codex_sha256': env['codex_sha256'], 'guidance_delivery': env['guidance_delivery'],
              'official_binary_sha256': env['official']['cli_sha256'],
              'candidate_binary_sha256': env['next']['cli_sha256'],
              'official_guidance_tree_sha256': env['official']['skills_tree_sha256'],
              'candidate_inherited_guidance_tree_sha256': env['next']['skills_tree_sha256']}
    if any(old.get(k) != v for k, v in checks.items()):
        raise ValueError('Requested historical reuse does not match measured inputs')
    if runtime_sources(old_source) != env['runtime_sources']:
        raise ValueError('Historical model/fixture/scoring code differs')
    if digest(old['candidate_guidance']) != env['next']['entry_tree_sha256']:
        raise ValueError('Historical entry guide changed')
    for job in matrix_jobs:
        if job['suite'] != old['task_kind']:
            continue
        origin = directory / job['section'] / 'batch-01.json'
        if not origin.exists():
            continue
        batch = read(origin)
        validate_batch(batch, job, env)
        target = output / 'jobs' / job['key']
        target.mkdir(parents=True, exist_ok=True)
        provenance = {'kind': 'historical_reuse', 'condition': old['condition'],
                      'migration_review_sha256': sha(reuse_review),
                      'batch_sha256': sha(origin), 'environment_sha256': sha(directory / 'environment.json'),
                      'review_sha256': sha(directory / 'result-review.json'),
                      'source': str(origin.relative_to(ROOT))}
        existing = target / job['section'] / 'batch-01.json'
        if existing.exists() and sha(existing) != sha(origin):
            raise ValueError('Refusing to replace existing model results')
        write(existing, batch)
        write(target / 'reuse.json', provenance)
        seal_batch(target, existing)


def score(rows, docs):
    before, after = rows['official'], rows['next']
    a, b = before['tokens_per_success'], after['tokens_per_success']
    saving = 1 - b / a if a is not None and a > 0 and b is not None else None
    regression = after['successes'] < before['successes']
    nonregression = False if regression else None if before['inconclusive'] or after['inconclusive'] else True
    threshold = None
    if saving is not None:
        x = after['known_total_tokens'] * before['successes']
        y = before['known_total_tokens'] * after['successes']
        threshold = 2 * x < y if docs else 10 * x <= 7 * y
    gate = 'missed' if False in (threshold, nonregression) else 'unavailable' if None in (threshold, nonregression) else 'met'
    return {'saving_fraction': saving, 'observed_success_nonregression': nonregression,
            'gate': gate, 'target': '>50%' if docs else '>=30%'}


def report(output, matrix_jobs, env):
    tasks, evidence, all_runs = [], [], []
    fresh_tokens = reused_tokens = 0
    complete_jobs = 0
    for job in matrix_jobs:
        folder = output / 'jobs' / job['key']
        path = folder / job['section'] / 'batch-01.json'
        if not path.exists():
            continue
        batch = read(path)
        validate_batch(batch, job, env, complete=False)
        if batch.get('completed'):
            seal_batch(folder, path)
        runs = batch['runs']
        complete_jobs += bool(batch.get('completed'))
        reused = (folder / 'reuse.json').exists()
        tokens = sum((r.get('usage') or r.get('known_usage') or {}).get('total_tokens', 0) for r in runs)
        if reused:
            provenance = read(folder / 'reuse.json')
            if provenance['batch_sha256'] != sha(path):
                raise ValueError('Reused results changed')
            reused_tokens += tokens
        else:
            fresh_tokens += tokens
        evidence.append({'suite': job['suite'], 'section': job['section'], 'key': job['key'],
                         'path': str(path.relative_to(output)), 'sha256': sha(path),
                         'reused': reused, 'completed': batch.get('completed', False), 'attempts': len(runs)})
        grouped = aggregate(runs, ORIGINAL_ARMS)['groups']
        for task in job['tasks']:
            rows = {r['arm']: r for r in grouped if r['scope'] == 'task_id' and r['name'] == task}
            if set(rows) != set(ORIGINAL_ARMS):
                continue
            done = all(row['attempts'] == 3 for row in rows.values())
            verdict = score(rows, job['section'] == 'docs') if done else {'saving_fraction': None, 'gate': 'pending'}
            tasks.append({'suite': job['suite'], 'section': job['section'], 'task': task,
                          'reused': reused, 'completed': done, 'arms': rows, **verdict})
        all_runs.extend(runs)
    completed = complete_jobs == len(matrix_jobs)
    summary = {'schema': 1, 'completed': completed, 'planned_attempts': sum(j['planned_attempts'] for j in matrix_jobs),
               'completed_attempts': len(all_runs), 'fresh_known_tokens': fresh_tokens,
               'reused_known_tokens': reused_tokens, 'known_tokens': fresh_tokens + reused_tokens,
               'unknown_usage_attempts': sum(r.get('usage') is None for r in all_runs),
               'tasks': tasks, 'evidence': evidence}
    # Pooled score is descriptive for this fixed equal-repetition task mixture,
    # available only with all outcomes/accounting usable. Never an individual gate.
    totals = {r['arm']: r for r in aggregate(all_runs, ORIGINAL_ARMS)['groups'] if r['scope'] == 'all'}
    summary['arm_totals'] = totals
    summary['pooled_saving_fraction'] = score(totals, False)['saving_fraction'] if completed and len(totals) == 2 else None
    summary['secondary_raw_token_saving_fraction'] = (1 - totals['next']['known_total_tokens'] / totals['official']['known_total_tokens']
        if completed and len(totals) == 2 and all(r['usage_complete'] for r in totals.values())
        and totals['official']['known_total_tokens'] > 0 else None)
    write(output / 'summary.json', summary)
    lines = ['# Full CLI evaluation matrix', '',
             f"Status: **{'complete' if completed else 'in progress'}**; {len(all_runs)}/{summary['planned_attempts']} attempts recorded.", '',
             'GPT-5.6 Luna / medium; native Codex CLI 0.153.4; original/full versus Next; three repetitions per task and arm; filesystem-only guidance.', '',
             f'New observed provider usage: **{fresh_tokens:,} tokens**. Reused historical usage: **{reused_tokens:,} tokens** (not new spend). Unknown usage attempts: {summary["unknown_usage_attempts"]}.', '',
             '| Suite | Task | Original verified | Next verified | Token saving/success | Gate | Evidence |',
             '| --- | --- | ---: | ---: | ---: | --- | --- |']
    for task in tasks:
        values = [task['arms'][arm] for arm in ORIGINAL_ARMS]
        counts = [f"{r['successes']}/{r['attempts']}" + (f"; {r['inconclusive']} inconclusive" if r['inconclusive'] else '') for r in values]
        saving = task['saving_fraction']
        lines.append('| ' + ' | '.join([task['suite'], task['task'], *counts,
                     'Unavailable' if saving is None else f'{saving:.1%}', task['gate'], 'Reused' if task['reused'] else 'New']) + ' |')
    lines += ['', '## Whole-matrix accounting', '',
              '| Arm | Verified / attempts | Inconclusive | Observed tokens | Complete usage |',
              '| --- | ---: | ---: | ---: | --- |']
    for arm, row in totals.items():
        lines.append(f"| {arm} | {row['successes']}/{row['attempts']} | {row['inconclusive']} | {row['known_total_tokens']:,} | {row['usage_complete']} |")
    pooled = summary['pooled_saving_fraction']
    raw = summary['secondary_raw_token_saving_fraction']
    lines += ['', 'Whole-matrix tokens-per-verified-success saving: **' + ('unavailable' if pooled is None else f'{pooled:.1%}') + '**.', '',
              'Secondary raw token-spend reduction: **' + ('unavailable' if raw is None else f'{raw:.1%}') + '**. This compares all attempted work, without correcting for success; it cannot establish an efficiency gate or replace unavailable primary comparisons.']
    lines += ['', 'All failed, inconclusive and recovery costs remain included. Cached input is counted once in full. Missing accounting, inconclusive outcomes or zero verified successes make affected savings unavailable. Negative savings mean more tokens.', '',
              'Docs require strictly more than 50% saving; other tasks require at least 30%, with observed success nonregression. Three repetitions do not establish statistical success parity. Historical reuse is noncontemporaneous; the provider model alias has no immutable snapshot.', '',
              'Coverage includes every implemented development matrix: paired functions, harder Docs, domain reads, workflow scenarios, scaling, business, foundation and Contact. The development/held-out inventory stubs are not executable additional trials; held-out release acceptance and live-tenant parity remain separate.', '',
              '[Exact results and evidence hashes](summary.json) · [Frozen inputs](environment.json) · [Execution plan](plan.json)', '']
    (output / 'REPORT.md').write_text('\n'.join(lines))
    return summary


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path, required=True, help='Local paths; never commit credentials/config')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--execute', action='store_true', help='Run missing slots, with safe resume; otherwise plan/report only')
    parser.add_argument('--reuse-condition', type=Path, help='Reviewed historical condition09 directory')
    parser.add_argument('--reuse-evaluator', type=Path, help='Its intact frozen evaluator source')
    parser.add_argument('--reuse-review', type=Path, help='Reviewed old/new launcher equivalence bound to exact code hashes')
    args = parser.parse_args()
    if len({bool(args.reuse_condition), bool(args.reuse_evaluator), bool(args.reuse_review)}) != 1:
        parser.error('Historical condition, evaluator and migration review must be supplied together')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    with (output / '.matrix.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        env, paths = environment(read(args.config))
        env_path = output / 'environment.json'
        if env_path.exists() and read(env_path) != env:
            raise ValueError('Inputs changed: use a new output directory; previous evidence is immutable')
        write(env_path, env)
        matrix_jobs = jobs(env)
        bind_jobs(matrix_jobs, env, paths, output)
        write(output / 'plan.json', {'jobs': matrix_jobs, 'planned_attempts': sum(j['planned_attempts'] for j in matrix_jobs)})
        if args.reuse_condition:
            import_condition(args.reuse_condition.resolve(), args.reuse_evaluator.resolve(), env, matrix_jobs, output, args.reuse_review.resolve())
        pending = []
        for job in matrix_jobs:
            folder = output / 'jobs' / job['key']
            batch_path = folder / job['section'] / 'batch-01.json'
            if batch_path.exists() and read(batch_path).get('completed'):
                validate_batch(read(batch_path), job, env)
                seal_batch(folder, batch_path)
                print(json.dumps({'suite': job['suite'], 'section': job['section'], 'status': 'cached'}), flush=True)
                continue
            if not args.execute or (output / 'STOP').exists():
                continue
            pending.append(job)

        def execute_job(job):
            folder = output / 'jobs' / job['key']
            batch_path = folder / job['section'] / 'batch-01.json'
            if (output / 'STOP').exists():
                return
            # Bind complete execution code for new/resumed jobs. Report-only and
            # completed cache reads do not rerun after orchestration-only edits.
            execution = {name: sha(ROOT / 'evaluation' / name) for name in ('harness.py', 'run_read_batch.py', 'run_matrix.py')}
            contract_path = folder / 'execution.json'
            if contract_path.exists() and read(contract_path) != execution:
                raise ValueError('Execution code changed during incomplete job; do not silently resume')
            write(contract_path, execution)
            stop = folder / 'STOP'
            if not stop.is_symlink():
                if stop.exists():
                    raise ValueError('Job stop marker requires reconciliation')
                stop.symlink_to(output / 'STOP')
            if environment(read(args.config))[0] != env:
                raise ValueError('Inputs changed during matrix execution')
            command = [sys.executable, str(ROOT / 'evaluation/run_read_batch.py'),
                       '--task-kind', job['suite'], '--section', job['section'], '--original-comparison',
                       '--guidance-delivery', 'file', '--model', env['model'], '--reasoning', env['reasoning'],
                       '--output-root', str(folder), '--resume']
            for key, path in paths.items():
                command += ['--' + key.replace('_', '-'), str(path)]
            print(json.dumps({'suite': job['suite'], 'section': job['section'], 'status': 'starting',
                              'planned_attempts': job['planned_attempts']}), flush=True)
            result = subprocess.run(command, check=False)
            if not batch_path.exists() or not read(batch_path).get('completed'):
                if (output / 'STOP').exists():
                    return
                raise ValueError('Incomplete job retained; automatic retries forbidden for unresolved attempts')
            if result.returncode not in (0, 1):
                raise ValueError('Launcher execution error')
        errors = []
        with ThreadPoolExecutor(max_workers=env.get('section_workers', 3)) as pool:
            futures = [pool.submit(execute_job, job) for job in pending]
            for future in as_completed(futures):
                try:
                    future.result()
                except Exception as error:
                    errors.append(error)
                    # Stop only on an execution-integrity error, never a task
                    # failure or missed savings target. Finish active attempts.
                    (output / 'STOP').touch()
                report(output, matrix_jobs, env)
        summary = report(output, matrix_jobs, env)
        print(json.dumps({key: summary[key] for key in ('completed', 'planned_attempts', 'completed_attempts', 'fresh_known_tokens', 'reused_known_tokens')}), flush=True)
        if errors:
            raise errors[0]
    return 0


if __name__ == '__main__':
    sys.exit(main())
