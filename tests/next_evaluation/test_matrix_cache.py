# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Cross-output cache controls; no models, credentials, or host builds."""
import copy
import fcntl
import io
import json
from pathlib import Path
import shutil
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'evaluation'))
import run_matrix as matrix


class MatrixCacheTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.source = self.root / 'previous-output'
        self.output = self.root / 'new-output'
        self.output.mkdir()
        self.env = {'model': 'gpt-5.6-luna', 'reasoning': 'medium', 'guidance_delivery': 'file', 'agent_seconds': 180}
        self.batch = matrix.read(ROOT / 'evidence/evaluation/token-efficiency-09/docs/batch-01.json')
        self.job = next(j for j in matrix.jobs(self.env) if j['suite'] == 'docs-hard')
        self.job['expected_arms'] = self.batch['arms']

    def folder(self, output=None):
        return (output or self.source) / 'jobs' / self.job['key']

    def batch_path(self, output=None):
        return self.folder(output) / self.job['section'] / 'batch-01.json'

    def cache(self, output=None, batch=None):
        output = output or self.source
        matrix.write(output / 'environment.json', self.env)
        matrix.write(output / 'plan.json', {'jobs': [self.job]})
        matrix.write(self.batch_path(output), self.batch if batch is None else batch)
        matrix.seal_batch(self.folder(output), self.batch_path(output))

    def import_cache(self):
        matrix.import_sibling_jobs(self.output, [self.job], self.env)

    def invoke(self):
        config = self.root / 'paths.json'
        matrix.write(config, {})
        args = ['run_matrix', '--config', str(config), '--output', str(self.output), '--execute']
        with patch.object(sys, 'argv', args), patch.object(matrix, 'environment', return_value=(self.env, {})), \
                patch.object(matrix, 'jobs', return_value=[copy.deepcopy(self.job)]), patch.object(matrix, 'bind_jobs'), \
                patch.object(matrix.subprocess, 'run') as run, patch.object(sys, 'stdout', io.StringIO()):
            try:
                return matrix.main()
            finally:
                run.assert_not_called()

    def test_new_output_reuses_exact_whole_job_and_all_failed_costs_without_dispatch(self):
        self.cache()
        original = self.batch_path().read_bytes()
        prior = {'kind': 'historical_reuse', 'condition': 9, 'batch_sha256': matrix.sha(self.batch_path())}
        matrix.write(self.folder() / 'reuse.json', prior)
        (self.folder() / 'STOP').symlink_to(self.source / 'STOP')
        (self.folder() / 'docs/batch-01.json.lock').touch()
        self.assertEqual(self.invoke(), 0)
        self.assertEqual(self.batch_path(self.output).read_bytes(), original)
        self.assertEqual(self.batch_path().read_bytes(), original)
        copied = self.folder(self.output)
        self.assertFalse((copied / 'STOP').exists())
        self.assertFalse((copied / 'STOP').is_symlink())
        self.assertFalse((copied / 'docs/batch-01.json.lock').exists())
        provenance = matrix.read(copied / 'reuse.json')
        self.assertEqual(provenance['sources'][0]['prior_reuse'], prior)
        summary = matrix.read(self.output / 'summary.json')
        self.assertEqual(summary['fresh_known_tokens'], 0)
        self.assertEqual(summary['reused_known_tokens'], 901686)
        self.assertEqual(summary['completed_attempts'], 18)
        self.assertIsNone(next(t for t in summary['tasks'] if t['task'] == 'docs-hard-multi-edit')['saving_fraction'])
        self.assertEqual(self.invoke(), 0)  # Same-output cached behavior remains a no-op.

    def test_matching_partial_inflight_and_unsealed_jobs_block_import(self):
        for fault in ('partial', 'missing_batch', 'missing_seal', 'seal_changed', 'inflight'):
            with self.subTest(fault=fault):
                shutil.rmtree(self.source, ignore_errors=True)
                self.cache()
                if fault == 'partial':
                    batch = copy.deepcopy(self.batch)
                    batch.update(runs=batch['runs'][:2], completed=False)
                    matrix.write(self.batch_path(), batch)
                elif fault == 'missing_batch': self.batch_path().unlink()
                elif fault == 'missing_seal': (self.folder() / 'completed.json').unlink()
                elif fault == 'seal_changed': matrix.write(self.folder() / 'completed.json', {'batch_sha256': 'wrong'})
                elif fault == 'inflight': matrix.write(self.folder() / 'docs/batch-01.json.inflight.json', {})
                with self.assertRaises(ValueError): self.import_cache()
                self.assertFalse(self.batch_path(self.output).exists())

    def test_failed_new_output_does_not_poison_original_cache_resume(self):
        partial = copy.deepcopy(self.batch)
        partial.update(runs=partial['runs'][:2], completed=False)
        self.cache(batch=partial)
        with self.assertRaises(ValueError):
            self.invoke()
        self.assertFalse((self.output / 'environment.json').exists())
        (self.folder() / 'completed.json').unlink()
        self.cache()  # The original serial job later finishes normally.
        self.output = self.source
        self.assertEqual(self.invoke(), 0)

    def test_conflicting_complete_copies_are_not_selected_by_success(self):
        self.cache()
        changed = copy.deepcopy(self.batch)
        changed['runs'][0]['success'] = not changed['runs'][0]['success']
        self.cache(self.root / 'another-output', changed)
        with self.assertRaisesRegex(ValueError, 'Conflicting completed'):
            self.import_cache()
        self.assertFalse(self.batch_path(self.output).exists())

    def test_identical_complete_copies_retain_every_source(self):
        self.cache()
        self.cache(self.root / 'another-output')
        self.import_cache()
        provenance = matrix.read(self.folder(self.output) / 'reuse.json')
        self.assertEqual(len(provenance['sources']), 2)
        self.assertEqual(self.batch_path(self.output).read_bytes(), self.batch_path().read_bytes())

    def test_changed_environment_or_job_fingerprint_is_not_a_cache_hit(self):
        self.cache()
        matrix.write(self.source / 'environment.json', {**self.env, 'model': 'changed'})
        self.import_cache()
        self.assertFalse(self.batch_path(self.output).exists())
        matrix.write(self.source / 'environment.json', self.env)
        matrix.write(self.source / 'plan.json', {'jobs': [{**self.job, 'seed': self.job['seed'] + 1}]})
        with self.assertRaisesRegex(ValueError, 'fingerprint'):
            self.import_cache()
        matrix.write(self.source / 'plan.json', {'jobs': []})
        with self.assertRaisesRegex(ValueError, 'outside its retained plan'):
            self.import_cache()

    def test_unreadable_or_dangling_candidate_output_is_ignored(self):
        unreadable = self.root / 'unreadable-output'
        unreadable.mkdir()
        unreadable.chmod(0o000)
        dangling = self.root / 'dangling-output'
        dangling.symlink_to(self.root / 'missing')
        self.cache()
        self.import_cache()
        self.assertTrue(self.batch_path(self.output).is_file())

    def test_source_and_parent_locks_prevent_cross_output_races(self):
        self.cache()
        for path in (self.source / '.matrix.lock', self.root / '.matrix-cache.lock'):
            with self.subTest(lock=path.name), path.open('a') as lock:
                fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
                with self.assertRaises(BlockingIOError): self.invoke()
        self.assertFalse(self.batch_path(self.output).exists())

    def test_qualifications_and_diagnostic_links_survive_import_and_report(self):
        self.cache()
        qualification = '# Limits\n\nRead [diagnostics](read-regression-diagnostics.json) and [report](REPORT.md).\n'
        (self.source / 'QUALIFICATIONS.md').write_text(qualification)
        matrix.write(self.source / 'read-regression-diagnostics.json', {'finding': 'synthetic coverage limitation'})
        self.invoke()
        self.assertEqual((self.output / 'QUALIFICATIONS.md').read_text(), qualification)
        self.assertEqual((self.output / 'read-regression-diagnostics.json').read_bytes(),
                         (self.source / 'read-regression-diagnostics.json').read_bytes())
        report = (self.output / 'REPORT.md').read_text()
        self.assertIn('[Interpretation limits and qualifications](QUALIFICATIONS.md)', report)
        self.assertIn('Reused usage:', report)
        self.assertNotIn('Reused historical usage:', report)
        provenance = matrix.read(self.folder(self.output) / 'reuse.json')
        self.assertEqual(provenance['sources'][0]['qualifications']['QUALIFICATIONS.md'],
                         matrix.sha(self.output / 'QUALIFICATIONS.md'))

    def test_existing_local_completed_job_keeps_its_original_fresh_accounting(self):
        self.cache(self.output)
        self.invoke()
        summary = matrix.read(self.output / 'summary.json')
        self.assertEqual(summary['fresh_known_tokens'], 901686)
        self.assertEqual(summary['reused_known_tokens'], 0)

    def test_extension_hosts_are_copied_and_bound_without_rebuilding(self):
        source = self.root / 'frozen-source'
        entry = source / 'skills/lark-extensions/SKILL.md'
        entry.parent.mkdir(parents=True)
        entry.write_text('Unchanged extension guidance.')
        self.env.update(official={'cli_sha256': 'normal-official'},
                        next={'cli_sha256': 'normal-next', 'entry_sha256': matrix.sha(entry)})
        self.job = next(j for j in matrix.jobs(self.env) if j['section'] == 'extensions')
        arms = {}
        for arm in matrix.ORIGINAL_ARMS:
            binary = self.folder() / '.hosts' / arm / 'host'
            binary.parent.mkdir(parents=True)
            binary.write_bytes(('exact measured ' + arm + ' executable').encode())
            binary.chmod(0o755)
            (binary.parent / 'sha256').write_text(matrix.sha(binary))
            arms[arm] = {'cli_sha256': matrix.sha(binary), 'skill_sha256': matrix.sha(entry)}
        self.job['expected_arms'] = arms
        manifest = matrix.read(ROOT / 'evaluation' / matrix.SUITES[self.job['suite']][0])
        tasks = [t for t in manifest['tasks'] if t['section'] == 'extensions']
        rows = [{**copy.deepcopy(self.batch['runs'][0]), 'task_id': t['id'], 'section': t['section'], 'trial': trial, 'arm': arm}
                for t, trial, arm in matrix.schedule(tasks, 3, self.job['seed'], matrix.ORIGINAL_ARMS)]
        self.batch.update(runs=rows, arms=arms, seed=self.job['seed'], planned_runs=len(rows),
                          task_manifest_sha256=self.job['manifest_sha256'])
        self.cache()
        unbound = {key: value for key, value in self.job.items() if key != 'expected_arms'}
        import run_read_batch
        with patch.object(run_read_batch, 'extension_host', side_effect=AssertionError('must not rebuild')):
            matrix.import_sibling_jobs(self.output, [unbound], self.env)
            matrix.bind_jobs([unbound], self.env, {'official_source': source, 'next_source': source}, self.output)
        self.assertEqual(unbound['expected_arms'], arms)
        matrix.validate_batch(matrix.read(self.batch_path(self.output)), unbound, self.env)
        for arm in matrix.ORIGINAL_ARMS:
            old = self.folder() / '.hosts' / arm / 'host'
            new = self.folder(self.output) / '.hosts' / arm / 'host'
            self.assertEqual(new.read_bytes(), old.read_bytes())
            self.assertEqual(new.stat().st_mode, old.stat().st_mode)


if __name__ == '__main__':
    unittest.main()
