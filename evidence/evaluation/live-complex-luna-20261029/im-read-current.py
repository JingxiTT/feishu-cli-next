# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Read explicitly designated TEST messages; never seed or send in model attempts."""
import hashlib
import json
import uuid
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
                and 0 < value['start_time'] < value['end_time']
                and value.get('chat_kind') != 'p2p', 'bounded_group_window_required')
    return value


def private_chat(s, config):
    require(config['account_binding'] == s.data['binding'], 'im_fixture_binding_mismatch')
    recipient = s.request.get('recipient') if config.get('requires_recipient') is True else None
    expected = {s.account['open_id']}
    if recipient is not None:
        require(set(recipient) == {'open_id', 'tenant_key'}
                and recipient['tenant_key'] == s.account['tenant_key'], 'im_recipient_tenant_mismatch')
        lookup = s.call(['contact', '+search-user', '--user-ids', recipient['open_id'], '--as', 'user'])
        users = lookup.get('users', [])
        require(lookup.get('has_more') is False and len(users) == 1
                and users[0].get('open_id') == recipient['open_id']
                and users[0].get('is_cross_tenant') is False, 'im_recipient_identity_unverified')
        expected.add(recipient['open_id'])
        if config.get('chat_kind') == 'p2p':
            require(users[0].get('p2p_chat_id') == config['chat_id'], 'im_direct_destination_mismatch')
    require(not config.get('requires_recipient') or recipient is not None, 'im_recipient_required')
    if config.get('chat_kind') != 'p2p':
        chat = s.api('GET', '/open-apis/im/v1/chats/' + config['chat_id'])
        require('[TEST]' in chat.get('name', '') and chat.get('owner_id') == s.account['open_id'],
                'private_test_chat_owner_unverified')
    else:
        require(recipient is not None and config.get('seeded_by_run') is True,
                'im_direct_test_provenance_required')
    members = s.call(['im', '+chat-members-list', '--chat-id', config['chat_id'],
        '--page-all', '--page-limit', '0', '--as', 'user'])
    require(members.get('has_more') is False and not members.get('truncations')
            and members.get('bots') == [] and len(members.get('users', [])) == len(expected),
            'private_test_chat_members_unverified')
    require({u.get('member_id') for u in members['users']} == expected
            and all(u.get('tenant_key') == s.account['tenant_key'] for u in members['users']),
            'private_test_chat_identity_mismatch')
    return recipient


def read(s, config):
    if s.request['task']['id'] == 'collaboration-group-read':
        items = s.pages('/open-apis/im/v1/messages', params={
            'container_id_type': 'chat', 'container_id': config['chat_id'],
            'start_time': str(config['start_time']), 'end_time': str(config['end_time']),
            'sort_type': 'ByCreateTimeAsc'}, page_size=config.get('page_size', 50))
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


def seed(s, config):
    """Trusted calibration only: four fictitious TEST messages, shared read-only by all arms."""
    require(config.get('seeded_by_run') is True and config.get('send_authorized_by_user') is True,
            'im_test_send_authorization_required')
    private_chat(s, config)
    require(not s.data.get('seed_started'), 'im_seed_cannot_be_replayed')
    s.data['seed_started'] = True
    s.save()
    for index, text in enumerate([
            'R43 | INV-0001 | APPROVED | Amount=0 | 你好 😀',
            'R43 | INV-0002 | PENDING | Amount=-7.25 | résumé',
            'R42 | INV-0001 | APPROVED | Amount=12 | old release',
            'R43 | INV-0003 | APPROVED | Amount=7.25 | line\nbreak']):
        # Text plus rich post exercise raw-body preservation without uploading resources.
        content = {'text': s.data['tag'] + ' | ' + text}
        kind = 'text'
        if index == 3:
            kind = 'post'
            content = {'zh_cn': {'title': s.data['tag'], 'content': [[{'tag': 'text', 'text': text}]]}}
        s.mutation('send-test-message', lambda content=content, kind=kind: s.api(
            'POST', '/open-apis/im/v1/messages', params={'receive_id_type': 'chat_id'}, body={
                'receive_id': config['chat_id'], 'msg_type': kind,
                'content': json.dumps(content, ensure_ascii=False), 'uuid': str(uuid.uuid4())}),
            ('message', 'message_id'))
    return [item['id'] for item in s.data['owned'] if item['kind'] == 'message']


def cleanup_seed(s, config):
    private_chat(s, config)
    require(s.data['pending'] is None, 'mutation_requires_reconciliation')
    for item in s.data['owned']:
        require(item['kind'] == 'message', 'unexpected_im_resource')
        if item.get('deleted'):
            continue
        data = s.api('GET', '/open-apis/im/v1/messages/' + item['id'])
        messages = data.get('items', [])
        require(len(messages) == 1 and messages[0].get('chat_id') == config['chat_id']
                and messages[0].get('sender', {}).get('id') == s.account['open_id']
                and s.data['tag'] in messages[0].get('body', {}).get('content', ''),
                'im_cleanup_ownership_unverified')
        s.mutation('recall-owned-test-message', lambda item=item: s.api(
            'DELETE', '/open-apis/im/v1/messages/' + item['id']),
            target={'kind': 'message', 'id': item['id']})
        result = s.api('GET', '/open-apis/im/v1/messages/' + item['id'])
        require(len(result.get('items', [])) == 1 and result['items'][0].get('deleted') is True,
                'im_recall_unconfirmed')
    s.data['cleanup_confirmed'] = True
    s.save()


def run(request, cli, binding, resources):
    config = fixture(resources, request['task']['id'])
    s = Session(request, cli, binding, resources)
    if request['phase'] == 'cleanup':
        require(s.data['pending'] is None and s.data['owned'] == [], 'unexpected_im_mutation')
        s.data['cleanup_confirmed'] = True
        s.save()
        return {'cleaned': True}
    recipient = private_chat(s, config)
    if request['phase'] == 'preflight':
        return {'backend': BACKEND, 'prerequisites_verified': True, 'model_execution': 'not_run'}
    current = read(s, config)
    if config.get('seeded_by_run'):
        require(all('[TEST]' in v.get('body', {}).get('content', '')
                    and v.get('sender', {}).get('id') == s.account['open_id'] for v in current),
                'im_test_message_provenance_mismatch')
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
                **({'verified_recipient': recipient} if recipient else {}),
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
