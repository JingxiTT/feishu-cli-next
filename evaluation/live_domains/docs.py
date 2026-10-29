# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Native Docx block verification, including formatting and untouched blocks."""
import json
import re

from live_domains.session import exact, require

LINK = 'https://example.com/evaluation'
CONTENT = ('<h1>Release notes</h1><p>Milestone: 2027-03-18</p>'
           '<p>Old <a href="' + LINK + '"><b>status</b></a> — 你好</p>'
           '<p>Keep résumé 😀 and 001 unchanged.</p>')
MARKDOWN = ('# Release checklist\n\nKeep **status** and [reference](' + LINK + ').\n\n'
            '| Code | Value |\n| --- | --- |\n| 001 | 你好 😀 |\n\n```python\nprint("ready")\n```\n')


def audit_lines():
    return [f"{'R41' if i % 5 == 0 else 'R42'} | {'APPROVED' if i % 3 == 0 else 'PENDING'} | "
            f"INV-{i:05d} | Amount={i % 23 - 4} | résumé 你好 😀" for i in range(520)]


def audit_report(native):
    chosen = []
    for block in native:
        match = re.fullmatch(r'(R4[12]) \| (APPROVED|PENDING) \| (INV-\d{5}) \| Amount=(-?\d+) \| résumé 你好 😀',
                             text(block))
        if match and match[1] == 'R42' and match[2] == 'APPROVED' and int(match[4]) > 0:
            chosen.append((block['block_id'], match[3], int(match[4])))
    return {'block_ids': sorted(row[0] for row in chosen), 'invoice_codes': sorted(row[1] for row in chosen),
            'record_count': len(chosen), 'total_amount': sum(row[2] for row in chosen)}


def text(block):
    return ''.join(element.get('text_run', {}).get('content', '')
                   for value in block.values() if isinstance(value, dict)
                   for element in value.get('elements', []))


def blocks(s, token):
    items = s.pages('/open-apis/docx/v1/documents/' + token + '/blocks', page_size=500)
    require(all(isinstance(b.get('block_id'), str) and b['block_id'] for b in items)
            and len({b['block_id'] for b in items}) == len(items), 'duplicate_or_missing_block_id')
    return items


def styled_status(block, prefix):
    runs = block.get('text', {}).get('elements', [])
    chars = [(char, run.get('text_run', {}).get('text_element_style', {}))
             for run in runs for char in run.get('text_run', {}).get('content', '')]
    if ''.join(char for char, _ in chars) != prefix + ' status — 你好':
        return False
    for index, (_, style) in enumerate(chars):
        marked = len(prefix) + 1 <= index < len(prefix) + 7
        if bool(style.get('bold')) != marked or style.get('link', {}).get('url', '') != (LINK if marked else ''):
            return False
    return True


def setup(s):
    folder = s.folder()
    if s.request['task']['id'] == 'common-docs-create':
        (s.workspace / 'source.md').write_text(MARKDOWN)
        return ('Create one native document titled ' + s.data['tag'] + ' inside folder ' + folder
                + ' from source.md. Preserve its heading, bold text, link, table and code block. '
                  'Return its document_id. Do not create other documents.')
    data = s.mutation('create-document', lambda: s.api('POST', '/open-apis/docx/v1/documents',
        body={'folder_token': folder, 'title': s.data['tag']}), ('docx', 'document.document_id'))
    token = data['document']['document_id']
    s.private_drive(token, 'docx')
    s.mutation('seed-document', lambda: s.call(['docs', '+update', '--doc', token,
        '--command', 'overwrite', '--doc-format', 'xml', '--content', CONTENT, '--as', 'user']))
    if s.request['task']['id'] == 'common-docs-audit-large':
        roots = [block for block in blocks(s, token) if block.get('block_type') == 1]
        require(len(roots) == 1, 'native_document_root_missing')
        lines = audit_lines()
        for offset in range(0, len(lines), 50):
            children = [{'block_type': 2, 'text': {'elements': [{'text_run': {'content': line}}]}}
                        for line in lines[offset:offset + 50]]
            result = s.mutation('seed-audit-paragraphs', lambda children=children: s.api('POST',
                '/open-apis/docx/v1/documents/' + token + '/blocks/' + roots[0]['block_id'] + '/children',
                body={'children': children, 'index': -1}))
            require(len(result.get('children', [])) == len(children), 'native_paragraph_seed_incomplete')
    before = blocks(s, token)
    require(len({b['block_id'] for b in before}) == len(before), 'duplicate_native_blocks')
    require(sum(styled_status(b, 'Old') for b in before) == 1, 'native_doc_seed_mismatch')
    s.data.update(document_id=token, before=before)
    s.save()
    if s.request['task']['id'] == 'common-docs-audit-large':
        observed = [text(block) for block in before]
        require(all(observed.count(line) == 1 for line in audit_lines()), 'native_audit_seed_mismatch')
        return ('Audit the entire native document ' + token + ', including more than 500 blocks and all '
                'continuation pages. Among invoice paragraphs of the form release | status | invoice '
                'code | Amount=number | note, select only R42, APPROVED, and strictly positive Amount. '
                'Ignore R41, PENDING, zero/negative amounts and unrelated prose. Write audit.json with '
                'sorted native block_ids, sorted string invoice_codes (preserving leading zeros), '
                'record_count and total_amount. Preserve the entire document, all block IDs, formatting '
                'and links; do not edit anything. Return exactly {"path":"audit.json"}.')
    if s.request['task']['id'] == 'common-docs-read':
        return ('In document ' + token + ', find the milestone date. Return the document_id, '
                'milestone date and the native block_id containing the date. Do not edit it.')
    return ('In document ' + token + ', replace only "Old status" with "New status". '
            'Preserve the bold formatting and existing link on "status", the Unicode text and '
            'every other block. Return the document_id.')


def verify(s, answer):
    task = s.request['task']['id']
    token = s.data['document_id']
    s.private_drive(token, 'docx')
    after = blocks(s, token)
    if task == 'common-docs-audit-large':
        return (answer.keys() == {'path'} and exact(after, s.data['before'])
                and exact(json.loads(s.artifact(answer['path']).read_text()), audit_report(after)))
    if task == 'common-docs-read':
        sources = [b['block_id'] for b in after if text(b) == 'Milestone: 2027-03-18']
        if len(sources) != 1:
            return False
        return (exact(after, s.data['before']) and answer == {
            'document_id': token, 'milestone': '2027-03-18', 'block_id': sources[0]})
    before = {b['block_id']: b for b in s.data['before']}
    now = {b['block_id']: b for b in after}
    targets = [key for key, b in before.items() if styled_status(b, 'Old')]
    if answer != {'document_id': token} or before.keys() != now.keys() or len(targets) != 1:
        return False
    target = targets[0]
    return styled_status(now[target], 'New') and all(exact(before[k], now[k]) for k in before if k != target)


def cleanup(s):
    s.cleanup_drive()
