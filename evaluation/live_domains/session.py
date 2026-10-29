# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Account binding, private fixture state and single-attempt controller calls."""
import hashlib
import json
import math
import os
from pathlib import Path
import stat
import subprocess
import uuid

from harness import atomic_json
from live_contact import current_account


class Blocked(ValueError):
    """A fixed public reason; never include API output or identifiers."""


def require(condition, reason):
    if not condition:
        raise Blocked(reason)


def private_json(path):
    path = Path(path)
    info = path.lstat()
    require(path.is_absolute() and stat.S_ISREG(info.st_mode) and not info.st_mode & 0o077,
            'private_file_required')
    return json.loads(path.read_text())


def exact(left, right):
    """Do not equate JSON false with zero, or an omitted value with null."""
    if type(left) is bool or type(right) is bool:
        return type(left) is type(right) and left == right
    if isinstance(left, (int, float)) and isinstance(right, (int, float)):
        return all(type(v) is int or math.isfinite(v) for v in (left, right)) and left == right
    if type(left) is not type(right):
        return False
    if isinstance(left, dict):
        return left.keys() == right.keys() and all(exact(left[k], right[k]) for k in left)
    if isinstance(left, list):
        return len(left) == len(right) and all(exact(a, b) for a, b in zip(left, right))
    return left == right


def at(value, path):
    for key in path.split('.'):
        value = value[int(key)] if isinstance(value, list) else value[key]
    return value


class Session:
    def __init__(self, request, cli, binding, resources):
        require(not os.environ.get('CI'), 'live_evaluation_forbidden_in_ci')
        self.request, self.cli = request, Path(cli)
        self.state = Path(request['state_dir']).resolve()
        self.workspace = Path(request['workspace']).resolve()
        require(self.cli.is_absolute() and self.cli.is_file(), 'production_cli_required')
        require(self.workspace.parent == self.state and self.workspace.is_dir(), 'invalid_workspace')
        require(not self.state.stat().st_mode & 0o077, 'private_state_required')
        self.path = self.state / 'common-state.json'
        self.resources = private_json(resources)
        require(isinstance(self.resources, dict), 'invalid_resources_manifest')
        self.account = current_account(self.cli, request['cli_config_dir'], request['cli_data_dir'], self.state)
        proof = private_json(binding)
        digest = hashlib.sha256(('contact-current-account-v1\0' + self.account['open_id'] + '\0'
                                 + self.account['tenant_key']).encode()).hexdigest()
        require(proof == {'algorithm': 'sha256', 'digest': digest}, 'account_binding_mismatch')
        self.env = {'PATH': os.defpath, 'HOME': str(self.state), 'TMPDIR': str(self.state),
                    'LANG': 'C.UTF-8', 'LARKSUITE_CLI_CONFIG_DIR': request['cli_config_dir'],
                    'LARKSUITE_CLI_DATA_DIR': request['cli_data_dir'],
                    'LARKSUITE_CLI_REMOTE_META': 'off', 'LARKSUITE_CLI_NO_UPDATE_NOTIFIER': '1',
                    'LARKSUITE_CLI_NO_SKILLS_NOTIFIER': '1'}
        if self.path.exists():
            self.data = private_json(self.path)
            require(self.data['binding'] == digest and self.data['task_id'] == request['task']['id'],
                    'fixture_binding_changed')
        else:
            require(request['phase'] in ('setup', 'preflight'), 'fixture_state_missing')
            self.data = {'binding': digest, 'task_id': request['task']['id'],
                         'tag': '[TEST] cli-eval-' + uuid.uuid4().hex[:16], 'owned': [], 'pending': None}
            self.save()

    def save(self):
        atomic_json(self.path, self.data)
        self.path.chmod(0o600)

    def call(self, arguments):
        result = subprocess.run([str(self.cli), *arguments], cwd=self.state, env=self.env,
                                capture_output=True, text=True, timeout=45, check=False)
        # Recovery evidence belongs outside the model workspace, including on failure.
        name = self.state / ('command-' + uuid.uuid4().hex + '.json')
        atomic_json(name, {'exit_code': result.returncode, 'stdout': result.stdout, 'stderr': result.stderr})
        name.chmod(0o600)
        try:
            parsed = json.loads(result.stdout)
        except ValueError:
            parsed = None
        self.last_data = parsed.get('data') if isinstance(parsed, dict) else None
        if result.returncode != 0:
            try:
                problem = json.loads(result.stderr).get('error', {})
            except (ValueError, AttributeError):
                problem = {}
            if problem.get('subtype') in ('missing_scope', 'token_expired', 'token_missing',
                                          'permission_denied', 'confirmation_required'):
                raise Blocked('controller_' + problem['subtype'])
        require(result.returncode == 0, 'controller_command_failed')
        require(isinstance(parsed, dict) and parsed.get('ok') is True
                and isinstance(parsed.get('data'), dict), 'invalid_controller_result')
        require(not parsed.get('meta', {}).get('partial') and not parsed.get('data', {}).get('partial'),
                'partial_controller_result')
        return parsed['data']

    def api(self, method, path, params=None, body=None):
        arguments = ['api', method, path, '--as', 'user']
        if params is not None:
            arguments += ['--params', json.dumps(params)]
        if body is not None:
            arguments += ['--data', json.dumps(body, ensure_ascii=False)]
        return self.call(arguments)

    def mutation(self, label, operation, resource=None, target=None):
        require(self.data['pending'] is None, 'mutation_requires_reconciliation')
        if target:
            require(not any(r['kind'] == target['kind'] and r['id'] == target['id'] and r.get('deleted')
                            for r in self.data['owned']), 'acknowledged_delete_cannot_be_replayed')
        self.data['pending'] = {'operation': label, 'target': target} if target else label
        self.save()  # A process crash or ambiguous response must never replay this write.
        self.last_data = None
        try:
            result = operation()
        except Exception:
            # Partial CLI output may already contain a returned cleanup coordinate.
            # Retain it without treating the mutation as confirmed or clearing pending.
            if resource and isinstance(self.last_data, dict):
                kind, key = resource
                try:
                    identifier = at(self.last_data, key)
                except (KeyError, IndexError, TypeError):
                    identifier = None
                if isinstance(identifier, str) and identifier:
                    self.data['owned'].append({'kind': kind, 'id': identifier,
                                               'creation_outcome': 'unknown', 'deleted': False})
                    self.save()
            raise
        if resource:
            kind, key = resource
            identifier = at(result, key)
            require(isinstance(identifier, str) and identifier, 'created_id_missing')
            self.data['owned'].append({'kind': kind, 'id': identifier, 'deleted': False})
        if target:
            owned = next((r for r in self.data['owned']
                          if r['kind'] == target['kind'] and r['id'] == target['id']), None)
            if owned is None:
                owned = {**target, 'source': 'verified_cleanup_target'}
                self.data['owned'].append(owned)
            owned['deleted'] = True
        self.data['pending'] = None
        self.save()
        return result

    def pages(self, path, key='items', params=None, page_size=100):
        items, cursor, seen = [], '', set()
        for _ in range(1000):
            data = self.api('GET', path, {**(params or {}), 'page_size': page_size,
                                         **({'page_token': cursor} if cursor else {})})
            require(isinstance(data.get(key), list) and type(data.get('has_more')) is bool,
                    'pagination_extent_missing')
            items.extend(data[key])
            if data['has_more'] is False:
                return items
            cursor = data.get('page_token')
            require(isinstance(cursor, str) and cursor and cursor not in seen, 'pagination_not_advancing')
            seen.add(cursor)
        raise Blocked('pagination_budget_exceeded')

    def artifact(self, name):
        require(isinstance(name, str) and name and not Path(name).is_absolute(), 'relative_artifact_required')
        path = (self.workspace / name).resolve()
        require(path.is_relative_to(self.workspace) and path != self.workspace, 'artifact_outside_workspace')
        return path

    def private_drive(self, token, kind):
        settings = self.call(['drive', '+permission-get-setting', '--token', token, '--type', kind, '--as', 'user'])
        members = self.call(['drive', '+member-list', '--token', token, '--type', kind, '--as', 'user'])
        metadata = self.api('POST', '/open-apis/drive/v1/metas/batch_query', body={
            'request_docs': [{'doc_token': token, 'doc_type': kind}], 'with_url': False})
        owner = self.account['open_id']
        require(settings.get('permission_public', {}).get('link_share_entity') == 'closed', 'link_access_not_private')
        items, metas = members.get('items'), metadata.get('metas')
        require(isinstance(items, list) and len(items) == 1 and items[0].get('member_type') == 'openid'
                and items[0].get('member_id') == owner and items[0].get('perm') == 'full_access',
                'collaborators_not_private')
        require(kind != 'folder' or items[0].get('perm_type') == 'container', 'folder_inheritance_unverified')
        require(isinstance(metas, list) and len(metas) == 1 and metas[0].get('doc_token') == token
                and metas[0].get('doc_type') == kind and metas[0].get('owner_id') == owner,
                'resource_owner_unverified')

    def parent(self):
        parent = self.resources.get('private_parent', {})
        require(parent.get('creation_authorized_by_user') is True and parent.get('label') == '[TEST]'
                and parent.get('account_binding') == self.data['binding']
                and isinstance(parent.get('folder_token'), str) and parent['folder_token'],
                'authorized_private_test_parent_required')
        self.private_drive(parent['folder_token'], 'folder')
        return parent['folder_token']

    def folder(self):
        parent = self.parent()
        result = self.mutation('create-attempt-folder', lambda: self.api(
            'POST', '/open-apis/drive/v1/files/create_folder', body={
                'name': self.data['tag'], 'folder_token': parent}), ('folder', 'token'))
        token = result['token']
        self.data['folder'] = token
        self.save()
        self.private_drive(token, 'folder')
        return token

    def cleanup_drive(self):
        require(self.data['pending'] is None, 'mutation_requires_reconciliation')
        folder = self.data.get('folder')
        if not folder:
            require(not self.data['owned'], 'resource_requires_reconciliation')
            return
        self.private_drive(folder, 'folder')
        # Cleanup is by the exact IDs journaled at creation/readback. Broad Drive
        # listing is deliberately unnecessary: these are fresh run-owned folders.
        owned = [r for r in self.data['owned'] if r['id'] != folder and not r.get('deleted')]
        for item in owned + [{'kind': 'folder', 'id': folder}]:
            self.private_drive(item['id'], item['kind'])
            result = self.mutation('delete-owned-resource', lambda item=item: self.call([
                'drive', '+delete', '--file-token', item['id'], '--type', item['kind'], '--as', 'user', '--yes']),
                target={'kind': item['kind'], 'id': item['id']})
            require(result.get('deleted') is True, 'resource_cleanup_unconfirmed')
