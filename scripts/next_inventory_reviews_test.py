import unittest
from next_inventory_reviews import effective_status, review_records


class ReviewCoverageTest(unittest.TestCase):
    def record(self, sha="old", verdict="approved_offline", verified=True):
        return {"path": "x", "review_sha256": sha, "review_revision": "commit", "verdict": verdict, "hash_verified": verified}

    def test_parent_approval_does_not_approve_unverdict_path(self):
        self.assertEqual(list(review_records({"verdict": "approved_offline", "paths": [{"path": "x", "review_sha256": "a", "review_revision": "r"}]})), [])

    def test_old_hash_never_approves_new_source(self):
        self.assertEqual(effective_status("new", [self.record()]), "changed_after_review")

    def test_closure_preserves_historical_failure(self):
        records = [self.record(verdict="changes_requested"), self.record("new")]
        self.assertEqual(effective_status("new", records), "recorded_approval_at_current_hash")
        self.assertEqual(effective_status("old", records), "changes_requested_at_current_hash")

    def test_conflicting_exact_verdict_stays_open(self):
        self.assertEqual(effective_status("old", [self.record(), self.record(verdict="changes_requested")]), "changes_requested_at_current_hash")

    def test_explicit_same_hash_dependency_closure_keeps_history(self):
        objection = {**self.record(verdict="changes_requested"), "findings": [{"id": "CR-1"}]}
        closure = {**self.record(), "prior_review_sha256": "old", "findings": [{"id": "CR-1", "status": "closed_at_review_revision"}]}
        self.assertEqual(effective_status("old", [objection, closure]), "recorded_approval_at_current_hash")
        self.assertEqual(objection["verdict"], "changes_requested")

    def test_explicit_closure_resolves_string_finding_references(self):
        objection = {**self.record(verdict="changes_requested"), "findings": ["CI1", "CI2"]}
        closure = {**self.record(), "prior_review_sha256": "old", "findings": [{"id": "CI1", "status": "closed_at_review_revision"}, {"id": "CI2", "status": "closed_at_review_revision"}]}
        self.assertEqual(effective_status("old", [objection, closure]), "recorded_approval_at_current_hash")
        closure["findings"].pop()
        self.assertEqual(effective_status("old", [objection, closure]), "changes_requested_at_current_hash")

    def test_invalid_hash_is_not_approval(self):
        self.assertEqual(effective_status("old", [self.record(verified=False)]), "changed_after_review")

    def test_nested_records_keep_reviewer_and_scope(self):
        value = {"reviewer": "independent", "reviews": [{"reviewed_scope": "narrow closure", "paths": [self.record()]}]}
        rows = list(review_records(value))
        self.assertEqual(len(rows), 1)
        self.assertEqual(rows[0]["reviewer"], "independent")
        self.assertEqual(rows[0]["reviewed_scope"], "narrow closure")
        self.assertEqual(rows[0]["pointer"], "/reviews/0/paths/0")


if __name__ == "__main__":
    unittest.main()
