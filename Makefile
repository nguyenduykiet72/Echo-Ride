#PROTO_DIR := proto
#PROTO_SRC := $(wildcard $(PROTO_DIR)/*.proto)
#GO_OUT := .
#
#.PHONY: generate-proto
#generate-proto:
#	protoc \
#		--proto_path=$(PROTO_DIR) \
#		--go_out=$(GO_OUT) \
#		--go-grpc_out=$(GO_OUT) \
#		$(PROTO_SRC)
#

SERVICES=ride-service auth-service location-service payment-service

.PHONY: help

help:
	@echo "Usage: make [command] [service=<name>]"
	@echo ""
	@echo "  run service=ride-service     Run một service"
	@echo "  dev service=ride-service     Hot reload một service"
	@echo "  build-all                    Build tất cả service"
	@echo "  migrate-up service=ride-service"
	@echo "  docker-up / docker-down"

# Delegate xuống service cụ thể
.PHONY: run dev build migrate-up migrate-down migrate-status migrate-create

run dev build migrate-up migrate-down migrate-status migrate-reset:
	@[ "$(service)" ] || (echo "Usage: make $@ service=<service-name>"; exit 1)
	@$(MAKE) -C services/$(service) $@ name=$(name)

# Build tất cả
.PHONY: build-all
build-all:
	@for svc in $(SERVICES); do \
		echo "Building $$svc..."; \
		$(MAKE) -C services/$$svc build; \
	done

# Docker
.PHONY: docker-up docker-down docker-logs
docker-up:
	@docker-compose -f infra/development/docker/docker-compose.dev.yml up -d

docker-down:
	@docker-compose -f infra/development/docker/docker-compose.dev.yml down

docker-logs:
	@docker-compose -f infra/development/docker/docker-compose.dev.yml logs -f

# From root
#make run service=ride-service
#make migrate-up service=ride-service
#make migrate-create service=ride-service name=add_driver_table
#make build-all
#
## From service
#cd services/ride-service
#make dev
#make migrate-up