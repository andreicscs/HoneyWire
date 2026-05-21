.PHONY: build test test-gateway test-backend test-chaos

-include .env

GATEWAY_IP ?= 127.0.0.1
BACKEND_IP ?= 127.0.0.1
CHAOS_IP   ?= 127.0.0.1

# URL remote nodes use
HUB_URL ?= http://localhost:8080

# Local API URL
LOCAL_HUB_URL ?= http://127.0.0.1:8080

# Dashboard password for provisioning
HUB_PASSWORD ?= defaultpass

ENV ?= dev

ifeq ($(ENV),public)
	MANIFEST_FILE = ../Sensors/official/manifests.json
else
	MANIFEST_FILE = ../Sensors/official/manifests.dev.json
endif

build:
	mkdir -p build
	GOOS=linux GOARCH=amd64 go build -o build/wizard ./cmd/wizard

test:
	go test ./...

define deploy_target
	@echo "[*] TARGETING $(1) ($(ENV))"

	scp build/wizard root@$($(2)):/root/wizard
	scp $(MANIFEST_FILE) root@$($(2)):/root/manifests.json

	ssh root@$($(2)) '\
		chmod +x /root/wizard && \
		rm -f /etc/honeywire/config.json && \
		HW_DASHBOARD_PASSWORD="$(HUB_PASSWORD)" \
		/root/wizard --link $(HUB_URL) \
	'
endef

.ONESHELL:
test-gateway: build
	$(call deploy_target,gateway,GATEWAY_IP)

.ONESHELL:
test-backend: build
	$(call deploy_target,backend,BACKEND_IP)

.ONESHELL:
test-chaos: build
	$(call deploy_target,chaos,CHAOS_IP)