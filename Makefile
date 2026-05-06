# Force Make to use Bash so our interactive prompts work on Ubuntu
SHELL := /bin/bash

# ==========================================================
# INTERNAL DEVELOPER WORKFLOW (For Homelab / Gitea Setup)
# ==========================================================
.PHONY: save release

# Use this 100 times a day. It asks for a commit message and sends it to Gitea.
save:
	@echo "🛠️ Saving to Local Dev Environment..."
	@read -p "Enter commit message (or press Enter for default): " msg; \
	if [ -z "$$msg" ]; then msg="dev: auto-save update"; fi; \
	git add .; \
	git commit -m "$$msg" || true; \
	git push origin dev

# Use this once a week. It switches to main, merges your work, and publishes to GitHub.
release:
	@echo "🚀 Publishing to Public GitHub..."
	git checkout main
	git merge dev --squash -m "Release: New features and sensor updates"
	git push origin main
	git push public main
	git checkout dev
