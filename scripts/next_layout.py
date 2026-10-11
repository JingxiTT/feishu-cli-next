#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Keep the fork layer bounded: a fixed root, markdown-only docs, sealed evidence.

`check` reports every violation; `seal` records new evidence files and refuses
to change a recorded one. Upstream paths are guarded separately by the
capability gate's source inventory.
"""
import argparse
import hashlib
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
MANIFEST = 'evidence/MANIFEST.sha256'
# Evidence files that describe the directory rather than record a result.
UNSEALED = (MANIFEST, 'evidence/README.md')

# Upstream's root at the synced release. A sync that adds an entry extends this.
UPSTREAM_ROOT = (
    '.codecov.yml', '.gitattributes', '.github', '.gitignore', '.gitleaks.toml',
    '.gitleaksignore', '.golangci.yml', '.goreleaser.yml', '.licenserc.yaml',
    'AGENTS.md', 'CHANGELOG.md', 'LICENSE', 'Makefile', 'README.md', 'README.zh.md',
    'affordance', 'build.sh', 'cmd', 'content_embed.go', 'content_embed_affordance_test.go',
    'errs', 'events', 'extension', 'go.mod', 'go.sum', 'internal', 'isolated-skills',
    'lark_suite_assets_test.go', 'lint', 'main.go', 'main_authsidecar.go',
    'main_noauthsidecar.go', 'main_noauthsidecar_test.go', 'package-lock.json',
    'package.json', 'scripts', 'shortcuts', 'sidecar', 'skill-template', 'skills', 'tests',
)
# The fork layer. The three Go files test or feed the root-package embed, which
# go:embed pins to the repository root.
FORK_ROOT = (
    'README.upstream.md', 'SECURITY.md', 'agent-skills', 'agent_content_embed.go',
    'agent_content_embed_test.go', 'agent_skill_guidance_test.go', 'baseline', 'docs',
    'evaluation', 'evidence',
)


def tracked(root):
    out = subprocess.check_output(['git', '-C', str(root), 'ls-files', '-z']).decode()
    return [name for name in out.split('\0') if name]


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read_manifest(root):
    """Return {path: sha256}; a duplicate or malformed line is itself a violation."""
    entries, problems = {}, []
    path = root / MANIFEST
    if not path.is_file():
        return entries, [MANIFEST + ': missing']
    for number, line in enumerate(path.read_text(encoding='utf-8').splitlines(), 1):
        value, separator, name = line.partition('  ')
        if not separator or len(value) != 64 or not name or name in entries:
            problems.append('{}:{}: malformed or duplicate entry'.format(MANIFEST, number))
            continue
        entries[name] = value
    return entries, problems


def sealed_files(names):
    return sorted(n for n in names if n.startswith('evidence/') and n not in UNSEALED)


def check(root, names=None):
    names = tracked(root) if names is None else names
    problems = []
    allowed = set(UPSTREAM_ROOT) | set(FORK_ROOT)
    for entry in sorted({name.split('/', 1)[0] for name in names} - allowed):
        problems.append('{}: new root entry; place it in an existing directory or extend the allowlist in scripts/next_layout.py'.format(entry))
    for name in sorted(n for n in names if n.startswith('docs/') and not n.endswith('.md')):
        problems.append('{}: docs/ holds markdown only; records belong in evidence/, gate inputs in baseline/'.format(name))
    entries, malformed = read_manifest(root)
    problems.extend(malformed)
    present = sealed_files(names)
    for name in present:
        if name not in entries:
            problems.append('{}: not sealed; run python3 scripts/next_layout.py seal'.format(name))
        elif not (root / name).is_file() or digest(root / name) != entries[name]:
            problems.append('{}: sealed evidence changed; add a new file instead of editing a record'.format(name))
    for name in sorted(set(entries) - set(present)):
        problems.append('{}: sealed evidence removed or renamed'.format(name))
    return problems


def seal(root, names=None):
    """Append unsealed evidence files; never rewrite an existing entry."""
    names = tracked(root) if names is None else names
    entries, malformed = read_manifest(root) if (root / MANIFEST).is_file() else ({}, [])
    if malformed:
        raise ValueError('; '.join(malformed))
    added = [name for name in sealed_files(names) if name not in entries]
    (root / MANIFEST).parent.mkdir(parents=True, exist_ok=True)
    with (root / MANIFEST).open('a', encoding='utf-8', newline='\n') as stream:
        for name in added:
            stream.write('{}  {}\n'.format(digest(root / name), name))
    return added


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=('check', 'seal'))
    parser.add_argument('--target', type=Path, default=ROOT)
    args = parser.parse_args(argv)
    root = args.target.resolve()
    if args.action == 'seal':
        try:
            added = seal(root)
        except ValueError as error:
            print(error, file=sys.stderr)
            return 2
        print('sealed {} new evidence file(s)'.format(len(added)))
    problems = check(root)
    for problem in problems:
        print(problem, file=sys.stderr)
    return 1 if problems else 0


if __name__ == '__main__':
    sys.exit(main())
