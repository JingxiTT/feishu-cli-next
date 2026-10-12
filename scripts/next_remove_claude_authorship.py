#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Write cleaned commit objects and print the new tip; never update refs or push."""
import re
import subprocess
import sys


CLAUDE_TRAILER = re.compile(
    rb'^[ \t]*co-authored-by:[ \t]*(?:Claude\b[^\r\n]*|'
    rb'[^\r\n]*<(?:noreply|claude)@anthropic\.com>[ \t]*)\r?\n?$',
    re.IGNORECASE,
)


def git(*args, data=None):
    return subprocess.check_output(['git', *args], input=data)


def clean_message(message):
    return b''.join(line for line in message.splitlines(keepends=True)
                    if not CLAUDE_TRAILER.fullmatch(line))


def rewrite_commit(raw, rewritten):
    headers, message = raw.split(b'\n\n', 1)
    # Continuation lines belong to their header (notably commit signatures).
    fields = re.split(rb'\n(?! )', headers)
    updated = []
    for field in fields:
        if field.startswith(b'parent '):
            parent = field[7:].decode('ascii')
            field = b'parent ' + rewritten[parent].encode('ascii')
        updated.append(field)
    cleaned = clean_message(message)
    if updated == fields and cleaned == message:
        return raw
    # These signatures bind the old commit/parents and cannot survive rewriting.
    updated = [field for field in updated
               if not field.startswith((b'gpgsig ', b'gpgsig-sha256 ', b'mergetag '))]
    return b'\n'.join(updated) + b'\n\n' + cleaned


def rewrite_history(tip):
    if git('rev-parse', '--is-shallow-repository').strip() != b'false':
        raise ValueError('Refusing to rewrite a shallow repository; fetch full history first')
    rewritten = {}
    for commit in git('rev-list', '--reverse', '--topo-order', tip).decode('ascii').splitlines():
        raw = git('cat-file', 'commit', commit)
        cleaned = rewrite_commit(raw, rewritten)
        rewritten[commit] = (commit if cleaned == raw else
                             git('hash-object', '-t', 'commit', '-w', '--stdin', data=cleaned)
                             .decode('ascii').strip())
    return rewritten[tip]


if __name__ == '__main__':
    tip = git('rev-parse', '--verify', sys.argv[1] + '^{commit}').decode('ascii').strip()
    print(rewrite_history(tip))
