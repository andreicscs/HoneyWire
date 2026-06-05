.PHONY: all build clean deploy test

-include .env

BINARY_NAME=honeywire
BUILD_DIR=build

ENV_IP ?= 127.0.0.1
ENV_PATH ?= /root

all: clean build

build:
	@echo "[*] Compiling HoneyWire for Linux (amd64 and arm64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-amd64 cmd/wizard/main.go
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-arm64 cmd/wizard/main.go
	@echo "[*] Generating checksums..."
	cd $(BUILD_DIR) && sha256sum $(BINARY_NAME)-* > checksums.txt

deploy: build
	@echo "[*] Deploying to root@$(ENV_IP):$(ENV_PATH)..."
	scp $(BUILD_DIR)/$(BINARY_NAME)-amd64 root@$(ENV_IP):$(ENV_PATH)
	scp $(BUILD_DIR)/$(BINARY_NAME)-arm64 root@$(ENV_IP):$(ENV_PATH)
	scp $(BUILD_DIR)/checksums.txt root@$(ENV_IP):$(ENV_PATH)

	@echo "✅ Deployment complete!"

test:
	@echo "[*] Running tests..."
	go test ./... -v

clean:
	@echo "[*] Cleaning build directory..."
	rm -rf $(BUILD_DIR)
