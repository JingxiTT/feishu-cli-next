"""Versioned harder Docs cohort; bounded native hierarchy, no live resources.

Native table/cell shapes reuse paired.markdown's live-checked contract. List
fields follow oapi-sdk-go v3.7.2 docx/v1 Block/Text/TextStyle; XML list tags follow
docs_fetch_im_markdown_test.go. These are synthetic projections, not fresh live
rendering observations. Existing native inline edit semantics are reused intact.
"""
import copy
from html import escape
import json
from urllib.parse import urlsplit
from xml.etree import ElementTree

from paired import docs_edit
from scenarios import docs as native

DOCUMENT = native.DOCUMENT
CASES = ('docs-hard-rich-read', 'docs-hard-focused-read', 'docs-hard-multi-edit')


def text_block(identifier, text, kind='paragraph', parent=DOCUMENT):
    return {'block_id': identifier, 'parent_id': parent, 'kind': kind, 'text': text}


def rich_blocks():
    blocks = [text_block('hardHeading', 'Release checklist', 'title')]
    rows = [['SKU', 'Label', 'Note'], ['001', '你好', ''], ['010', 'résumé', 'hold']]
    cells = [f'hardCell{r}{c}' for r in range(3) for c in range(3)]
    blocks.append(dict(text_block('hardTable', '', 'table'), children=cells, row_size=3, column_size=3))
    for r, row in enumerate(rows):
        for c, value in enumerate(row):
            cell, paragraph = f'hardCell{r}{c}', f'hardValue{r}{c}'
            blocks.append(dict(text_block(cell, '', 'cell', 'hardTable'), children=[paragraph]))
            blocks.append(text_block(paragraph, value, parent=cell))
    blocks += [dict(text_block('hardPrepare', 'Prepare', 'ordered'), sequence='1', children=['hardSeal', 'hardLabel']),
               text_block('hardSeal', 'Inspect seal', 'bullet', 'hardPrepare'),
               text_block('hardLabel', 'Read label', 'bullet', 'hardPrepare'),
               dict(text_block('hardRelease', 'Release', 'ordered'), sequence='2'),
               copy.deepcopy(native.INITIAL['blocks'][3])]
    return blocks


def focused_blocks():
    # Distinct operational records, not repeated padding or a forced page-count target.
    projects = [
        ('Atlas', 'AT-0412', 'silver', 'carton sealer', 'closure force', 'North packaging'),
        ('Lyra', 'LY-0520', 'violet', 'dry-box sensor', 'humidity drift', 'Sensor laboratory'),
        ('Vega', 'VE-0603', 'copper', 'spool adapter', 'bore clearance', 'Tooling workshop'),
        ('Cygnus', 'CY-0618', 'teal', 'feed roller', 'traction loss', 'Motion laboratory'),
        ('Draco', 'DR-0629', 'indigo', 'heater harness', 'connector retention', 'Electrical laboratory'),
        ('Phoenix', 'PH-0702', 'bronze', 'air filter', 'pressure drop', 'Airflow laboratory'),
        ('Auriga', 'AU-0708', 'jade', 'tension arm', 'spring preload', 'Assembly cell'),
        ('Hydra', 'HY-0714', 'scarlet', 'water tray', 'leak rate', 'Molding workshop'),
        ('Perseus', 'PE-0719', 'ivory', 'display cover', 'scratch resistance', 'Optics laboratory'),
        ('Lynx', 'LX-0723', 'ochre', 'fan bracket', 'resonance', 'Acoustics laboratory'),
        ('Orion archive', 'OR-0621', 'amber', 'first-release enclosure', 'hinge fatigue', 'Pilot line'),
        ('Orion', 'OR-0731', 'cobalt', 'revised enclosure', 'latch alignment', 'Final assembly'),
    ]
    blocks = []
    for section, (name, code, lane, part, metric, team) in enumerate(projects):
        title = 'Launch gate Orion' if section == 11 else f'Archived launch gate {name}'
        blocks.append(text_block(f'hardSection{section:02}', title, 'title'))
        facts = [
            f'{name} scope: qualify the {part}. {team} owns the release record and the {metric} acceptance check.',
            f'{name} sample tracking: lot {section + 1:03} is reserved for inspection; the leading zeros identify a physical lot, not a quantity.',
            f'{name} inspection: measure {metric} before assembly, after conditioning, and after the shipping simulation. Record each result separately.',
            f'{name} supplier disposition: replacement {part} samples require a new inspection sheet; prior approval does not transfer to a changed drawing.',
            f'{name} tooling: retain the signed setup photograph and cavity trace with the {part} drawing. {team} reviews any fixture adjustment.',
            f'{name} assembly: stop work if the {metric} check fails. Place suspect units in the review rack with their original lot label attached.',
            f'{name} packaging: include the handling insert for the {part}; bilingual labels must retain résumé and 你好 exactly in the print proof.',
            f'{name} release ownership: the {lane} lane coordinator schedules the pilot only after {team} has signed the inspection summary.',
            f'{name} rollback: retain the prior {part} revision and setup sheet until field review closes; switching back requires an explicit lot record.',
            f'{name} evidence index: dimensional report, {metric} results, assembly photographs, and shipment checklist are filed under the project name.',
            f'{name} exception policy: a missing inspection result cannot be replaced by another project approval code. Escalate to {team}.',
            f'{name} handoff: the next shift must reconcile accepted and quarantined {part} quantities before starting a new sample lot.',
            f'{name} record status: ' + ('current release decision; supersedes the earlier Orion archive.' if section == 11 else 'archived decision; retain for traceability, not current Orion release authorization.'),
            f'{name} final approval code: {code}; release lane: {lane}. ' + ('This is the current Launch gate Orion decision.' if section == 11 else 'Applies only to this archived project record.'),
        ]
        blocks.extend(text_block(f'hardFact{section:02}{row:02}', fact) for row, fact in enumerate(facts))
    return blocks


def edit_blocks():
    blocks = rich_blocks()
    first = text_block('hardEditA', 'Draft status; amber flag — 你好')
    first['runs'] = native.runs_from_characters([(c, bold, None) for text, bold in
        [('Draft status; ', False), ('amber', True), (' flag — 你好', False)] for c in text], 0)
    second = text_block('hardEditB', 'Open legacy portal for résumé 😀.')
    second['runs'] = native.runs_from_characters([(c, False, link) for text, link in
        [('Open ', None), ('legacy portal', 'https://example.com/reference'), (' for résumé 😀.', None)] for c in text], 0)
    return blocks[:1] + [first, second] + blocks[1:]


def wire(block):
    kind = block['kind']
    if kind == 'page':
        return {'block_id': block['block_id'], 'block_type': 1, 'children': block['children'],
                'page': {'elements': [{'text_run': {'content': block['text'], 'text_element_style': {}}}]}}
    if kind in ('table', 'cell'):
        value = {key: copy.deepcopy(block[key]) for key in ('block_id', 'parent_id', 'children')}
        if kind == 'cell':
            return dict(value, block_type=32, table_cell={})
        return dict(value, block_type=31, table={'cells': block['children'],
            'property': {'row_size': block['row_size'], 'column_size': block['column_size']}})
    value = native.wire_block(block)
    if kind in ('bullet', 'ordered'):
        value[kind] = value.pop('text')
        value['block_type'] = 12 if kind == 'bullet' else 13
        if kind == 'ordered':
            value[kind]['style'] = {'sequence': block['sequence']}
        if block.get('children'):
            value['children'] = block['children']
    return value


def render(blocks, format_='xml', ids=True):
    index = {block['block_id']: block for block in blocks}

    def one(block, depth=0):
        kind = block['kind']
        attr = ' block-id="' + escape(block['block_id'], quote=True) + '"' if ids else ''
        children = [index[key] for key in block.get('children', []) if key in index]
        if kind == 'page':
            return siblings(children)
        if kind == 'table':
            rows = [children[n:n + block['column_size']] for n in range(0, len(children), block['column_size'])]
            if format_ == 'markdown':
                lines = ['| ' + ' | '.join(one(cell).strip() for cell in row) + ' |' for row in rows]
                lines.insert(1, '| ' + ' | '.join('---' for _ in range(block['column_size'])) + ' |')
                return '\n'.join(lines)
            return '<table' + attr + '>' + ''.join('<tr>' + ''.join(one(cell) for cell in row) + '</tr>' for row in rows) + '</table>'
        if kind == 'cell':
            content = ''.join(one(child) for child in children)
            return '<td' + attr + '>' + content + '</td>' if format_ == 'xml' else content
        if kind in ('ordered', 'bullet'):
            if format_ == 'markdown':
                prefix = block.get('sequence', '') + '. ' if kind == 'ordered' else '- '
                return '  ' * depth + prefix + native.text_of(block) + ''.join('\n' + one(child, depth + 1) for child in children)
            sequence = ' seq="' + block['sequence'] + '"' if kind == 'ordered' else ''
            return '<li' + attr + sequence + '>' + escape(native.text_of(block)) + (
                '<ul>' + ''.join(one(child) for child in children) + '</ul>' if children else '') + '</li>'
        return native.xml_block(block, ids) if format_ == 'xml' else native.markdown_block(block)

    def siblings(children):
        parts = []
        for block in children:
            content = one(block)
            if format_ == 'xml' and block['kind'] in ('ordered', 'bullet'):
                tag = 'ol' if block['kind'] == 'ordered' else 'ul'
                content = '<' + tag + '>' + content + '</' + tag + '>'
            parts.append(content)
        return ('' if format_ == 'xml' else '\n\n').join(parts)

    return siblings([block for block in blocks if block['parent_id'] not in index])


def initial(case):
    state = copy.deepcopy(native.INITIAL)
    state['blocks'] = {'docs-hard-rich-read': rich_blocks, 'docs-hard-focused-read': focused_blocks,
                       'docs-hard-multi-edit': edit_blocks}[case]()
    state['title'] = 'Harder synthetic Docs v1'
    children = [b['block_id'] for b in state['blocks'] if b['parent_id'] == DOCUMENT]
    state['blocks'].insert(0, dict(text_block(DOCUMENT, state['title'], 'page', ''), children=children))
    state['block_order'] = [b['block_id'] for b in state['blocks']]
    state['full_xml'] = render(state['blocks'])
    return state


ANSWERS = {
    CASES[0]: {'sku': '001', 'label': '你好', 'note': '', 'table_block_id': 'hardTable',
               'ordered_steps': ['Prepare', 'Release'], 'prepare_children': ['Inspect seal', 'Read label'],
               'prepare_block_id': 'hardPrepare'},
    CASES[1]: {'approval_code': 'OR-0731', 'release_lane': 'cobalt', 'block_id': 'hardFact1113'},
    CASES[2]: {'document_id': DOCUMENT, 'paragraphs': ['Final status; green flag — 你好', 'Open current portal for résumé 😀.']},
}
PROMPTS = {
    CASES[0]: f'Read document {DOCUMENT}. In its SKU table, return the row whose SKU is 001: sku, label and note, preserving an empty cell as an empty string. Also return table_block_id, the ordered top-level checklist steps as ordered_steps, the nested children of Prepare in their original order as prepare_children, and prepare_block_id. Return JSON; do not modify the document.',
    CASES[1]: f'Read document {DOCUMENT}. Find the final approval code and release lane specifically in the Launch gate Orion section, ignoring archived sections. Return JSON with approval_code, release_lane and the native block_id containing both facts. Full or focused reads are valid. Do not modify the document.',
    CASES[2]: f'In document {DOCUMENT}, make exactly three wording changes: Draft status to Final status; amber to green; legacy portal to current portal. The first two occur in one paragraph. Preserve all remaining text, native bold formatting, hyperlink URLs, Unicode, table/list/media content and hierarchy, permissions and unrelated block IDs. Return JSON with document_id and paragraphs: the complete plain text of the two edited paragraphs in document order. Do not create, share or delete resources.',
}


def task(case):
    answer = ANSWERS[case]
    return {'id': case, 'section': 'docs', 'prompt': PROMPTS[case],
            'fixture': {'scenario': 'docs-hard-v1', 'scenario_section': 'docs-hard-v1', 'case_id': case,
                        'allow_synthetic_mutations': case == CASES[2],
                        'answer_schema': {'type': 'object', 'additionalProperties': False, 'required': list(answer),
                            'properties': {key: ({'type': 'array', 'items': {'type': 'string'}} if isinstance(value, list)
                                                 else {'type': 'string'}) for key, value in answer.items()}}},
            'expected': {'answer': answer}}


class Scenario(docs_edit.Scenario):
    def __init__(self, case):
        super().__init__()
        self.case = case
        self.state = initial(case)
        self.observations = []

    def snapshot(self):
        with self.lock:
            return copy.deepcopy({'initial': initial(self.case), 'final': self.state,
                                  'mutations': self.mutations, 'observations': self.observations})

    def _reply(self, route, data, proof=None):
        index = {block['block_id']: block for block in self.state['blocks']}
        if route == 'docs_blocks':
            data['items'] = [wire(index[item['block_id']]) for item in data['items']]
        elif route == 'docs_block_get':
            data['block'] = wire(index[data['block']['block_id']])
        return super()._reply(route, data, proof)

    def _fetch(self, data):
        reply = super()._fetch(data)  # Preserve native request validation/scopes.
        blocks, mode = self._select(data.get('read_option', {}))
        selected_ids = {b['block_id'] for b in blocks}
        if any(set(b.get('children', [])) - selected_ids for b in blocks):
            raise ValueError('unsupported strategy: native scoped Markdown-document fetch unobserved')
        format_ = data.get('format', 'xml')
        content = render(blocks, format_, data.get('export_option', {}).get('export_block_id', False))
        if format_ == 'xml' and mode != 'full':
            content = '<fragment mode="' + mode + '">' + content + '</fragment>'
        reply.json['data']['document']['content'] = content
        return reply

    def _commit(self, candidate, route):
        if native.canonical(candidate) == native.canonical(self.state):
            return {'mutation': False, 'document_id': DOCUMENT}
        before = copy.deepcopy(self.state)
        renamed = {old['block_id']: new['block_id'] for old, new in zip(before['blocks'], candidate['blocks'])
                   if old['block_id'] != new['block_id']}
        for block in candidate['blocks']:
            if 'children' in block:
                block['children'] = [renamed.get(identifier, identifier) for identifier in block['children']]
        candidate['revision_id'] += 1
        candidate['block_order'] = [b['block_id'] for b in candidate['blocks']]
        candidate['full_xml'] = render(candidate['blocks'])
        event = {'before': before, 'after': copy.deepcopy(candidate), 'before_hash': native.digest(before),
                 'after_hash': native.digest(candidate), 'changed_block_ids': native.changed_block_ids(before, candidate)}
        self.state = candidate
        self.mutations.append(event)
        return {'mutation': True, 'document_id': DOCUMENT, **{k: event[k] for k in ('before_hash', 'after_hash', 'changed_block_ids')}}

    def dispatch(self, method, uri, body_bytes, headers):
        if self.case != CASES[2] and method != 'GET' and not (method == 'POST' and urlsplit(uri).path == native.AI_PATH + '/fetch'):
            raise ValueError('read-only task rejects document mutations')
        with self.lock:
            reply = super().dispatch(method, uri, body_bytes, headers)
            observed = []
            data = reply.json.get('data', {})
            if reply.route in ('docs_blocks', 'docs_block_get'):
                returned = data.get('items', []) if reply.route == 'docs_blocks' else [data.get('block')]
                observed = [b['block_id'] for b in self.state['blocks'] if wire(b) in returned]
            elif reply.route == 'docs_fetch':
                # Only actual XML nodes carry evidence IDs; child references and
                # empty-string membership do not prove that a block was returned.
                content = data['document']['content']
                if json.loads(body_bytes or b'{}').get('format', 'xml') == 'xml':
                    nodes = ElementTree.fromstring('<root>' + content + '</root>')
                    observed = [node.attrib['block-id'] for node in nodes.iter() if 'block-id' in node.attrib]
            proof = {**reply.proof, 'response_sha256': native.digest(reply.json),
                     'observed_block_ids': observed}
            reply.proof = proof
            self.observations.append(copy.deepcopy(proof))
            return reply


def preserved(state):
    value = copy.deepcopy(state)
    identifiers = [block['block_id'] for block in value['blocks']]
    if (any(not isinstance(i, str) or not i for i in identifiers) or len(set(identifiers)) != len(identifiers) or
            identifiers != value['block_order']):
        raise ValueError('invalid block order or identities')
    value.pop('revision_id')
    value.pop('full_xml')
    for n in (2, 3):
        block = value['blocks'][n]
        native.native_characters(block)
        old_id = block['block_id']
        block['block_id'] = ('hardEditA', 'hardEditB')[n - 2]
        value['blocks'][0]['children'] = [block['block_id'] if i == old_id else i for i in value['blocks'][0]['children']]
        value['block_order'][n] = block['block_id']
        block.pop('text')
        block.pop('runs', None)
    return value


def desired_edit_state():
    state = initial(CASES[2])
    for n, replacements in [(2, [('Draft', 'Final'), ('amber', 'green')]), (3, [('legacy', 'current')])]:
        block = state['blocks'][n]
        for old, new in replacements:
            for run in block['runs']:
                run['text'] = run['text'].replace(old, new)
        block['text'] = native.text_of(block)
    return state


def known_failure_diagnostics(case, answer, receipt):
    """Failures independently proved despite an unrelated unmodeled request.

    Missing completion/read proof is not a known failure: an unsupported native
    strategy may have supplied it. Only answer contradictions and authenticated
    mutations violating preservation are definitive here.
    """
    reasons = []
    if native.canonical(answer) != native.canonical(ANSWERS[case]):
        reasons.append('paired_docs_answer_mismatch')
    if case != CASES[2]:
        return reasons
    try:
        start = initial(case)
        if receipt['state']['initial'] != start:
            return reasons
        outside = preserved(start)
        desired = desired_edit_state()
        for event in receipt['state']['mutations']:
            if (native.digest(event['before']) != event['before_hash'] or
                    native.digest(event['after']) != event['after_hash']):
                continue
            authenticated = any(r.get('status') == 200 and r.get('signature_valid') is True and
                r.get('route') in ('docs_update', 'docs_block_patch') and r.get('proof', {}).get('mutation') is True and
                all(r['proof'].get(k) == event[k] for k in ('before_hash', 'after_hash', 'changed_block_ids'))
                for r in receipt['requests'])
            if not authenticated:
                continue
            after = event['after']
            if preserved(after) != outside:
                reasons.append('paired_docs_unrelated_state_changed')
            # Target styles have the same final-state gate as verify(); an
            # intermediate target edit that is repaired is not a new failure.
            if after != receipt['state']['final']:
                continue
            for n in (2, 3):
                actual, target = after['blocks'][n], desired['blocks'][n]
                if native.text_of(actual) == native.text_of(target):
                    actual_chars, target_chars = native.native_characters(actual), native.native_characters(target)
                    if [c[1] for c in actual_chars] != [c[1] for c in target_chars]:
                        reasons.append('paired_docs_bold_mismatch')
                    if [c[2] for c in actual_chars] != [c[2] for c in target_chars]:
                        reasons.append('paired_docs_link_mismatch')
    except (KeyError, TypeError, ValueError, IndexError, AttributeError):
        pass  # Incomplete proof remains inconclusive; do not invent a violation.
    return sorted(set(reasons))


def verify(case, workspace, answer, receipt, expected):
    del workspace, expected
    try:
        if native.canonical(answer) != native.canonical(ANSWERS[case]) or receipt['scenario'] != 'docs-hard-v1':
            return False
        state = receipt['state']
        start = initial(case)
        if state['initial'] != start:
            return False
        requests = receipt['requests']
        if any(r.get('status') != 200 or r.get('signature_valid') is not True for r in requests):
            return False
        if [r.get('proof') for r in requests] != state['observations']:
            return False
        if case != CASES[2]:
            required = ({'hardTable', 'hardPrepare', 'hardRelease', 'hardSeal', 'hardLabel'} |
                        {f'hard{kind}{row}{col}' for kind in ('Cell', 'Value') for row in (0, 1) for col in range(3)}) if case == CASES[0] else {'hardFact1113'}
            observed = {identifier for o in state['observations'] for identifier in o['observed_block_ids']}
            return state['final'] == start and not state['mutations'] and required <= observed
        if not state['mutations']:
            return False
        previous = start
        outside = preserved(start)
        for event in state['mutations']:
            before, after = event['before'], event['after']
            if (before != previous or native.digest(before) != event['before_hash'] or native.digest(after) != event['after_hash'] or
                    type(after['revision_id']) is not int or after['revision_id'] != before['revision_id'] + 1 or
                    preserved(after) != outside or after['full_xml'] != render(after['blocks']) or
                    event['changed_block_ids'] != native.changed_block_ids(before, after) or not event['changed_block_ids']):
                return False
            if not any(r.get('route') in ('docs_update', 'docs_block_patch') and r['proof'].get('mutation') is True and
                       all(r['proof'].get(k) == event[k] for k in ('before_hash', 'after_hash', 'changed_block_ids')) for r in requests):
                return False
            previous = after
        if state['final'] != previous:
            return False
        desired = desired_edit_state()
        for n in (2, 3):
            if native.native_characters(state['final']['blocks'][n]) != native.native_characters(desired['blocks'][n]):
                return False
        return True
    except (KeyError, TypeError, ValueError, IndexError, AttributeError, RecursionError):
        return False


def verification_diagnostics(case, workspace, answer, receipt, expected):
    reasons = []
    if native.canonical(answer) != native.canonical(ANSWERS[case]):
        reasons.append('paired_docs_answer_mismatch')
    if not verify(case, workspace, ANSWERS[case], receipt, expected):
        reasons.append('paired_docs_proof_mismatch')
    return reasons
