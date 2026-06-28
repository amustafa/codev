# Fork management — keeps amustafa-main rebased on upstream/main
# and provides helpers for upstream PRs and cherry-picks.

.PHONY: sync rebase upstream-pr cherry-pick-upstream status

# Fetch upstream and rebase amustafa-main on top of upstream/main.
# Run this regularly to stay current.
sync:
	git fetch upstream
	git checkout amustafa-main
	git rebase upstream/main
	@echo "✓ amustafa-main rebased on upstream/main"

# Same as sync but also force-pushes (needed after rebase rewrites history).
# Only use this when you're sure nobody else is working off your branch.
rebase:
	git fetch upstream
	git checkout amustafa-main
	git rebase upstream/main
	git push origin amustafa-main --force-with-lease
	@echo "✓ amustafa-main rebased and pushed"

# Also sync local main to track upstream exactly.
sync-main:
	git fetch upstream
	git checkout main
	git reset --hard upstream/main
	git push origin main
	@echo "✓ main synced to upstream/main"

# Create a branch off upstream/main for an upstream PR.
# Usage: make upstream-pr BRANCH=feat/my-upstream-change
upstream-pr:
	@test -n "$(BRANCH)" || (echo "Usage: make upstream-pr BRANCH=feat/my-change" && exit 1)
	git fetch upstream
	git checkout -b $(BRANCH) upstream/main
	@echo "✓ Branch $(BRANCH) created from upstream/main"
	@echo "  Push with: git push -u origin $(BRANCH)"
	@echo "  PR with:   gh pr create --repo cluesmith/codev"

# Cherry-pick a commit from upstream onto amustafa-main.
# Usage: make cherry-pick-upstream COMMIT=abc123
cherry-pick-upstream:
	@test -n "$(COMMIT)" || (echo "Usage: make cherry-pick-upstream COMMIT=abc123" && exit 1)
	git checkout amustafa-main
	git cherry-pick $(COMMIT)
	@echo "✓ Cherry-picked $(COMMIT) onto amustafa-main"

# Show divergence between amustafa-main and upstream/main.
status:
	@git fetch upstream --quiet
	@echo "=== Commits on amustafa-main not in upstream/main ==="
	@git log --oneline upstream/main..amustafa-main
	@echo ""
	@echo "=== Commits on upstream/main not in amustafa-main ==="
	@git log --oneline amustafa-main..upstream/main | head -20
	@echo "(showing max 20)"
