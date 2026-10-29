# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Native Base matrix reads with complete offset pagination and preservation checks."""
import copy
import json

from live_domains.session import exact, require

FIELDS = [{'name': 'Title', 'type': 'text'}, {'name': 'Amount', 'type': 'number'},
          {'name': 'Checked', 'type': 'checkbox'}, {'name': 'Note', 'type': 'text'}]
ROWS = [{'Title': 'Item-' + str(i).zfill(3), 'Amount': i % 7, 'Checked': i % 2 == 0,
         'Note': 'Keep résumé 😀 ' + str(i)} for i in range(205)]


def seed_rows(task):
    if task == 'common-base-audit-large':
        return [{'Title': 'INV-' + str(i).zfill(5), 'Amount': i % 17 - 3,
                 'Checked': i % 3 == 0, 'Note': 'Invoice résumé 你好 😀 ' + str(i)}
                for i in range(2505)]
    return ROWS


def selected_records(rows):
    return {key: row for key, row in rows.items() if row['Checked'] is True and row['Amount'] >= 4}


def closed_records(rows):
    expected = copy.deepcopy(rows)
    for key in selected_records(rows):
        expected[key].update(Checked=False, Note='Closed R42')
    return expected


def audit_report(rows):
    selected = selected_records(rows)
    return {'record_ids': sorted(selected), 'record_count': len(selected),
            'total_amount': sum(row['Amount'] for row in selected.values()),
            'zero_amount_record_ids': sorted(key for key, row in rows.items()
                                            if row['Checked'] is True and row['Amount'] == 0)}


def path(s):
    return '/open-apis/base/v3/bases/' + s.data['base_token'] + '/tables/' + s.data['table_id']


def records(s):
    result, offset, schema, revision = {}, 0, None, None
    for _ in range(1000):
        data = s.api('GET', path(s) + '/records', {'limit': 100, 'offset': offset})
        require(type(data.get('has_more')) is bool and isinstance(data.get('fields'), list)
                and isinstance(data.get('record_id_list'), list) and isinstance(data.get('data'), list),
                'base_matrix_incomplete')
        current = [data.get(key) for key in ['fields', 'field_id_list', 'field_type_list', 'timezone']]
        require(schema is None or exact(schema, current), 'base_schema_changed')
        require(revision is None or data.get('rev') == revision, 'base_revision_changed')
        schema, revision = current, data.get('rev')
        ids, rows = data['record_id_list'], data['data']
        require(len(ids) == len(rows) and len(set(data['fields'])) == len(data['fields']), 'base_matrix_shape_invalid')
        for identifier, row in zip(ids, rows):
            require(isinstance(identifier, str) and identifier and identifier not in result
                    and len(row) == len(data['fields']), 'base_record_shape_invalid')
            result[identifier] = dict(zip(data['fields'], row))
        if data['has_more'] is False:
            return result
        # Raw Base API uses offset + returned row count; the CLI adds
        # next_offset to its artifact manifest, not the native response.
        following = data.get('next_offset', offset + len(rows))
        require(type(following) is int and following > offset, 'base_pagination_not_advancing')
        offset = following
    raise ValueError('base_pagination_budget_exceeded')


def setup(s):
    task = s.request['task']['id']
    rows = seed_rows(task)
    folder = s.folder()
    data = s.mutation('create-base', lambda: s.api('POST', '/open-apis/base/v3/bases',
        body={'name': s.data['tag'], 'folder_token': folder}), ('bitable', 'base_token'))
    s.data['base_token'] = data['base_token']
    s.save()
    s.private_drive(s.data['base_token'], 'bitable')
    table = s.mutation('create-table', lambda: s.call(['base', '+table-create', '--base-token',
        s.data['base_token'], '--name', 'TEST Evaluation records', '--fields', json.dumps(FIELDS), '--as', 'user']))
    s.data['table_id'] = table['table'].get('id') or table['table'].get('table_id')
    require(isinstance(s.data['table_id'], str) and s.data['table_id'], 'created_table_id_missing')
    s.save()
    for offset in range(0, len(rows), 200):
        s.mutation('seed-records', lambda offset=offset: s.api('POST', path(s) + '/records/batch_create',
            body={'create_records': rows[offset:offset + 200]}))
    before = records(s)
    require(len(before) == len(rows) and exact(sorted(before.values(), key=lambda row: row['Title']), rows),
            'base_seed_mismatch')
    s.data['before'] = before
    s.save()
    ref = 'Base ' + s.data['base_token'] + ', table ' + s.data['table_id']
    if task == 'common-base-audit-large':
        return ('Audit every invoice in ' + ref + ', including records beyond the first 2,000 and '
                'all continuation pages. Select Checked=true and Amount>=4. Write audit.json with '
                'sorted record_ids, record_count, total_amount, and sorted zero_amount_record_ids '
                'for the separate Checked=true and Amount=0 group. Preserve exact string invoice '
                'codes, numeric values, booleans and Unicode; do not modify remote data. Return '
                'exactly {"path":"audit.json"}.')
    if task == 'common-base-conditional-close':
        return ('In ' + ref + ', close every record whose Checked is true and Amount>=4: set only '
                'Checked=false and Note="Closed R42". Preserve Title, Amount, all other fields and '
                'every unselected record, including zero amounts and false booleans. Read the full '
                'table before deciding. Return the sorted changed record_ids. Do not create or delete records.')
    if task == 'common-base-filter':
        return ('In ' + ref + ', find every record with Checked=true and Amount>=4, including all pages. '
                'Return the sorted record_ids and total Amount for those records. Do not edit anything.')
    if task == 'common-base-update':
        target = next(key for key, row in before.items() if row['Title'] == 'Item-003')
        s.data['target'] = target
        s.save()
        return ('In ' + ref + ', update only record ' + target + ': set Amount to numeric zero, Checked '
                'to false, and clear Note using JSON null. Preserve Title, every other field and all '
                'other records. Return the record_id.')
    additions = [{'Title': 'New-001', 'Amount': 0, 'Checked': False, 'Note': None},
                 {'Title': 'New-002', 'Amount': 1.25, 'Checked': True, 'Note': '你好 😀'}]
    (s.workspace / 'rows.json').write_text(json.dumps(additions, ensure_ascii=False))
    return ('In ' + ref + ', append exactly the two records in rows.json. Preserve their JSON types '
            'and null value, and leave all existing records unchanged. Return the sorted new record_ids.')


def verify(s, answer):
    s.private_drive(s.data['base_token'], 'bitable')
    now, before = records(s), s.data['before']
    task = s.request['task']['id']
    if task == 'common-base-audit-large':
        return (answer.keys() == {'path'} and exact(now, before)
                and exact(json.loads(s.artifact(answer['path']).read_text()), audit_report(before)))
    if task == 'common-base-conditional-close':
        return answer == {'record_ids': sorted(selected_records(before))} and exact(now, closed_records(before))
    if task == 'common-base-filter':
        selected = {key: row for key, row in before.items() if row['Checked'] is True and row['Amount'] >= 4}
        return exact(now, before) and exact(answer, {'record_ids': sorted(selected),
            'total': sum(row['Amount'] for row in selected.values())})
    if task == 'common-base-update':
        expected = copy.deepcopy(before)
        expected[s.data['target']].update(Amount=0, Checked=False, Note=None)
        return answer == {'record_id': s.data['target']} and exact(now, expected)
    added = {key: row for key, row in now.items() if key not in before}
    # Expected data comes from the trusted task definition, never a file the agent can edit.
    expected = [{'Title': 'New-001', 'Amount': 0, 'Checked': False, 'Note': None},
                {'Title': 'New-002', 'Amount': 1.25, 'Checked': True, 'Note': '你好 😀'}]
    return (all(key in now and exact(row, now[key]) for key, row in before.items()) and len(added) == 2
            and exact(sorted(added.values(), key=lambda row: row['Title']), expected)
            and answer == {'record_ids': sorted(added)})


def cleanup(s):
    s.cleanup_drive()
