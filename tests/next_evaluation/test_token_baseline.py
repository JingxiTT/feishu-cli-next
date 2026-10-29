# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Offline token arithmetic; the injected encoder is not a model measurement."""
import copy
import hashlib
import json
from pathlib import Path
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'evaluation'))
from harness import aggregate, performance_claims_supported
from token_baseline import measure, minimum_for, tooling_usage, validate
import token_baseline


class TokenBaselineTests(unittest.TestCase):
    def setUp(self):
        self.spec = {'model': 'test-model', 'encoding': 'test-encoding', 'tasks': {
            'a': {'input_texts': ['task goal', 'source data'],
                  'output_texts': ['answer', '文档'], 'rationale': 'Minimum task data and artifacts'}}}
        self.baseline = measure(self.spec, lambda text: list(text.encode()), 'test-version')
        self.minimum = minimum_for({'token_baseline': self.baseline}, 'a')

    def run_row(self, arm='official', **changes):
        return {'task_id': 'a', 'section': 'docs', 'trial': 0, 'arm': arm,
                'success': True, 'outcome_inconclusive': False, 'categories': [],
                'usage': {'input_tokens': 100, 'cached_input_tokens': 60, 'output_tokens': 40,
                          'total_tokens': 140}, 'minimum_tokens': self.minimum,
                'latency_seconds': 1, 'api_calls': 1, 'retries': 0, **changes}

    def test_counts_exact_input_and_output_text_separately_without_publishing_text(self):
        row = self.baseline['tasks']['a']
        self.assertEqual((row['input_tokens'], row['output_tokens']), (20, 12))
        self.assertEqual(row['output_sha256'][1], hashlib.sha256('文档'.encode()).hexdigest())
        self.assertNotIn('input_texts', row)
        self.assertNotIn('output_texts', row)
        self.assertNotIn('rationale', row)
        self.assertTrue(self.baseline['estimated'])
        self.assertIsNone(minimum_for({}, 'a'))

    def test_counter_cli_uses_pinned_encoder_and_refuses_to_overwrite(self):
        encoded = []
        def encode(text, **options):
            self.assertEqual(options, {'disallowed_special': ()})
            encoded.append(text)
            return list(text.encode())
        def get_encoding(name):
            self.assertEqual(name, 'test-encoding')
            return SimpleNamespace(encode=encode)
        tokenizer = SimpleNamespace(get_encoding=get_encoding, __version__='test-version')
        with tempfile.TemporaryDirectory() as directory:
            spec = Path(directory) / 'spec.json'
            output = Path(directory) / 'baseline.json'
            spec.write_text(json.dumps(self.spec))
            args = ['token_baseline.py', '--spec', str(spec), '--output', str(output)]
            with patch.dict(sys.modules, {'tiktoken': tokenizer}), patch.object(sys, 'argv', args):
                self.assertEqual(token_baseline.main(), 0)
                self.assertEqual(json.loads(output.read_text()), self.baseline)
                self.assertEqual(encoded, ['task goal', 'source data', 'answer', '文档'])
                original = output.read_bytes()
                with self.assertRaises(ValueError):
                    token_baseline.main()
                self.assertEqual(output.read_bytes(), original)

    def test_model_task_tokenizer_and_count_validation(self):
        for change in ({'model': 'other'}, {'method': 'observed_best_run'}, {'estimated': False},
                       {'tokenizer': {'encoding': 'test', 'library': 'tiktoken', 'version': ''}}):
            with self.subTest(change=change), self.assertRaises(ValueError):
                validate({**self.baseline, **change}, 'test-model')
        with self.assertRaises(ValueError):
            validate(self.baseline, 'test-model', ['missing-task'])
        for count in (-1, True, 0.5):
            baseline = copy.deepcopy(self.baseline)
            baseline['tasks']['a']['input_tokens'] = count
            with self.subTest(count=count), self.assertRaises(ValueError):
                validate(baseline, 'test-model')
        for texts in (None, '', [None], [''], []):
            spec = copy.deepcopy(self.spec)
            spec['tasks']['a']['input_texts'] = texts
            with self.subTest(texts=texts), self.assertRaises(ValueError):
                measure(spec, list, 'test-version')

    def test_input_and_output_subtracted_once_cache_already_included(self):
        self.assertEqual(tooling_usage(self.run_row()), {
            'input_tokens': 80, 'output_tokens': 28, 'total_tokens': 108})
        group = aggregate([self.run_row()])['groups'][0]
        self.assertEqual(group['known_total_tokens'], 140)
        self.assertEqual(group['minimum_input_tokens'], 20)
        self.assertEqual(group['minimum_output_tokens'], 12)
        self.assertEqual(group['tooling_input_tokens'], 80)
        self.assertEqual(group['tooling_output_tokens'], 28)
        self.assertEqual(group['tooling_tokens_per_success'], 108)

    def test_failures_remain_fully_charged_per_verified_completion(self):
        runs = [self.run_row(), self.run_row(trial=1, success=False)]
        group = aggregate(runs)['groups'][0]
        self.assertEqual(group['known_total_tokens'], 280)
        self.assertEqual(group['minimum_input_tokens'], 20)
        self.assertEqual(group['minimum_output_tokens'], 12)
        self.assertEqual(group['tooling_tokens_per_success'], 248)
        runs[0]['success'] = False
        self.assertIsNone(aggregate(runs)['groups'][0]['tooling_tokens_per_success'])

    def test_missing_incomplete_excessive_and_mismatched_baselines_have_no_score(self):
        for change in ({'minimum_tokens': None}, {'usage': None},
                       {'usage': {'input_tokens': 19, 'output_tokens': 40, 'total_tokens': 59}},
                       {'usage': {'input_tokens': 100, 'output_tokens': 11, 'total_tokens': 111}},
                       {'minimum_tokens': {**self.minimum, 'baseline_sha256': 'private text'}}):
            with self.subTest(change=change):
                self.assertIsNone(tooling_usage(self.run_row(**change)))
                self.assertIsNone(aggregate([self.run_row(**change)])['groups'][0]['tooling_tokens_per_success'])
        for change in ({'baseline_sha256': 'a' * 64}, {'input_tokens': 21}):
            runs = [self.run_row(), self.run_row('next', minimum_tokens={**self.minimum, **change})]
            result = aggregate(runs, ('official', 'next'))
            self.assertTrue(all(g['tooling_tokens_per_success'] is None for g in result['groups']))
            self.assertIsNone(result['matched_success'][0]['pairs'][0]['next_tooling_tokens'])

    def test_matched_pairs_keep_raw_counts_and_compare_tooling(self):
        runs = [self.run_row(), self.run_row('next', usage={
            'input_tokens': 60, 'output_tokens': 20, 'total_tokens': 80})]
        pair = aggregate(runs, ('official', 'next'))['matched_success'][0]['pairs'][0]
        self.assertEqual((pair['baseline_tokens'], pair['next_tokens']), (140, 80))
        self.assertEqual((pair['baseline_tooling_tokens'], pair['next_tooling_tokens']), (108, 48))

    def test_zero_tooling_usage_is_valid_and_no_output_can_have_zero_baseline(self):
        run = self.run_row(usage={'input_tokens': 20, 'output_tokens': 12, 'total_tokens': 32})
        self.assertEqual(tooling_usage(run)['total_tokens'], 0)
        self.assertEqual(aggregate([run])['groups'][0]['tooling_tokens_per_success'], 0)
        self.spec['tasks']['a']['output_texts'] = []
        self.assertEqual(measure(self.spec, list, 'test-version')['tasks']['a']['output_tokens'], 0)

    def test_claim_support_requires_complete_real_tooling_comparison(self):
        def evidence(runs, **changes):
            return {'mode': 'real', 'completed': True, **aggregate(runs, ('official', 'next')), **changes}
        runs = [self.run_row(), self.run_row('next')]
        self.assertTrue(performance_claims_supported(evidence(runs)))
        for changes in ({'completed': False}, {'mode': 'controlled'},
                        {'arm_selection': {'mode': 'candidate_only'}}):
            self.assertFalse(performance_claims_supported(evidence(runs, **changes)))
        self.assertFalse(performance_claims_supported(evidence([self.run_row('next')])))
        runs[0]['minimum_tokens'] = None
        self.assertFalse(performance_claims_supported(evidence(runs)))
        runs[0] = self.run_row(usage={'input_tokens': 20, 'output_tokens': 12, 'total_tokens': 32})
        self.assertFalse(performance_claims_supported(evidence(runs)))


if __name__ == '__main__':
    unittest.main()
