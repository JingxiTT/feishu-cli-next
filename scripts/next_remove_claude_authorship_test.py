# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import next_remove_claude_authorship as cleanup


ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / 'scripts/next_remove_claude_authorship.py'


class AuthorCleanupTest(unittest.TestCase):
    def test_removes_only_claude_coauthor_lines(self):
        message = (b'Document Claude usage\n\nKeep this body.\n\n'
                   b'Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>\n'
                   b'co-authored-by: Claude Sonnet <bot@example.com>\r\n'
                   b'Co-authored-by: Bot <claude@anthropic.com>\n'
                   b'Co-authored-by: Alice <alice@anthropic.com>\n'
                   b'Co-authored-by: Claudette <human@example.com>\n'
                   b'Signed-off-by: Alice <alice@example.com>\n')
        expected = (b'Document Claude usage\n\nKeep this body.\n\n'
                    b'Co-authored-by: Alice <alice@anthropic.com>\n'
                    b'Co-authored-by: Claudette <human@example.com>\n'
                    b'Signed-off-by: Alice <alice@example.com>\n')
        self.assertEqual(cleanup.clean_message(message), expected)
        self.assertEqual(cleanup.clean_message(expected), expected)
        self.assertEqual(cleanup.clean_message(b'Co-authored-by: Bot <noreply@anthropic.com>'), b'')

    def test_changed_commits_drop_invalid_signatures_and_preserve_other_headers(self):
        headers = (b'tree tree-id\nauthor Alice <a@example.com> 123 +0800\n'
                   b'committer Bob <b@example.com> 456 -0700\nencoding ISO-8859-1\n')
        signatures = (b'gpgsig -----BEGIN SIGNATURE-----\n payload\n end\n'
                      b'gpgsig-sha256 signature\n payload\n'
                      b'mergetag signed-tag\n payload\n')
        raw = headers + signatures + b'\ncaf\xe9\n'
        self.assertEqual(cleanup.rewrite_commit(raw, {}), raw)
        dirty = raw + b'Co-authored-by: Claude <noreply@anthropic.com>\n'
        self.assertEqual(cleanup.rewrite_commit(dirty, {}), headers + b'\ncaf\xe9\n')
        child = b'parent old\n' + raw
        self.assertEqual(cleanup.rewrite_commit(child, {'old': 'new'}),
                         b'parent new\n' + headers + b'\ncaf\xe9\n')

    def test_rewrites_merged_history_without_moving_refs_or_changing_trees(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)

            def git(*args, data=None):
                return subprocess.check_output(['git', '-C', str(root), *args], input=data)

            git('init', '-q', '-b', 'main')
            git('config', 'user.name', 'Human Author')
            git('config', 'user.email', 'human@example.com')
            git('config', 'commit.gpgsign', 'false')
            git('config', 'core.hooksPath', '/dev/null')
            (root / 'file').write_text('unchanged upstream\n')
            git('add', 'file')
            git('commit', '-qm', 'Clean upstream')
            upstream = git('rev-parse', 'HEAD').strip()
            git('checkout', '-qb', 'feature')
            (root / 'feature').write_text('feature\n')
            git('add', 'feature')
            git('commit', '-qm', 'Feature\n\nCo-Authored-By: Claude Opus <noreply@anthropic.com>\n'
                'Co-authored-by: Alice <alice@example.com>')
            git('checkout', '-q', 'main')
            git('commit', '--allow-empty', '-qm', 'Parallel work')
            git('merge', '--no-ff', '-qm', 'Merge feature', 'feature')
            original = git('rev-parse', 'HEAD').strip()
            before = git('rev-list', '--reverse', '--topo-order', original.decode()).splitlines()

            def run(tip):
                return subprocess.check_output([sys.executable, str(SCRIPT), tip.decode()], cwd=root).strip()

            cleaned = run(original)
            self.assertNotEqual(cleaned, original)
            self.assertEqual(git('rev-parse', 'main').strip(), original)
            after = git('rev-list', '--reverse', '--topo-order', cleaned.decode()).splitlines()
            self.assertEqual(len(before), len(after))
            self.assertEqual(after[0], upstream)
            for old, new in zip(before, after):
                for fmt in ('%T', '%an <%ae> %at %ai', '%cn <%ce> %ct %ci'):
                    self.assertEqual(git('show', '-s', '--format=' + fmt, old.decode()),
                                     git('show', '-s', '--format=' + fmt, new.decode()))
                self.assertEqual(len(git('show', '-s', '--format=%P', old.decode()).split()),
                                 len(git('show', '-s', '--format=%P', new.decode()).split()))
            log = git('log', '--format=%B', cleaned.decode())
            self.assertNotIn(b'Co-Authored-By: Claude', log)
            self.assertIn(b'Co-authored-by: Alice <alice@example.com>', log)
            self.assertEqual(run(cleaned), cleaned)
            git('fsck', '--no-reflogs')

            remote = root / 'remote.git'
            git('init', '--bare', '-q', str(remote))
            git('remote', 'add', 'origin', str(remote))
            git('push', '-q', 'origin', 'main')
            git('commit', '--allow-empty', '-qm', 'Concurrent merge')
            git('push', '-q', 'origin', 'main')
            concurrent = git('rev-parse', 'main').strip()
            rejected = subprocess.run(
                ['git', '-C', str(root), 'push',
                 '--force-with-lease=refs/heads/main:' + original.decode(),
                 'origin', cleaned.decode() + ':refs/heads/main'], capture_output=True)
            self.assertNotEqual(rejected.returncode, 0)
            self.assertEqual(git('ls-remote', 'origin', 'refs/heads/main').split()[0], concurrent)

            (root / '.git/shallow').write_bytes(upstream + b'\n')
            result = subprocess.run([sys.executable, str(SCRIPT), original.decode()], cwd=root,
                                    capture_output=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn(b'Refusing to rewrite a shallow repository', result.stderr)

    def test_workflow_runs_after_merges_with_authorized_token_and_explicit_lease(self):
        workflow = (ROOT / '.github/workflows/remove-claude-authorship.yml').read_text()
        self.assertIn('  push:\n    branches: [main]\n', workflow)
        self.assertIn('  workflow_dispatch:', workflow)
        self.assertIn("if: github.ref == 'refs/heads/main'", workflow)
        self.assertIn('cancel-in-progress: false', workflow)
        self.assertIn('fetch-depth: 0', workflow)
        self.assertIn('token: ${{ secrets.CLAUDE_CLEANUP_TOKEN }}', workflow)
        self.assertIn('python3 scripts/next_remove_claude_authorship.py "$before"', workflow)
        self.assertIn('git diff --exit-code "$before^{tree}" "$after^{tree}"', workflow)
        self.assertIn('git push --force-with-lease="refs/heads/main:$before" origin "$after:refs/heads/main"', workflow)
        self.assertNotIn('pull_request_target:', workflow)


if __name__ == '__main__':
    unittest.main()
