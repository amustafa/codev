# Fork management — keeps main rebased on upstream/main
# and provides helpers for upstream PRs and cherry-picks.

.PHONY: sync rebase upstream-pr cherry-pick-upstream status

# Fetch upstream and rebase main on top of upstream/main.
# Run this regularly to stay current.
sync:
	git fetch upstream
	git checkout main
	git rebase upstream/main
	@echo "✓ main rebased on upstream/main"

# Same as sync but also force-pushes (needed after rebase rewrites history).
# Only use this when you're sure nobody else is working off your branch.
rebase:
	git fetch upstream
	git checkout main
	git rebase upstream/main
	git push origin main --force-with-lease
	@echo "✓ main rebased and pushed"

# Create a branch off upstream/main for an upstream PR.
# Usage: make upstream-pr BRANCH=feat/my-upstream-change
upstream-pr:
	@test -n "$(BRANCH)" || (echo "Usage: make upstream-pr BRANCH=feat/my-change" && exit 1)
	git fetch upstream
	git checkout -b $(BRANCH) upstream/main
	@echo "✓ Branch $(BRANCH) created from upstream/main"
	@echo "  Push with: git push -u origin $(BRANCH)"
	@echo "  PR with:   gh pr create --repo cluesmith/codev"

# Cherry-pick a commit from upstream onto main.
# Usage: make cherry-pick-upstream COMMIT=abc123
cherry-pick-upstream:
	@test -n "$(COMMIT)" || (echo "Usage: make cherry-pick-upstream COMMIT=abc123" && exit 1)
	git checkout main
	git cherry-pick $(COMMIT)
	@echo "✓ Cherry-picked $(COMMIT) onto main"

# Show divergence between main and upstream/main.
status:
	@git fetch upstream --quiet
	@echo "=== Commits on main not in upstream/main ==="
	@git log --oneline upstream/main..main
	@echo ""
	@echo "=== Commits on upstream/main not in main ==="
	@git log --oneline main..upstream/main | head -20
	@echo "(showing max 20)"
