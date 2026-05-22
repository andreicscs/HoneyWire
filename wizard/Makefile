.PHONY: build test test-gateway test-backend test-chaos
.PHONY: all build build-linux deploy test clean

-include .env
BINARY_NAME=wizard
BUILD_DIR=build

GATEWAY_IP ?= 127.0.0.1
BACKEND_IP ?= 127.0.0.1
CHAOS_IP   ?= 127.0.0.1
# Remote deployment defaults (override via `make deploy TARGET_IP=192.168.1.X`)
TARGET_USER?=root
TARGET_IP?=192.168.1.11
TARGET_PATH?=/root/wizard

# URL remote nodes use
HUB_URL ?= http://localhost:8080
all: clean build

# Local API URL
LOCAL_HUB_URL ?= http://127.0.0.1:8080
build:
	@echo "[*] Compiling Wizard locally..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/wizard/main.go

# Dashboard password for provisioning
HUB_PASSWORD ?= defaultpass
build-linux:
	@echo "[*] Compiling Wizard for Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)_linux cmd/wizard/main.go

ENV ?= dev
deploy: build-linux
	@echo "[*] Pushing Wizard to $(TARGET_USER)@$(TARGET_IP)..."
	scp $(BUILD_DIR)/$(BINARY_NAME)_linux $(TARGET_USER)@$(TARGET_IP):$(TARGET_PATH)
	@echo "✅ Deployment complete! You can now run $(TARGET_PATH) on the remote node."

build:
	mkdir -p build
	GOOS=linux GOARCH=amd64 go build -o build/wizard ./cmd/wizard

test:
	go test ./...
	@echo "[*] Running tests..."
	go test ./... -v

define deploy_target
	@echo "[*] TARGETING $(1) ($(ENV))"

	scp build/wizard root@$($(2)):/root/wizard

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
clean:
	@echo "[*] Cleaning build directory..."
	rm -rf $(BUILD_DIR)