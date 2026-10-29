# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Native document creation, byte-preserving attachments and board imports."""
import copy
import json
import struct
import time
import uuid
import xml.etree.ElementTree as ET
import zlib

from live_domains import docs
from live_domains.session import exact, require

TABLE = ('<table><thead><tr><th><p>Code</p></th><th><p>Owner</p></th></tr></thead>'
         '<tbody><tr><td><p>001</p></td><td><p>研发 😀</p></td></tr>'
         '<tr><td><p>002</p></td><td><p>Ops</p></td></tr></tbody></table>')
MERMAID = ('sequenceDiagram\nparticipant Buyer\nparticipant Finance\nparticipant Warehouse\n'
           'Buyer->>Finance: Submit INV-001, amount 0\nalt Approved\n'
           'Finance->>Warehouse: Release 001\nWarehouse-->>Buyer: Shipped 你好\n'
           'else Rejected\nFinance-->>Buyer: Retain résumé\nend')
DRAWING = ('<svg xmlns="http://www.w3.org/2000/svg" width="640" height="200" viewBox="0 0 640 200">'
           '<circle cx="70" cy="100" r="40" fill="#dbeafe" stroke="#2563eb"/>'
           '<text x="70" y="105" text-anchor="middle" font-size="16">Intake 001</text>'
           '<polygon points="280,45 345,100 280,155 215,100" fill="#fef3c7" stroke="#d97706"/>'
           '<text x="280" y="105" text-anchor="middle" font-size="16">Approved?</text>'
           '<rect x="440" y="65" width="160" height="70" rx="16" fill="#dcfce7" stroke="#16a34a"/>'
           '<text x="520" y="105" text-anchor="middle" font-size="16">Ship 你好</text>'
           '<polyline points="110,100 215,100" fill="none" stroke="#334155" stroke-width="3"/>'
           '<polyline points="345,100 440,100" fill="none" stroke="#334155" stroke-width="3"/></svg>')
FILES = {'intake/交接.csv': 'Code,Amount,Note\r\n001,0,"résumé, 你好 😀\nline two"\r\n002,-7.5,""\r\n'.encode(),
         'archive/交接.bin': b'R42\x00\r\n' + bytes(range(256)) * 4 + b'\x00END'}


def png(width, height, variant):
    def chunk(kind, body):
        return struct.pack('>I', len(body)) + kind + body + struct.pack('>I', zlib.crc32(kind + body))
    pixels = b''.join(b'\0' + bytes(v for x in range(width) for v in
        ((x * 3 + variant) % 256, (y * 5 + variant) % 256, (x + y + variant) % 256))
        for y in range(height))
    return (b'\x89PNG\r\n\x1a\n' + chunk(b'IHDR', struct.pack('>IIBBBBB', width, height, 8, 2, 0, 0, 0))
            + chunk(b'IDAT', zlib.compress(pixels)) + chunk(b'IEND', b''))


IMAGES = {'east/image.png': png(96, 64, 10), 'west/image.png': png(64, 96, 80)}


def appended(before, after, kind, count):
    old = {b['block_id']: b for b in before}
    now = {b['block_id']: b for b in after}
    added = [b for b in after if b['block_id'] not in old]
    if (len(old) != len(before) or len(now) != len(after) or not old.keys() <= now.keys()
            or len(added) != count or any(b.get('block_type') not in
                (kind if isinstance(kind, tuple) else (kind,)) for b in added)):
        return None
    roots = [b for b in before if b.get('block_type') == 1]
    if len(roots) != 1:
        return None
    expected = copy.deepcopy(old)
    root = roots[0]['block_id']
    expected[root]['children'] = old[root].get('children', []) + [b['block_id'] for b in added
                                                               if b.get('parent_id', root) == root]
    return added if all(exact(b, now[k]) for k, b in expected.items()) else None


def setup(s):
    task = s.request['task']['id']
    if task == 'collaboration-document-create':
        s.folder()
        s.data['creation_started_at'] = int(time.time())
        s.save()
        (s.workspace / 'handoff.xml').write_text('<title>' + s.data['tag'] + '</title>' + docs.CONTENT + TABLE)
        return ('Create exactly one new private Feishu document from the complete handoff.xml, '
                'preserving its title, native heading, bold hyperlink, Unicode, leading-zero codes '
                'and 3-by-2 table. Use user identity. The private staging parent is ' + s.data['folder']
                + '. Do not invite/share/send anything. Do not repeat an uncertain creation. '
                'Immediately save the returned document_id in creation.json, then return {"document_id":"..."}. '
                'This task checks the document itself, not destination-folder placement.')
    docs.setup(s)
    token = s.data['document_id']
    if task in ['collaboration-image-upload', 'collaboration-file-upload']:
        assets = IMAGES if task == 'collaboration-image-upload' else FILES
        for name, content in assets.items():
            path = s.workspace / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(content)
        detail = ('two images east/image.png then west/image.png; they deliberately share a basename '
                  'but have different pixels. Display the first at 192-by-128 and the second at 128-by-192 pixels'
                  if task == 'collaboration-image-upload' else
                  'two files intake/交接.csv then archive/交接.bin as native file attachments, with their original filenames')
        return ('Append the ' + detail + ' to native document ' + token + '. Preserve the exact uploaded '
                'bytes and order, and all existing native block IDs, text, links and formatting. '
                'Do not create other documents, replace existing content, add captions or share/send anything. '
                'Return {"document_id":"' + token + '"}.')
    source, name, kind = ((MERMAID, 'approval.mmd', 'Mermaid') if task == 'collaboration-mermaid-create'
                          else (DRAWING, 'handoff.svg', 'SVG drawing'))
    (s.workspace / name).write_text(source)
    preservation = ('Preserve the Mermaid source exactly, including Unicode and branches. '
                    if task == 'collaboration-mermaid-create' else
                    'Preserve the drawing labels, colors, shape geometry and connector endpoints. '
                    'Native editable shapes/text and a vector fallback for the diamond are allowed. ')
    return ('Append exactly one populated native whiteboard to document ' + token + ' from the complete '
            + kind + ' source in ' + name + '. ' + preservation
            + 'It must render in Feishu as a diagram, not a code fence, '
            'plain text or image attachment. Preserve all existing document blocks, IDs, styles and links. '
            'Do not share, invite, send messages or create another document. '
            'Return {"document_id":"' + token + '"}.')


def created_document(s):
    token = None
    for name in ['creation.json', 'answer.json']:
        path = s.workspace / name
        if path.exists():
            data = json.loads(s.artifact(name).read_text())
            token = data.get('document_id')
            require(isinstance(token, str) and token.isalnum() and 10 <= len(token) <= 100,
                    'invalid_created_document_id')
            break  # The creation journal owns cleanup even if the final answer is wrong.
    require(token is not None, 'created_document_requires_reconciliation')
    s.private_drive(token, 'docx')
    meta = s.api('POST', '/open-apis/drive/v1/metas/batch_query', body={
        'request_docs': [{'doc_token': token, 'doc_type': 'docx'}], 'with_url': False})['metas'][0]
    created_at = int(meta['create_time'])
    if created_at >= 10 ** 12:
        created_at //= 1000
    require(meta.get('title') == s.data['tag']
            and s.data['creation_started_at'] <= created_at <= int(time.time()) + 60,
            'created_document_not_run_owned')
    if not any(r['id'] == token for r in s.data['owned']):
        s.data['owned'].append({'id': token, 'kind': 'docx', 'deleted': False,
                                'source': 'verified_unique_run_title_and_creation_time'})
        s.data['document_id'] = token
        s.save()
    return token


def valid_creation(native, title):
    roots = [b for b in native if b.get('block_type') == 1]
    headings = [docs.text(b) for b in native if b.get('block_type') == 3]
    texts = [docs.text(b) for b in native if b.get('block_type') == 2]
    tables = [b for b in native if b.get('block_type') == 31]
    return (len(roots) == 1 and docs.text(roots[0]) == title and headings == ['Release notes']
            and texts == ['Milestone: 2027-03-18', 'Old status — 你好', 'Keep résumé 😀 and 001 unchanged.',
                          'Code', 'Owner', '001', '研发 😀', '002', 'Ops']
            and sum(docs.styled_status(b, 'Old') for b in native) == 1
            and len(tables) == 1 and tables[0]['table'].get('property', {}).get('row_size') == 3
            and tables[0]['table'].get('property', {}).get('column_size') == 2)


def download(s, token, suffix):
    name = 'readback-' + uuid.uuid4().hex + suffix
    s.call(['docs', '+media-download', '--token', token, '--output', name, '--as', 'user'])
    path = s.state / name
    require(path.is_file(), 'native_download_missing')
    return path.read_bytes()


def board_nodes(s, token):
    data = s.api('GET', '/open-apis/board/v1/whiteboards/' + token + '/nodes')
    require(isinstance(data.get('nodes'), list) and data['nodes'], 'native_board_empty')
    require(all(data.get(key, False) is False for key in ['has_more', 'truncated', 'is_truncated']),
            'native_board_read_incomplete')
    return data['nodes']


def valid_board(nodes, source, syntax):
    matches = [n for n in nodes if n.get('syntax', {}).get('code') == source
               and n['syntax'].get('syntax_type') == syntax]
    return len(matches) == 1


def valid_drawing(nodes):
    shapes = [n for n in nodes if n.get('type') == 'composite_shape']
    texts = [n for n in nodes if n.get('type') == 'text_shape']
    lines = [n for n in nodes if n.get('type') == 'connector']
    diamonds = [n for n in nodes if n.get('type') == 'image']
    if len(nodes) != 8 or [len(v) for v in [shapes, texts, lines, diamonds]] != [2, 3, 2, 1]:
        return False
    if any(n.get('angle') != 0 for n in nodes):
        return False
    expected_shapes = {('ellipse', 30, 60, 80, 80, '#dbeafe', '#2563eb'),
                       ('round_rect', 440, 65, 160, 70, '#dcfce7', '#16a34a')}
    actual_shapes = {(n['composite_shape']['type'], n['x'], n['y'], n['width'], n['height'],
                      n['style']['fill_color'].lower(), n['style']['border_color'].lower()) for n in shapes}
    if actual_shapes != expected_shapes:
        return False
    rounded = next(n for n in shapes if n['composite_shape']['type'] == 'round_rect')
    if rounded['style'].get('border_radius') != dict.fromkeys(
            ['top_left', 'top_right', 'bottom_left', 'bottom_right'], 16):
        return False
    labels = {n.get('text', {}).get('text'): n for n in texts}
    if set(labels) != {'Intake 001', 'Approved?', 'Ship 你好'}:
        return False
    for label, x in [('Intake 001', 70), ('Approved?', 280), ('Ship 你好', 520)]:
        n = labels[label]
        if (abs(n['x'] + n['width'] / 2 - x) > 1 or abs(n['y'] + n['height'] / 2 - 100) > 15
                or n['text'].get('font_size') != 16 or n['text'].get('text_color', '').lower() != '#000000'):
            return False
    endpoints = set()
    for n in lines:
        c = n['connector']
        endpoints.add(tuple(c[end]['position'][axis] for end in ['start', 'end'] for axis in ['x', 'y']))
        if n['style']['border_color'].lower() != '#334155':
            return False
    if endpoints != {(110, 100, 215, 100), (345, 100, 440, 100)}:
        return False
    try:
        vector = ET.fromstring(diamonds[0]['image']['token'])
    except ET.ParseError:
        return False
    parts = list(vector)
    return (len(parts) == 1 and parts[0].tag == '{http://www.w3.org/2000/svg}polygon'
            and parts[0].get('points') == '280,45 345,100 280,155 215,100'
            and parts[0].get('fill', '').lower() == '#fef3c7'
            and parts[0].get('stroke', '').lower() == '#d97706')


def verify(s, answer):
    task = s.request['task']['id']
    if task == 'collaboration-document-create':
        token = created_document(s)
        return answer == {'document_id': token} and valid_creation(docs.blocks(s, token), s.data['tag'])
    token = s.data['document_id']
    s.private_drive(token, 'docx')
    if answer != {'document_id': token}:
        return False
    native = docs.blocks(s, token)
    if task in ['collaboration-image-upload', 'collaboration-file-upload']:
        image = task == 'collaboration-image-upload'
        blocks = appended(s.data['before'], native, 27 if image else (23, 33), 2 if image else 4)
        if blocks is None:
            return False
        if not image:
            views = [b for b in blocks if b['block_type'] == 33]
            blocks = [b for b in blocks if b['block_type'] == 23]
            if len(views) != 2 or len(blocks) != 2 or any(v.get('children') != [b['block_id']]
                    or b.get('parent_id') != v['block_id'] for v, b in zip(views, blocks)):
                return False
        assets = IMAGES if image else FILES
        tokens = []
        for i, (block, (name, expected)) in enumerate(zip(blocks, assets.items())):
            media = block.get('image' if image else 'file', {})
            resource = media.get('token')
            if not isinstance(resource, str) or not resource:
                return False
            tokens.append(resource)
            if image:
                size = (192, 128) if i == 0 else (128, 192)
                if (media.get('width'), media.get('height')) != size:
                    return False
            elif media.get('name') != name.split('/')[-1]:
                return False
            if download(s, resource, '.png' if image else '.bin') != expected:
                return False
        return len(set(tokens)) == 2
    blocks = appended(s.data['before'], native, 43, 1)
    if blocks is None:
        return False
    board = blocks[0].get('board', {}).get('token')
    if not isinstance(board, str) or not board:
        return False
    nodes = board_nodes(s, board)
    if task == 'collaboration-mermaid-create':
        return (valid_board(nodes, MERMAID, 2) and sum(n.get('type') == 'life_line' for n in nodes) == 3
                and sum(n.get('type') == 'connector' for n in nodes) == 4
                and sum(n.get('type') == 'combined_fragment' for n in nodes) == 1)
    return valid_drawing(nodes)


def cleanup(s):
    if s.request['task']['id'] == 'collaboration-document-create' and not s.data.get('document_id'):
        created_document(s)
    s.cleanup_drive()
