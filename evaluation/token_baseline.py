#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Count reviewed minimum task input/output text with the model's tokenizer."""
import argparse
import hashlib
import json
from pathlib import Path
import sys

from harness import atomic_json, json_digest

METHOD = 'minimum_required_text_v1'


def measure(spec, encode, tokenizer_version):
    if (not isinstance(spec.get('model'), str) or not spec['model'].strip()
            or not isinstance(spec.get('encoding'), str) or not spec['encoding'].strip()
            or not isinstance(spec.get('tasks'), dict) or not spec['tasks']):
        raise ValueError('spec requires model, encoding and nonempty tasks')
    tasks = {}
    for identifier, task in spec['tasks'].items():
        if not isinstance(task.get('rationale'), str) or not task['rationale'].strip():
            raise ValueError('each task requires a rationale for its minimum text')
        row = {'rationale_sha256': json_digest(task['rationale'])}
        for direction in ('input', 'output'):
            texts = task.get(direction + '_texts')
            if not isinstance(texts, list) or not all(isinstance(t, str) and t for t in texts):
                raise ValueError('minimum text must be an array of nonempty strings')
            if direction == 'input' and not texts:
                raise ValueError('minimum input must include the task goal')
            # Count independent messages/artifacts separately, without invented separators.
            row[direction + '_tokens'] = sum(len(encode(t)) for t in texts)
            row[direction + '_sha256'] = [hashlib.sha256(t.encode()).hexdigest() for t in texts]
        tasks[identifier] = row
    result = {'schema_version': 1, 'method': METHOD, 'estimated': True,
              'model': spec['model'], 'tokenizer': {'encoding': spec['encoding'],
                  'library': 'tiktoken', 'version': tokenizer_version}, 'tasks': tasks}
    validate(result, spec['model'])
    return result


def validate(baseline, model, task_ids=()):
    if (not isinstance(baseline, dict) or baseline.get('schema_version') != 1
            or baseline.get('method') != METHOD or baseline.get('estimated') is not True
            or baseline.get('model') != model):
        raise ValueError('minimum token baseline must match the experiment model and method')
    tokenizer = baseline.get('tokenizer')
    if (not isinstance(tokenizer, dict) or tokenizer.get('library') != 'tiktoken'
            or not all(isinstance(tokenizer.get(k), str) and tokenizer[k].strip()
                       for k in ('encoding', 'version'))):
        raise ValueError('minimum token baseline requires pinned tokenizer provenance')
    tasks = baseline.get('tasks')
    if not isinstance(tasks, dict) or not tasks or set(task_ids) - set(tasks):
        raise ValueError('minimum token baseline must cover every selected task')
    for row in tasks.values():
        if not isinstance(row, dict):
            raise ValueError('invalid minimum token baseline task')
        for direction in ('input', 'output'):
            count = row.get(direction + '_tokens')
            hashes = row.get(direction + '_sha256')
            if type(count) is not int or count < (1 if direction == 'input' else 0):
                raise ValueError('minimum token counts must be nonnegative integers with positive input')
            if not isinstance(hashes, list) or (direction == 'input' and not hashes):
                raise ValueError('minimum token baseline requires text hashes')
            for value in hashes:
                validate_hash(value)
        validate_hash(row.get('rationale_sha256'))
    return baseline


def validate_hash(value):
    if not isinstance(value, str) or len(value) != 64 or any(c not in '0123456789abcdef' for c in value):
        raise ValueError('invalid minimum text hash')


def load(path, model, task_ids=()):
    path = Path(path)
    if not path.is_absolute() or not path.is_file():
        raise ValueError('token_baseline must be an absolute existing file')
    return validate(json.loads(path.read_text(encoding='utf-8')), model, task_ids)


def minimum_for(config, task_id):
    baseline = config.get('token_baseline')
    if baseline is None or task_id not in baseline['tasks']:
        return None
    row = baseline['tasks'][task_id]
    return {**{key: row[key] for key in ('input_tokens', 'output_tokens')},
            'baseline_sha256': json_digest(baseline)}


def provenance(baseline):
    if baseline is None:
        return None
    return {**{key: baseline[key] for key in ('method', 'estimated', 'model', 'tokenizer')},
            'sha256': json_digest(baseline)}


def tooling_usage(run):
    minimum, usage = run.get('minimum_tokens'), run.get('usage')
    if not isinstance(minimum, dict) or not usage:
        return None
    try:
        validate_hash(minimum.get('baseline_sha256'))
    except ValueError:
        return None
    result = {}
    for key in ('input_tokens', 'output_tokens'):
        value, floor = usage.get(key), minimum.get(key)
        if type(value) is not int or value < 0 or type(floor) is not int or floor < 0:
            return None
        # Failed attempts are entirely overhead: they did not deliver the task.
        floor = floor if run['success'] else 0
        if value < floor:
            return None
        result[key] = value - floor
    result['total_tokens'] = result['input_tokens'] + result['output_tokens']
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--spec', type=Path, required=True, help='Private reviewed minimum-text JSON')
    parser.add_argument('--output', type=Path, required=True, help='New baseline JSON; contains hashes, not text')
    args = parser.parse_args()
    if args.output.exists():
        raise ValueError('baseline already exists; use a new output path')
    # Optional, as in measure_cli.py; never substitute characters or another model's encoding.
    import tiktoken
    spec = json.loads(args.spec.read_text(encoding='utf-8'))
    encoding = tiktoken.get_encoding(spec['encoding'])
    baseline = measure(spec, lambda text: encoding.encode(text, disallowed_special=()), tiktoken.__version__)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    atomic_json(args.output, baseline)
    return 0


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError, KeyError, TypeError, AttributeError, ImportError):
        print(json.dumps({'status': 'blocked', 'category': 'minimum_text_or_tokenizer_unavailable'}), file=sys.stderr)
        sys.exit(2)
