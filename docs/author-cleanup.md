# Claude co-author cleanup

The `Remove Claude co-authorship` workflow runs after pushes to `main`, including
PR merges. Run it manually on `main` to clean existing history. A run reads the
latest `main`, removes Claude `Co-Authored-By` lines and the known Claude
`noreply@anthropic.com` / `claude@anthropic.com` identities, and rewrites affected
commits and their descendants. Human authors, committers, dates, trees and merge
parents are preserved. Other co-authors and ordinary mentions of Claude remain.

## Authorization

Set the repository Actions secret `CLAUDE_CLEANUP_TOKEN` to a dedicated bot token
with repository contents write access. That bot must be authorized to bypass
the applicable PR requirements and force-push protection on `main`. The workflow
does not change repository protection settings. The default `GITHUB_TOKEN`
cannot bypass the current protection. Missing authorization fails the run;
it does not report a successful cleanup.

Authorize the bot before merging this workflow, then use Actions → Remove Claude
co-authorship → Run workflow on `main` for a manual cleanup. Every later merge
triggers cleanup automatically. A bot-token push may trigger one further run,
which exits without pushing when there is nothing left to remove.

## Effects on history

Commit IDs change from the first affected commit onward. Signatures and signed
merge tags attached to changed commits are removed because they no longer
verify. Existing clones and open PR branches need to rebase onto the cleaned
`main`; merging an old branch can reintroduce the old history until the next
cleanup. Tags and other branches are not rewritten, and GitHub's cached
contributor display may take time to refresh or still include attribution
reachable through those refs.

Runs serialize, and the push uses an explicit force-with-lease. If another push
changes `main` during cleanup, the lease rejects this run; rerun against the new
tip. The Actions summary records the old and new tips for recovery. File
contents, including sealed evidence and baselines, remain byte-identical.

For local inspection, `python3 scripts/next_remove_claude_authorship.py main`
prints the cleaned tip and writes Git objects without moving any branch or
pushing. Compare it with `main` using `git log` and `git diff` before publishing.
