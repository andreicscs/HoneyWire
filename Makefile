.PHONY: tag-sensors

# Usage: make tag-sensors VERSION=v2.1.0
tag-sensors:
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ Error: VERSION is not set. Usage: make tag-sensors VERSION=v2.1.0"; \
		exit 1; \
	fi
	@$(eval CLEAN_VERSION := $(patsubst v%,%,$(VERSION)))
	@echo "🏷️  Tagging all official sensors with v$(CLEAN_VERSION)..."
	@for f in $$(find Sensors/official -mindepth 2 -maxdepth 2 -name "*.json" 2>/dev/null); do \
		id=$$(jq -r '.id // empty' "$$f" 2>/dev/null); \
		if [ -n "$$id" ] && [ "$$id" != "null" ]; then \
			sensor_name=$${id#hw-sensor-}; \
			tag="sensor/$$sensor_name/v$(CLEAN_VERSION)"; \
			echo "   Creating tag $$tag..."; \
			git tag "$$tag" 2>/dev/null || echo "   ⚠️ Tag $$tag already exists, skipping."; \
		fi; \
	done
	@echo ""
	@echo "✅ All sensor tags created locally."
	@echo "🚀 Run 'git push origin --tags' to push them to the server and trigger the CI pipelines."
