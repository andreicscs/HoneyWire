.PHONY: build test-gateway test-backend test-chaos fetch-token

# Load local secrets quietly (the '-' means don't throw an error if .env is missing)
-include .env

GATEWAY_IP?=127.0.0.1
BACKEND_IP?=127.0.0.1
CHAOS_IP?=127.0.0.1

# The URL the *remote VMs* will use to reach your Hub
HUB_URL?=http://localhost:8080

# The URL your local Makefile uses to hit the Hub API
LOCAL_HUB_URL?=http://127.0.0.1:8080

# Your dev dashboard password
HUB_PASSWORD?=defualtpass

# --- ENVIRONMENT SWITCH ---
# Defaults to 'dev'. Override by running: make test-gateway ENV=public
ENV ?= dev

ifeq ($(ENV),public)
	MANIFEST_FILE = ../Sensors/official/manifests.json
else
	MANIFEST_FILE = ../Sensors/official/manifests.dev.json
endif

build:
	GOOS=linux GOARCH=amd64 go build -o build/wizard .

# Function to fetch a fresh token (always runs, never cached)
define fetch_token
	@echo "[*] TARGETING $(ENV) ENVIRONMENT using $(MANIFEST_FILE)"
	@echo "[*] Negotiating fresh token with Hub API..."
	@curl -s -c cookie.txt -X POST $(LOCAL_HUB_URL)/login -d '{"password":"$(HUB_PASSWORD)"}' > /dev/null
	@curl -s -b cookie.txt -X POST $(LOCAL_HUB_URL)/api/v1/tokens/generate | grep -o '"token":"[^"]*' | cut -d'"' -f4 > .token
	@rm -f cookie.txt
	@if [ ! -s .token ] || [ -z "$$(cat .token)" ]; then \
		echo "❌ Failed to get token! Check your HUB_PASSWORD and ensure the Hub is running."; \
		rm -f .token; \
		exit 1; \
	fi
	@echo "[*] Token acquired: $$(cat .token)"
endef

.ONESHELL:
test-gateway: build
	$(call fetch_token)
	TOKEN=$$(cat .token); \
	trap 'rm -f .token' EXIT; \
	scp build/wizard root@$(GATEWAY_IP):/root/wizard && \
	scp $(MANIFEST_FILE) root@$(GATEWAY_IP):/root/manifests.json && \
	ssh root@$(GATEWAY_IP) "chmod +x /root/wizard && rm -f /etc/honeywire/config.json && /root/wizard --link $(HUB_URL) $$TOKEN && /root/wizard --registry /root/manifests.json"

.ONESHELL:
test-backend: build
	$(call fetch_token)
	TOKEN=$$(cat .token); \
	trap 'rm -f .token' EXIT; \
	scp build/wizard root@$(BACKEND_IP):/root/wizard && \
	scp $(MANIFEST_FILE) root@$(BACKEND_IP):/root/manifests.json && \
	ssh root@$(BACKEND_IP) "chmod +x /root/wizard && rm -f /etc/honeywire/config.json && /root/wizard --link $(HUB_URL) $$TOKEN && /root/wizard --registry /root/manifests.json"

.ONESHELL:
test-chaos: build
	$(call fetch_token)
	TOKEN=$$(cat .token); \
	trap 'rm -f .token' EXIT; \
	scp build/wizard root@$(CHAOS_IP):/root/wizard && \
	scp $(MANIFEST_FILE) root@$(CHAOS_IP):/root/manifests.json && \
	ssh root@$(CHAOS_IP) "chmod +x /root/wizard && rm -f /etc/honeywire/config.json && /root/wizard --link $(HUB_URL) $$TOKEN && /root/wizard --registry /root/manifests.json"
