#!/usr/bin/env python3
"""Run frozen development workloads serially with per-section guidance."""
import argparse
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def extension_host(source, output_root, next_mode):
    """Reuse the exact built host across resumes of the same frozen inputs."""
    from scenarios.extensions import build_host
    source = source.resolve()
    digest = hashlib.sha256(str(source).encode() + bytes([next_mode]))
    inputs = sorted(path for path in source.rglob('*') if path.is_file())
    for path in inputs:
        # Frozen source archives have no private working directories.
        if any(part in ('.git', '.tmp') for part in path.relative_to(source).parts):
            continue
        digest.update(str(path.relative_to(source)).encode())
        digest.update(hashlib.sha256(path.read_bytes()).digest())
    for path in (ROOT / 'evaluation/contracts/extensions.json', ROOT / 'evaluation/scenarios/extensions.py',
                 ROOT / 'scripts/next_test.py'):
        digest.update(hashlib.sha256(path.read_bytes()).digest())
    directory = output_root / '.hosts' / digest.hexdigest()
    binary = directory / ('host.exe' if os.name == 'nt' else 'host')
    receipt = directory / 'sha256'
    if binary.exists() or receipt.exists():
        if not binary.is_file() or not receipt.is_file() or hashlib.sha256(binary.read_bytes()).hexdigest() != receipt.read_text():
            raise ValueError('cached extension host is incomplete or changed')
        return binary
    directory.mkdir(parents=True, exist_ok=True)
    built = build_host(source, binary, next_mode)
    receipt.write_text(hashlib.sha256(built.read_bytes()).hexdigest())
    return built


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('official-source', 'official-cli', 'next-source', 'next-cli', 'codex', 'auth-home', 'output-root'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--previous-next-source', type=Path, help='Frozen previous Next source; replaces concise arm for a paired before/after comparison')
    parser.add_argument('--previous-next-cli', type=Path, help='Frozen previous Next binary; requires --previous-next-source')
    parser.add_argument('--section', action='append')
    parser.add_argument('--only-arm', choices=('next',), help='Run only candidate entries from the frozen balanced schedule; not a new balanced comparison')
    parser.add_argument('--original-comparison', action='store_true', help='Run a fresh balanced comparison of only the original CLI and Next; supports all development task suites')
    parser.add_argument('--prepare-only', action='store_true')
    parser.add_argument('--resume', action='store_true', help='Resume validated checkpoint evidence through the harness')
    parser.add_argument('--task-kind', choices=('read', 'scenario', 'scaling', 'business', 'paired', 'docs-hard', 'foundation', 'contact'), default='read')
    parser.add_argument('--model', default='gpt-6-astra', help='Codex model identifier; pin explicitly for a new comparison')
    parser.add_argument('--reasoning', default='high', help='Codex reasoning effort; pin explicitly for a new comparison')
    parser.add_argument('--guidance-delivery', choices=('inline', 'file'), default='inline',
                        help='Shared delivery for every arm: inline skill text or a required complete file read')
    args = parser.parse_args()
    if args.original_comparison and (args.previous_next_source or args.previous_next_cli or args.only_arm):
        parser.error('original comparison cannot combine previous Next or candidate-only options')
    if bool(args.previous_next_source) != bool(args.previous_next_cli):
        parser.error('previous Next source and CLI must be supplied together')
    if args.previous_next_source and args.task_kind != 'paired':
        parser.error('previous Next comparison is supported only for paired tasks')
    manifest_path = ROOT / 'evaluation' / {'read': 'tasks.read-sections.json', 'scenario': 'tasks.scenarios.json', 'scaling': 'tasks.scaling.json', 'business': 'tasks.business.json', 'paired': 'tasks.paired.json', 'docs-hard': 'tasks.docs-hard.json', 'foundation': 'tasks.foundation.json', 'contact': 'tasks.contact.json'}[args.task_kind]
    manifest = json.loads(manifest_path.read_text())
    sections = list(dict.fromkeys(task['section'] for task in manifest['tasks'] if not args.section or task['section'] in args.section))
    if not sections or args.section and set(args.section) - set(sections):
        raise ValueError('unknown section')
    from codex_runner import trace_vocabulary, validate_output_schema
    trace_vocabulary()
    for task in manifest['tasks']:
        if task['section'] in sections:
            validate_output_schema(task['fixture']['answer_schema'])
    failed = False
    with tempfile.TemporaryDirectory(prefix='eval-read-config-') as directory:
        temporary = Path(directory)
        for section in sections:
            if (args.output_root / "STOP").exists():
                print(json.dumps({"status": "stopped_before_next_section"}), flush=True)
                return 0
            index = next(i for i, task in enumerate(manifest["tasks"]) if task["section"] == section)
            skill_name = 'lark-' + {'docs': 'doc', 'events': 'event', 'packaging': 'shared', 'business': 'shared', 'foundation': 'shared'}.get(section, section)
            # The original Markdown skill excludes Markdown-to-online-Doc work
            # and routes it to Drive import. Keep that full source guide intact.
            official_skill = 'lark-drive' if args.task_kind == 'paired' and section == 'markdown' else skill_name
            official_entry = args.official_source / 'skills' / official_skill / 'SKILL.md'
            # An absent skill is a documented source-matched affordance fallback.
            if not official_entry.is_file():
                distribution = temporary / section / 'official'
                shutil.copytree(args.official_source / 'skills', distribution)
                official_entry = distribution / official_skill / 'SKILL.md'
                official_entry.parent.mkdir()
                if section == 'extensions':
                    documents = ('extension/README.md', 'extension/platform/README.md', 'extension/command/examples/chat-brief/main.go')
                    official_entry.write_text('\n\n'.join((args.official_source / name).read_text() for name in documents))
                else:
                    shutil.copyfile(args.official_source / 'affordance' / (section + '.md'), official_entry)
            concise_entry = None
            if not args.original_comparison and not args.previous_next_source:
                concise_distribution = temporary / section / 'concise'
                shutil.copytree(args.official_source / 'skills', concise_distribution)
                concise_entry = concise_distribution / skill_name / 'SKILL.md'
                concise_entry.parent.mkdir(exist_ok=True)
                concise_source = (ROOT / 'evaluation/official-contact-concise/SKILL.md' if section == 'contact'
                                  else ROOT / 'evaluation/official-concise/SKILL.md' if section == 'foundation'
                                  else ROOT / 'evaluation/concise' / ('packaging' if section == 'business' else section) / 'SKILL.md')
                shutil.copyfile(concise_source, concise_entry)
            official_cli, next_cli = args.official_cli, args.next_cli
            if section == 'extensions':
                if args.resume or args.original_comparison:
                    official_cli = extension_host(args.official_source, args.output_root, False)
                    next_cli = extension_host(args.next_source, args.output_root, True)
                else:
                    from scenarios.extensions import build_host
                    official_cli = build_host(args.official_source, temporary / 'official-host', False)
                    next_cli = build_host(args.next_source, temporary / 'next-host', True)
            config = {'mode': 'real', 'backend': manifest['backend'], 'model': args.model,
                      'guidance_delivery': args.guidance_delivery,
                      'reasoning': args.reasoning, 'tokenizer': 'provider-undisclosed', 'tools': ['shell', 'files'],
                      'permissions': 'workspace-write; managed proxy loopback-only with local binding; isolated synthetic sidecar',
                      'budget': {'timeout_seconds': 180}, 'timeout_seconds': 210,
                      'accounting_method': 'codex-cli 0.153.4 provider turn.completed; cached input included in full',
                      'runner': [sys.executable, str(ROOT / 'evaluation/codex_runner.py'), '--codex', str(args.codex),
                                 '--auth-home', str(args.auth_home), '--codex-version', 'codex-cli 0.153.4'],
                      'verifier': [sys.executable, str(ROOT / 'evaluation' / {'read': 'read_verifier.py', 'foundation': 'foundation_verifier.py', 'contact': 'contact_verifier.py'}.get(args.task_kind, 'scenario_verifier.py'))],
                      'arms': {'official': {'cli': str(official_cli), 'skill': str(official_entry)},
                               'official_concise': {'cli': str(official_cli), 'skill': str(concise_entry)},
                               'next': {'cli': str(next_cli), 'skill': str(args.next_source / 'agent-skills/feishu-cli-next/SKILL.md')}}}
            if args.previous_next_source:
                del config['arms']['official_concise']
                config['arms']['next_before'] = {
                    'cli': str(args.previous_next_cli),
                    'skill': str(args.previous_next_source / 'agent-skills/feishu-cli-next/SKILL.md')}
            elif args.original_comparison:
                del config['arms']['official_concise']
            if section == 'packaging':
                for arm in ('official', 'official_concise'):
                    if arm not in config['arms']:
                        continue
                    config['arms'][arm]['selected_skill_source'] = str(args.official_source / 'skills/lark-shared')
                config['arms']['next']['selected_skill_source'] = str(args.next_source / 'agent-skills/feishu-cli-next')
                config['permissions'] = 'workspace-write; protected home/config read-only; network denied; native enforcement probed'
            if args.task_kind == 'foundation':
                config['permissions'] = 'workspace-write; shell network disabled; isolated Feishu config'
            from harness import validate_config
            validate_config(config)
            path = temporary / (section + '.json')
            path.write_text(json.dumps(config), encoding='utf-8')
            path.chmod(0o600)
            if args.prepare_only:
                continue
            if not args.resume and (args.output_root / section / 'batch-01.json').exists():
                raise ValueError('refusing to overwrite completed or partial model evidence')
            attempts = 3 * (1 if args.only_arm else len(config['arms'])) * sum(task['section'] == section for task in manifest['tasks'])
            selection = {'only_arm': args.only_arm, 'balanced_comparison': False} if args.only_arm else {}
            if args.original_comparison:
                selection = {'comparison': 'original', 'balanced_comparison': True}
            print(json.dumps({'section': section, 'status': 'starting', 'planned_attempts': attempts, **selection}), flush=True)
            result = subprocess.run([sys.executable, str(ROOT / 'evaluation/harness.py'),
                                     '--tasks', str(manifest_path), '--section', section,
                                     '--config', str(path), '--output', str(args.output_root / section / 'batch-01.json'),
                                     '--seed', str({'read': 20260910, 'scenario': 20260930, 'scaling': 20261020, 'business': 20261101, 'paired': 20260908, 'docs-hard': 20260909, 'foundation': 20260907, 'contact': 20260907}[args.task_kind] + index), '--repeats', '3',
                                     '--stop-file', str(args.output_root / 'STOP')] +
                                    (['--only-arm', args.only_arm] if args.only_arm else []) +
                                    (['--original-comparison'] if args.original_comparison else []) +
                                    (['--resume'] if args.resume else []), check=False)
            failed |= result.returncode != 0
    print(json.dumps({'status': 'prepared' if args.prepare_only else 'complete_with_failures' if failed else 'complete',
                      'sections': sections}), flush=True)
    return int(failed)


if __name__ == '__main__':
    sys.exit(main())
