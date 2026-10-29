# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Verify native raw values separately from formula rendering."""
import copy
import csv
import json

from live_domains.session import exact, require

EXTENT = 'A1:F6'
VALUES = [['Code', 'Amount', 'Approved', 'Note', 'Formula', 'Keep'],
          ['001', 2, True, '你好 😀', '', 'guard-1'],
          ['002', 0, False, '', '', 'guard-2'],
          ['003', 3, True, 'résumé', '', 'guard-3'],
          ['004', 5, False, 'line\nbreak', '', 'guard-4'],
          ['GUARD', 0, False, 'Do not edit', '', 'end']]
INTAKE = [{'Code': '003', 'Amount': '7.5', 'Note': 'Approved, résumé\n你好 😀'},
          {'Code': '001', 'Amount': '0', 'Note': ''},
          {'Code': '004', 'Amount': '99', 'Note': 'Do not apply'},
          {'Code': '002', 'Amount': '99', 'Note': 'Do not apply'}]
INTAKE_CHANGES = {(1, 1): 0, (1, 3): '', (3, 1): 7.5, (3, 3): INTAKE[0]['Note']}


def expected_snapshot(before, changes):
    expected = copy.deepcopy(before)
    for (row, col), value in changes.items():
        expected['raw'][row][col]['value'] = value
        rendered = expected['formula'][row][col]
        if value == '':
            rendered.pop('value', None)
        elif type(value) in (int, float):
            rendered['value'] = str(int(value)) if value == int(value) else str(value)
        else:
            rendered['value'] = value
        if col == 1 and 1 <= row <= 4:
            expected['raw'][row][4]['value'] = value * 2
            expected['formula'][row][4]['value'] = value * 2
    return expected


def tool(s, name, inputs):
    data = s.api('POST', '/open-apis/sheet_ai/v2/spreadsheets/' + s.data['spreadsheet_token']
                 + '/tools/invoke_read', body={'tool_name': name, 'input': json.dumps(inputs)})
    require(isinstance(data.get('output'), str) and data['output'], 'native_sheet_output_missing')
    return json.loads(data['output'])


def grid(s, render):
    data = tool(s, 'get_cell_ranges', {'excel_id': s.data['spreadsheet_token'],
        'sheet_id': s.data['sheet_id'], 'ranges': [EXTENT], 'value_render_option': render,
        'include_styles': True, 'include_truncation_info': True, 'cell_limit': 1000, 'max_chars': 100000})
    require(len(data.get('ranges', [])) == 1, 'sheet_range_count_mismatch')
    block = data['ranges'][0]
    require(block.get('range') == EXTENT, 'sheet_range_mismatch')
    for envelope in [data, block]:
        for key in ['has_more', 'truncated', 'is_truncated']:
            require(key not in envelope or envelope[key] is False, 'sheet_read_truncated')
    cells = block.get('cells')
    require(isinstance(cells, list) and len(cells) == 6
            and all(isinstance(row, list) and len(row) == 6 for row in cells), 'sheet_extent_incomplete')
    require(all(isinstance(c, dict) for row in cells for c in row), 'invalid_native_cell')
    return cells


def snapshot(s):
    raw = grid(s, 'raw_value')
    formulas = grid(s, 'formula')
    require(exact(raw, grid(s, 'raw_value')), 'sheet_changed_during_read')
    return {'raw': raw, 'formula': formulas}


def setup(s):
    folder = s.folder()
    result = s.mutation('create-workbook', lambda: s.api('POST', '/open-apis/sheets/v3/spreadsheets',
        body={'title': s.data['tag'], 'folder_token': folder}), ('sheet', 'spreadsheet.spreadsheet_token'))
    token = result['spreadsheet']['spreadsheet_token']
    s.data['spreadsheet_token'] = token
    s.save()
    s.private_drive(token, 'sheet')
    structure = tool(s, 'get_workbook_structure', {'excel_id': token})
    require(len(structure.get('sheets', [])) == 1, 'workbook_seed_shape_invalid')
    s.data['sheet_id'] = structure['sheets'][0]['sheet_id']
    cells = [[{'value': value} for value in row] for row in VALUES]
    for row in range(1, 5):
        cells[row][0]['cell_styles'] = {'number_format': '@'}
        cells[row][4] = {'formula': '=B' + str(row + 1) + '*2'}
    s.mutation('seed-cells', lambda: s.call(['sheets', '+cells-set', '--spreadsheet-token', token,
        '--sheet-id', s.data['sheet_id'], '--range', EXTENT, '--cells', json.dumps(cells), '--as', 'user']))
    if s.request['task']['id'] == 'common-sheets-intake':
        # Fit the longer incoming note before the preservation snapshot; native
        # clipping diagnostics must remain comparable without ignoring fields.
        s.mutation('seed-note-column-width', lambda: s.call(['sheets', '+cols-resize',
            '--spreadsheet-token', token, '--sheet-id', s.data['sheet_id'],
            '--range', 'D', '--width', '600', '--as', 'user']))
    s.data['before'] = snapshot(s)
    s.save()
    raw = s.data['before']['raw']
    require(all(exact(raw[r][c].get('value'), VALUES[r][c]) for r in range(6) for c in range(6)
                if c != 4), 'native_sheet_seed_mismatch')
    for row in range(1, 5):
        formula = '=B' + str(row + 1) + '*2'
        rendered = s.data['before']['formula'][row][4]
        require((rendered.get('formula') == formula or rendered.get('value') == formula)
                and exact(raw[row][4].get('value'), VALUES[row][1] * 2), 'native_formula_seed_mismatch')
    reference = 'workbook ' + token + ', sheet ' + s.data['sheet_id'] + ', range ' + EXTENT
    task = s.request['task']['id']
    if task == 'common-sheets-intake':
        with (s.workspace / 'intake.csv').open('w', newline='', encoding='utf-8') as stream:
            writer = csv.DictWriter(stream, fieldnames=['Code', 'Amount', 'Note'])
            writer.writeheader()
            writer.writerows(INTAKE)
        return ('Reconcile intake.csv into ' + reference + ' by exact string Code, preserving leading '
                'zeros. Apply Amount as a number and Note as text only when the existing Approved '
                'cell is boolean true. Skip rows whose existing Approved value is false. Preserve '
                'all codes, approval flags, styles, formulas, headers and GUARD cells. Empty CSV Note '
                'means an empty string, and quoted commas/newlines/Unicode must survive. Formula '
                'results may recalculate. Return sorted updated_codes and skipped_codes for the CSV rows.')
    if task == 'common-sheets-analyze':
        return ('Read ' + reference + '. Sum Amount for rows whose Approved value is true. '
                'Return approved_total and the number of data rows; exclude the header and GUARD row. Do not edit.')
    if task == 'common-sheets-update':
        return ('In ' + reference + ', change only B2 to numeric zero and D2 to an empty string. '
                'Preserve all other cells, leading-zero codes, booleans, styles and formulas. '
                'Return spreadsheet_token. Formula results may recalculate normally.')
    return ('Export ' + reference + ' to a local JSON file. Use an object with range and cells, '
            'where cells is a 6-by-6 array of cell objects with exact raw JSON value types and a formula '
            'string on formula cells. Preserve leading-zero strings, empty strings, false, zero, '
            'Unicode and every formula; include computed numeric values too. Do not edit the workbook. '
            'Return the relative path of the JSON file. Any CLI/API composition is allowed.')


def verify(s, answer):
    token = s.data['spreadsheet_token']
    s.private_drive(token, 'sheet')
    now = snapshot(s)
    before = s.data['before']
    task = s.request['task']['id']
    if task == 'common-sheets-intake':
        return (answer == {'updated_codes': ['001', '003'], 'skipped_codes': ['002', '004']}
                and exact(now, expected_snapshot(before, INTAKE_CHANGES)))
    if task == 'common-sheets-update':
        if answer != {'spreadsheet_token': token}:
            return False
        expected = expected_snapshot(before, {(1, 1): 0, (1, 3): ''})
        return exact(now, expected)
    if not exact(now, before):
        return False
    if task == 'common-sheets-analyze':
        return exact(answer, {'approved_total': 5, 'data_rows': 4})
    data = json.loads(s.artifact(answer.get('path')).read_text())
    if data.get('range') != EXTENT or not isinstance(data.get('cells'), list):
        return False
    cells = data['cells']
    if len(cells) != 6 or any(len(row) != 6 for row in cells):
        return False
    for row in range(6):
        for col in range(6):
            value = cells[row][col]
            if not isinstance(value, dict) or not exact(value.get('value'), now['raw'][row][col].get('value')):
                return False
            if col == 4 and 1 <= row <= 4 and value.get('formula') != '=B' + str(row + 1) + '*2':
                return False
    return True


def cleanup(s):
    s.cleanup_drive()
