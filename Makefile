BIN_DIR := "./bin"

base_dir :=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
pwd = $(shell pwd)
local_name = service-datamonkey
tag ?= latest
api_version ?= 1.0.0

C_BLUE := "\\033[94m"
C_NONE := "\\033[0m"
C_CYAN := "\\033[36m"

.PHONY: default
default:
	@echo ""
	@echo "Please choose one of:"
	@echo ""
	@echo "$(C_CYAN)  ####### Project Management #######$(C_NONE)"
	@echo ""
	@echo "$(C_BLUE)    make install$(C_NONE)"
	@echo "      because dependencies matter"
	@echo ""
	@echo "$(C_BLUE)    make update$(C_NONE)"
	@echo "      pulls down openapi spec and generates code"
	@echo ""
	@echo "$(C_CYAN)  ####### Build #######$(C_NONE)"
	@echo ""
	@echo "$(C_BLUE)    make build$(C_NONE)"
	@echo "      builds just the service-datamonkey container"
	@echo ""
	@echo "$(C_CYAN)  ####### Run #######$(C_NONE)"
	@echo ""
	@echo "$(C_BLUE)    make start$(C_NONE)"
	@echo "      alias to docker compose up, starts all relevant services"
	@echo ""
	@echo "$(C_BLUE)    make stop$(C_NONE)"
	@echo "      alias to docker compose down, stops all relevant services"
	@echo ""
	@echo "$(C_CYAN)  ####### Slurm Testing #######$(C_NONE)"
	@echo ""
	@echo "$(C_BLUE)    make start-slurm-rest$(C_NONE)"
	@echo "      start service-datamonkey with service-slurm in REST mode"
	@echo ""
	@echo "$(C_BLUE)    make start-slurm-cli$(C_NONE)"
	@echo "      start service-datamonkey with service-slurm in CLI mode"
	@echo ""
	@echo "$(C_BLUE)    make test-slurm-modes$(C_NONE)"
	@echo "      test both REST and CLI modes"
	@echo ""


.PHONY: install
install:
	@$(BIN_DIR)/lib.sh "manageDeps"


.PHONY: update
update:
	@$(BIN_DIR)/lib.sh "getApiSpec"
	@$(BIN_DIR)/lib.sh "generateServer"


.PHONY: build
build:
	@echo Building $(local_name):$(tag)
	@docker build -t $(local_name):$(tag) . --no-cache


.PHONY: start
start:
	@docker compose up -d --force-recreate datamonkey

.PHONY: stop
stop:
	@docker compose down


# Slurm testing targets
.PHONY: start-slurm-rest
start-slurm-rest:
	@echo "Starting service-datamonkey with service-slurm in REST mode..."
	@./bin/switch-slurm-mode.sh rest

.PHONY: start-slurm-cli
start-slurm-cli:
	@echo "Starting service-datamonkey with service-slurm in CLI mode..."
	@./bin/switch-slurm-mode.sh cli

.PHONY: test-slurm-modes
test-slurm-modes:
	@echo "Testing both REST and CLI modes..."
	@./bin/test-slurm-modes.sh
