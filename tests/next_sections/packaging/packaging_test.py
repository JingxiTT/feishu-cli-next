# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT
"""Real local npm/executable lifecycle. No registry, credentials, or live API."""
import hashlib
import json
import os
from pathlib import Path
import platform
import shutil
import subprocess
import sys
import tempfile
import unittest
import zipfile
import tarfile

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'scripts'))
from next_package import TARGETS, archive_name, write_archive
from next_test import isolated_env


class PackagingTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='next-packaging-')
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name)
        self.env = isolated_env(self.base / 'env', os.environ, 'http://127.0.0.1:9')
        self.env.update(npm_config_cache=str(self.base / 'npm-cache'), npm_config_update_notifier='false', npm_config_audit='false', npm_config_fund='false')
        self.binary = Path(os.environ['LARK_CLI_BIN'])
        self.target = {'Darwin': 'darwin', 'Linux': 'linux', 'Windows': 'windows'}[platform.system()] + '/' + {'x86_64': 'amd64', 'AMD64': 'amd64', 'aarch64': 'arm64', 'arm64': 'arm64', 'riscv64': 'riscv64'}[platform.machine()]
        self.binary_name = 'feishu-cli-next' + ('.exe' if os.name == 'nt' else '')

    def run_cmd(self, *args, success=True, cwd=None):
        result = subprocess.run(list(map(str, args)), env=self.env, cwd=cwd or self.base, capture_output=True, text=True, timeout=90)
        if success:
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        else:
            self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        return result

    def package(self, version='0.1.0-beta.1'):
        package = self.base / ('source-' + version)
        package.mkdir()
        shutil.copytree(ROOT / 'scripts', package / 'scripts', ignore=shutil.ignore_patterns('*test*', '__pycache__'))
        metadata = json.loads((ROOT / 'package.json').read_text())
        metadata['version'] = version
        (package / 'package.json').write_text(json.dumps(metadata))
        shutil.copy2(ROOT / 'LICENSE', package / 'LICENSE')
        artifacts = self.base / ('artifacts-' + version)
        artifacts.mkdir()
        archive = artifacts / archive_name(version, self.target)
        write_archive(archive, {self.binary_name: self.binary})
        (package / 'checksums.txt').write_text(hashlib.sha256(archive.read_bytes()).hexdigest() + '  ' + archive.name + '\n')
        self.env['FEISHU_CLI_NEXT_ARTIFACT_DIR'] = str(artifacts)
        return package, archive

    def test_all_archive_formats_variants_reproducible(self):
        fixture = self.base / 'fixture'
        fixture.write_bytes(b'fixed executable fixture\x00\xff')
        self.assertEqual(len(TARGETS), 7)
        for target in TARGETS:
            for variant in ('normal', 'authsidecar'):
                name = archive_name('1.2.3-beta.4', target, variant)
                binary_name = 'feishu-cli-next' + ('.exe' if target.startswith('windows/') else '')
                first = self.base / name
                second_dir = self.base / ('repeat-' + target.replace('/', '-') + variant)
                second_dir.mkdir()
                second = second_dir / name
                write_archive(first, {binary_name: fixture})
                write_archive(second, {binary_name: fixture})
                self.assertEqual(first.read_bytes(), second.read_bytes())
                if first.suffix == '.zip':
                    with zipfile.ZipFile(first) as archive:
                        self.assertEqual(archive.read(binary_name), fixture.read_bytes())
                else:
                    with tarfile.open(first) as archive:
                        self.assertEqual(archive.extractfile(binary_name).read(), fixture.read_bytes())
                        self.assertEqual(archive.getmember(binary_name).mode, 0o755)

    def test_installer_fails_without_channel_or_on_tamper_preserves_binary(self):
        package, archive = self.package()
        self.run_cmd('node', package / 'scripts/install.js')
        dest = package / 'bin' / self.binary_name
        before = dest.read_bytes()
        archive.write_bytes(b'corrupt archive')
        failed = self.run_cmd('node', package / 'scripts/install.js', success=False)
        self.assertIn('Checksum mismatch', failed.stderr)
        self.assertEqual(dest.read_bytes(), before)
        del self.env['FEISHU_CLI_NEXT_ARTIFACT_DIR']
        failed = self.run_cmd('node', package / 'scripts/install.js', success=False)
        self.assertIn('No feishu-cli-next release channel', failed.stderr)
        self.assertEqual(dest.read_bytes(), before)

    @unittest.skipIf(os.name == 'nt', 'POSIX symlink construction; native Windows lifecycle remains a release gate')
    def test_installer_rejects_symlink_destination(self):
        package, _ = self.package()
        target = self.base / 'unrelated'
        target.write_text('keep')
        (package / 'bin').mkdir()
        (package / 'bin' / self.binary_name).symlink_to(target)
        self.run_cmd('node', package / 'scripts/install.js', success=False)
        self.assertEqual(target.read_text(), 'keep')
        target.unlink()
        self.run_cmd('node', package / 'scripts/install.js', success=False)
        self.assertTrue((package / 'bin' / self.binary_name).is_symlink())

    def test_real_npm_install_upgrade_guide_collision_uninstall(self):
        package, _ = self.package()
        packed = json.loads(self.run_cmd('npm', 'pack', '--ignore-scripts', '--offline', '--json', cwd=package).stdout)[0]
        prefix = self.base / 'install'
        prefix.mkdir()
        sentinel = prefix / 'unrelated.txt'
        sentinel.write_text('keep prefix')
        profile = Path(self.env['LARKSUITE_CLI_CONFIG_DIR']) / 'sentinel.json'
        profile.write_text('{"unrelated":"profile"}')
        self.run_cmd('npm', 'install', '--offline', '--prefix', prefix, package / packed['filename'])
        runner = prefix / 'node_modules/feishu-cli-next/scripts/run.js'
        self.assertIn('feishu-cli-next', self.run_cmd('node', runner, '--version').stdout)
        project = self.base / 'project'
        project.mkdir()
        unrelated_skill = project / '.agents/skills/other/SKILL.md'
        unrelated_skill.parent.mkdir(parents=True)
        unrelated_skill.write_text('keep skill')
        installed = json.loads(self.run_cmd('node', runner, 'install', '--project', project).stdout)
        self.assertTrue(installed['changed'])
        managed = project / '.agents/skills/feishu-cli-next/SKILL.md'
        original = managed.read_bytes()
        repeated = json.loads(self.run_cmd('node', runner, 'guide', 'install', '--project', project).stdout)
        self.assertFalse(repeated['changed'])
        managed.write_bytes(original + b'local edit')
        self.run_cmd('node', runner, 'guide', 'update', '--project', project, success=False)
        self.run_cmd('node', runner, 'guide', 'uninstall', '--project', project, success=False)
        self.assertEqual(managed.read_bytes(), original + b'local edit')
        managed.write_bytes(original)
        second, _ = self.package('0.1.0-beta.2')
        packed = json.loads(self.run_cmd('npm', 'pack', '--ignore-scripts', '--offline', '--json', cwd=second).stdout)[0]
        self.run_cmd('npm', 'install', '--offline', '--prefix', prefix, second / packed['filename'])
        metadata = json.loads((prefix / 'node_modules/feishu-cli-next/package.json').read_text())
        self.assertEqual(metadata['version'], '0.1.0-beta.2')
        updated = json.loads(self.run_cmd('node', runner, 'guide', 'update', '--project', project).stdout)
        self.assertFalse(updated['changed']) # fixture binary deliberately unchanged
        self.run_cmd('npm', 'uninstall', '--offline', '--prefix', prefix, 'feishu-cli-next')
        self.assertFalse(runner.exists())
        self.assertEqual(managed.read_bytes(), original)
        self.assertEqual(unrelated_skill.read_text(), 'keep skill')
        self.assertEqual(profile.read_text(), '{"unrelated":"profile"}')
        self.assertEqual(sentinel.read_text(), 'keep prefix')
        self.run_cmd(self.binary, 'guide', 'uninstall', '--project', project)
        self.assertFalse(managed.exists())
        self.assertEqual(unrelated_skill.read_text(), 'keep skill')

    def test_prefix_binary_lifecycle_collisions_and_preservation(self):
        prefix = self.base / 'prefix'
        binary = prefix / 'bin' / self.binary_name
        binary.parent.mkdir(parents=True)
        sentinel = binary.parent / 'unrelated'
        sentinel.write_text('keep')
        script = ROOT / 'scripts/next_binary.py'
        source = self.base / 'source-binary'
        source.write_bytes(b'first executable fixture')
        def invoke(action, success=True):
            return self.run_cmd('python3', script, action, '--prefix', prefix, '--source', source, success=success)
        binary.write_text('collision')
        invoke('install', success=False)
        self.assertEqual(binary.read_text(), 'collision')
        invoke('uninstall', success=False)
        binary.unlink()
        self.assertTrue(json.loads(invoke('install').stdout)['changed'])
        self.assertEqual(binary.read_bytes(), source.read_bytes())
        if os.name != 'nt':
            self.assertEqual(binary.stat().st_mode & 0o777, 0o755)
        self.assertFalse(json.loads(invoke('install').stdout)['changed'])
        source.write_bytes(b'second executable fixture')
        invoke('install', success=False)
        invoke('update')
        self.assertEqual(binary.read_bytes(), source.read_bytes())
        binary.write_bytes(b'local edit')
        invoke('update', success=False)
        invoke('uninstall', success=False)
        self.assertEqual(binary.read_bytes(), b'local edit')
        binary.write_bytes(source.read_bytes())
        invoke('uninstall')
        self.assertFalse(binary.exists())
        self.assertFalse(json.loads(invoke('uninstall').stdout)['changed'])
        invoke('update', success=False)
        self.assertEqual(sentinel.read_text(), 'keep')
        if os.name != 'nt':
            binary.symlink_to(sentinel)
            invoke('install', success=False)
            invoke('uninstall', success=False)
            self.assertEqual(sentinel.read_text(), 'keep')

    def test_package_identity_and_publication_guards(self):
        package = json.loads((ROOT / 'package.json').read_text())
        self.assertEqual(package['name'], 'feishu-cli-next')
        self.assertTrue(package['private'])
        self.assertEqual(list(package['bin']), ['feishu-cli-next'])
        self.assertNotIn('dependencies', package)
        result = self.run_cmd('bash', ROOT / 'scripts/tag-release.sh', success=False)
        self.assertIn('Release publication is disabled', result.stderr)


if __name__ == '__main__':
    unittest.main()
