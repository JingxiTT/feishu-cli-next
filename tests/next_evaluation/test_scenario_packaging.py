"""Filesystem verifier regressions; no live API, model, or user-home access."""
import hashlib
import importlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'evaluation'))
packaging = importlib.import_module('scenarios.packaging')


class PackagingScenarioTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='packaging-scenario-')
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.workspace = self.root / 'workspace'
        self.workspace.mkdir()
        self.source = self.root / 'selected-clean-source'
        self.source.mkdir()
        self.scenario = None

    def prepare(self, name='lark-shared'):
        (self.source / 'SKILL.md').write_bytes(
            ('---\nname: ' + name + '\ndescription: Synthetic source distribution.\n---\nPreserve Unicode 世界.\n').encode())
        if name == 'lark-shared':
            (self.source / 'references').mkdir()
            (self.source / 'references/detail.md').write_bytes(b'preserve references\x00\n')
        scenario = packaging.Scenario()
        scenario.selected_skill_source = self.source
        scenario.setup(self.workspace)
        scenario.isolation_proof = {'protected_home_writes_denied': True, 'network_denied': True}
        self.scenario = scenario
        return scenario

    def install(self, managed=False):
        scenario = self.scenario
        destination = self.workspace / scenario.destination
        shutil.copytree(self.workspace / 'inputs/selected-skill', destination)
        if managed:
            skill = destination / 'SKILL.md'
            data = skill.read_bytes()
            skill.write_bytes(data + packaging.MARKER + hashlib.sha256(data).hexdigest().encode() + b' -->\n')
        return destination

    def answer(self):
        return {'installed_path': self.scenario.destination + '/SKILL.md',
                'skill_name': self.scenario.name, 'preserved_readme': True}

    def receipt(self):
        return {'scenario': packaging.SCENARIO, 'requests': [], 'state': self.scenario.snapshot()}

    def check(self, answer=None, receipt=None):
        return packaging.verify(self.workspace, answer if answer is not None else self.answer(),
                                receipt if receipt is not None else self.receipt(), packaging.TASK['expected'])

    def test_official_complete_copy_and_public_inputs(self):
        scenario = self.prepare()
        self.assertFalse(self.check())
        self.install()
        self.assertTrue(self.check())
        inputs = json.loads((self.workspace / 'inputs/install.json').read_text())
        self.assertEqual(inputs, {'skill_name': 'lark-shared', 'source_dir': 'inputs/selected-skill', 'entry_file': 'SKILL.md'})
        self.assertEqual(set(packaging.TASK['fixture']), {'scenario', 'answer_schema'})
        self.assertNotIn('const', json.dumps(packaging.TASK['fixture']['answer_schema']))
        self.assertNotIn('sha256', json.dumps(inputs))
        with self.assertRaises(ValueError):
            scenario.dispatch('GET', '/open-apis/authen/v1/user_info', b'', {})

    def test_next_copy_is_valid_without_native_marker(self):
        self.prepare('feishu-cli-next')
        self.install()
        self.assertTrue(self.check())

    def assert_installation_path_forms(self, name):
        self.prepare(name)
        self.install()
        destination = self.scenario.destination
        receipt = self.receipt()
        for path in (destination, destination + '/', destination + '/SKILL.md',
                     './' + destination, './' + destination + '/', './' + destination + '/SKILL.md'):
            with self.subTest(path=path):
                self.assertTrue(self.check({**self.answer(), 'installed_path': path}, receipt))
        for path in ('project/.agents/skills', destination + '/SKILL.md/',
                     '../' + destination, '/'+destination, destination + '/../' + name,
                     destination + '-other', 0, None):
            answer = {**self.answer(), 'installed_path': path}
            with self.subTest(path=path):
                self.assertFalse(self.check(answer, receipt))
                self.assertEqual(packaging.verification_diagnostics(self.workspace, answer, receipt,
                    packaging.TASK['expected']), ['packaging_answer_path_mismatch'])
        self.assertEqual(self.receipt(), receipt)
        (self.workspace / destination / 'SKILL.md').write_bytes(b'corrupt')
        self.assertFalse(self.check({**self.answer(), 'installed_path': destination}))

    def test_official_directory_and_entry_path_have_identical_artifact_checks(self):
        self.assert_installation_path_forms('lark-shared')

    def test_next_directory_and_entry_path_have_identical_artifact_checks(self):
        self.assert_installation_path_forms('feishu-cli-next')

    def test_failure_diagnostics_are_fixed_without_private_content(self):
        self.prepare()
        self.install()
        receipt = self.receipt()
        receipt['state']['isolation_proof'] = None
        self.assertEqual(packaging.verification_diagnostics(self.workspace, self.answer(), receipt,
            packaging.TASK['expected']), ['packaging_isolation_missing'])
        (self.workspace / 'inputs/install.json').write_text('private changed content')
        reasons = packaging.verification_diagnostics(self.workspace, self.answer(), self.receipt(),
            packaging.TASK['expected'])
        self.assertEqual(reasons, ['packaging_protected_changed'])
        self.assertNotIn('private', json.dumps(reasons))

    def test_next_managed_footer_is_valid_and_exact(self):
        self.prepare('feishu-cli-next')
        destination = self.install(managed=True)
        self.assertTrue(self.check())
        skill = destination / 'SKILL.md'
        original = skill.read_bytes()
        for suffix in (b'edit', b'\n'):
            with self.subTest(suffix=suffix):
                skill.write_bytes(original + suffix)
                self.assertFalse(self.check())
        skill.write_bytes(original.replace(b'sha256:', b'sha256:0', 1))
        self.assertFalse(self.check())

    def test_answers_do_not_substitute_for_artifacts_or_types(self):
        self.prepare()
        self.assertFalse(self.check())
        self.install()
        for change in ({'skill_name': 'feishu-cli-next'}, {'preserved_readme': 1},
                       {'installed_path': '../outside/SKILL.md'}, {'extra': 'claim'}):
            with self.subTest(change=change):
                self.assertFalse(self.check({**self.answer(), **change}))
        receipt = self.receipt()
        receipt['requests'] = [{'route': 'rejected', 'status': 403}]
        self.assertFalse(self.check(receipt=receipt))

    def test_readme_and_protected_home_config_inputs_must_be_preserved(self):
        self.prepare()
        self.install()
        for relative in ('project/README.md', 'home/.packaging-preserve',
                         'feishu-config/.packaging-preserve', 'inputs/install.json'):
            with self.subTest(path=relative):
                path = self.workspace / relative
                original = path.read_bytes()
                path.write_bytes(original + b'edit')
                self.assertFalse(self.check())
                path.write_bytes(original)
        other = self.workspace / 'project/unrelated-new-file'
        other.write_text('unexpected')
        self.assertFalse(self.check())

    def test_extra_missing_or_changed_reference_and_empty_directories_fail(self):
        self.prepare()
        destination = self.install()
        reference = destination / 'references/detail.md'
        original = reference.read_bytes()
        reference.unlink()
        self.assertFalse(self.check())
        reference.write_bytes(original + b'changed')
        self.assertFalse(self.check())
        reference.write_bytes(original)
        lock = destination / '.install.lock'
        lock.write_bytes(b'')
        self.assertFalse(self.check())
        lock.unlink()
        (destination / 'unexpected-empty').mkdir()
        self.assertFalse(self.check())

    @unittest.skipIf(os.name == 'nt', 'symlink privilege differs on Windows; no native Windows claim')
    def test_links_are_not_independent_installations(self):
        self.prepare()
        destination = self.install()
        skill = destination / 'SKILL.md'
        skill.unlink()
        source = self.workspace / 'inputs/selected-skill/SKILL.md'
        skill.symlink_to(source)
        self.assertFalse(self.check())
        skill.unlink()
        os.link(source, skill)
        self.assertFalse(self.check())

    def test_snapshot_and_trusted_isolation_evidence_are_required(self):
        self.prepare()
        self.install()
        self.assertTrue(self.check())
        for proof in (None, {}, {'protected_home_writes_denied': 1, 'network_denied': True},
                      {'protected_home_writes_denied': True, 'network_denied': False}):
            with self.subTest(proof=proof):
                receipt = self.receipt()
                receipt['state']['isolation_proof'] = proof
                self.assertFalse(self.check(receipt=receipt))
        receipt = self.receipt()
        (self.workspace / 'project/README.md').write_bytes(b'changed after snapshot')
        self.assertFalse(self.check(receipt=receipt))

    def test_setup_requires_explicit_clean_source_and_snapshot_is_detached(self):
        with self.assertRaises(ValueError):
            packaging.Scenario().setup(self.workspace)
        scenario = self.prepare()
        state = scenario.snapshot()
        state['source_files']['SKILL.md']['sha256'] = 'forged'
        self.assertNotEqual(scenario.snapshot()['source_files']['SKILL.md']['sha256'], 'forged')

    @unittest.skipUnless(os.environ.get('PACKAGING_EVAL_NEXT_BINARY') and os.environ.get('PACKAGING_EVAL_NEXT_SOURCE'),
                         'optional native check requires explicit clean-archive binary and skill source')
    def test_real_clean_candidate_guide_install(self):
        source = Path(os.environ['PACKAGING_EVAL_NEXT_SOURCE'])
        binary = Path(os.environ['PACKAGING_EVAL_NEXT_BINARY']).resolve()
        scenario = packaging.Scenario()
        scenario.selected_skill_source = source
        scenario.setup(self.workspace)
        self.scenario = scenario
        sys.path.insert(0, str(ROOT / 'scripts'))
        from next_test import isolated_env
        with tempfile.TemporaryDirectory(prefix='packaging-native-env-') as envdir:
            env = isolated_env(envdir, os.environ, 'http://127.0.0.1:9')
            env.update(HOME=str(self.workspace / 'home'), USERPROFILE=str(self.workspace / 'home'),
                       LARKSUITE_CLI_CONFIG_DIR=str(self.workspace / 'feishu-config'))
            result = subprocess.run([str(binary), 'guide', 'install', '--project', str(self.workspace / 'project')],
                                    env=env, capture_output=True, text=True, timeout=30)
        self.assertEqual(result.returncode, 0, result.stderr)
        # This optional direct invocation has no OS write-denial proof. Check
        # actual native artifacts/preservation, but do not claim full task success.
        state = scenario.snapshot()
        for directory in packaging.PROTECTED:
            self.assertEqual(state['initial_protected'][directory], state['final_protected'][directory])
        self.assertTrue((self.workspace / scenario.destination / 'SKILL.md').is_file())
        self.assertFalse(self.check())



if __name__ == '__main__':
    unittest.main()
