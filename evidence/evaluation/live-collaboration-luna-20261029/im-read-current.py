# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Read only explicitly designated private TEST conversations; never seed or send."""
import hashlib
import json
from urllib.parse import urlencode

from live_domains.session import Session, exact, private_json, require
from live_fixture import BACKEND


def fixture(resources, identifier):
    key = 'im_messages' if identifier == 'collaboration-im-read' else 'im_group'
    value = private_json(resources).get(key, {})
    require(value.get('read_authorized_by_user') is True and value.get('label') == '[TEST]'
            and isinstance(value.get('chat_id'), str) and value['chat_id'].startswith('oc_')
            and isinstance(value.get('account_binding'), str), 'designated_private_im_fixture_required')
    if key == 'im_messages':
        ids = value.get('message_ids')
        require(isinstance(ids, list) and ids and len(set(ids)) == len(ids)
                and all(isinstance(v, str) and v.startswith('om_') for v in ids), 'im_message_ids_required')
    else:
        require(type(value.get('start_time')) is int and type(value.get('end_time')) is int
                and 0 < value['start_time'] < value['end_time'], 'bounded_group_window_required')
    return value


def private_chat(s, config):
    require(config['account_binding'] == s.data['binding'], 'im_fixture_binding_mismatch')
    chat = s.api('GET', '/open-apis/im/v1/chats/' + config['chat_id'])
    require('[TEST]' in chat.get('name', '') and chat.get('owner_id') == s.account['open_id'],
            'private_test_chat_owner_unverified')
    members = s.call(['im', '+chat-members-list', '--chat-id', config['chat_id'],
        '--page-all', '--page-limit', '0', '--as', 'user'])
    require(members.get('has_more') is False and not members.get('truncations')
            and members.get('bots') == [] and len(members.get('users', [])) == 1,
            'private_test_chat_members_unverified')
    user = members['users'][0]
    require(user.get('member_id') == s.account['open_id']
            and user.get('tenant_key') == s.account['tenant_key'], 'private_test_chat_identity_mismatch')


def read(s, config):
    if s.request['task']['id'] == 'collaboration-group-read':
        items = s.pages('/open-apis/im/v1/messages', params={
            'container_id_type': 'chat', 'container_id': config['chat_id'],
            'start_time': str(config['start_time']), 'end_time': str(config['end_time']),
            'sort_type': 'ByCreateTimeAsc'}, page_size=50)
    else:
        items = []
        for offset in range(0, len(config['message_ids']), 50):
            query = urlencode([('message_ids', v) for v in config['message_ids'][offset:offset + 50]])
            data = s.api('GET', '/open-apis/im/v1/messages/mget?' + query)
            require(isinstance(data.get('items'), list), 'im_message_extent_missing')
            items.extend(data['items'])
        require({v.get('message_id') for v in items} == set(config['message_ids']), 'im_message_read_incomplete')
    require(items and len({v['message_id'] for v in items}) == len(items)
            and all(v.get('chat_id') == config['chat_id'] for v in items), 'im_message_scope_mismatch')
    return sorted(items, key=lambda v: (int(v['create_time']), v['message_id']))


def projection(items):
    return [{k: v.get(k) for k in ['message_id', 'msg_type', 'create_time', 'update_time',
                                  'deleted', 'updated', 'body', 'sender']} for v in items]


def report(items):
    counts = {}
    for item in items:
        counts[item['msg_type']] = counts.get(item['msg_type'], 0) + 1
    content = {v['message_id']: v['body']['content'] for v in items}
    return {'ordered_message_ids': [v['message_id'] for v in items], 'message_count': len(items),
            'message_type_counts': counts,
            'edited_message_ids': sorted(v['message_id'] for v in items if v.get('updated') is True
                or int(v.get('update_time', 0)) > int(v['create_time'])),
            'body_sha256': hashlib.sha256(json.dumps(content, ensure_ascii=False, sort_keys=True,
                separators=(',', ':')).encode()).hexdigest()}


def run(request, cli, binding, resources):
    config = fixture(resources, request['task']['id'])
    s = Session(request, cli, binding, resources)
    if request['phase'] == 'cleanup':
        require(s.data['pending'] is None and s.data['owned'] == [], 'unexpected_im_mutation')
        s.data['cleanup_confirmed'] = True
        s.save()
        return {'cleaned': True}
    private_chat(s, config)
    if request['phase'] == 'preflight':
        return {'backend': BACKEND, 'prerequisites_verified': True, 'model_execution': 'not_run'}
    current = read(s, config)
    if request['phase'] == 'setup':
        s.data['before'] = projection(current)
        s.save()
        target = ('all messages in the closed time window start_time=' + str(config['start_time'])
                  + ', end_time=' + str(config['end_time']) + ', including every page'
                  if request['task']['id'] == 'collaboration-group-read' else
                  'the exact message IDs in message-ids.json')
        if 'message_ids' in config:
            (s.workspace / 'message-ids.json').write_text(json.dumps(config['message_ids']))
        return {'backend': BACKEND, 'ready': True, 'account_verified': True, 'privacy_verified': True,
                'prompt': 'Read ' + target + ' in designated private TEST chat ' + config['chat_id']
                + '. Do not send, react, edit, delete, mark, join, invite or change anything. Write '
                'report.json with ordered_message_ids (create_time ascending, then ID), message_count, '
                'message_type_counts, sorted edited_message_ids (updated=true or update_time>create_time), '
                'and body_sha256: SHA256 of UTF-8 json.dumps of the message_id-to-raw-body.content '
                'mapping with ensure_ascii=False, sort_keys=True, separators=(",",":"). '
                'Preserve exact Unicode, raw message bodies and complete pagination. Return {"path":"report.json"}.'}
    require(request['phase'] == 'verify', 'unsupported_fixture_phase')
    try:
        answer = json.loads(s.artifact('answer.json').read_text())
        actual = json.loads(s.artifact(answer['path']).read_text())
        success = exact(projection(current), s.data['before']) and exact(actual, report(current))
    except (OSError, ValueError, KeyError, TypeError):
        success = False
    return {'backend': BACKEND, 'success': success}
