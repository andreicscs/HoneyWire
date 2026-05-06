# ==========================================================
# INTERNAL DEVELOPER WORKFLOW (For Homelab / Gitea Setup)
# Note: These commands require a dual-remote git setup.
# ==========================================================

# =========================
# DEVELOPER WORKFLOW
# =========================
.PHONY: save release

# Use this 100 times a day. It commits your messy code and sends it to Gitea.
save:
	@echo "🛠️ Saving to Local Dev Environment..."
	git add .
	git commit -m "dev: auto-save update" || true
	git push origin dev

# Use this once a week. It switches to main, merges your work, and publishes to GitHub.
release:
	@echo "🚀 Publishing to Public GitHub..."
	git checkout main
	git merge dev --squash -m "Release: New features and sensor updates"
	git push origin main
	git push public main
	git checkout dev
