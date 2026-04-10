APP_NAME := xp_2077
EXTRACTOR_NAME := github_extract
BIN_DIR := bin
APP_BIN := $(BIN_DIR)/$(APP_NAME)
EXTRACTOR_BIN := $(BIN_DIR)/$(EXTRACTOR_NAME)

# Variables para ejecutar el extractor.
# El contexto GitHub (owner/repo/project) está fijo en código.
OUTPUT_DB ?= ./tmp/github_extract.db

.PHONY: help build build-app build-extractor run run-extractor test fmt clean

help:
	@echo "Targets disponibles:"
	@echo "  make build            - Compila app principal y extractor"
	@echo "  make build-app        - Compila solo la app principal ($(APP_BIN))"
	@echo "  make build-extractor  - Compila solo el extractor ($(EXTRACTOR_BIN))"
	@echo "  make run              - Ejecuta la app principal (go run .)"
	@echo "  make run-extractor    - Ejecuta el extractor con flags/env"
	@echo "  make test             - Corre tests"
	@echo "  make fmt              - Formatea codigo Go"
	@echo "  make clean            - Elimina artefactos generados"
	@echo ""
	@echo "Variables run-extractor:"
	@echo "  OUTPUT_DB=<ruta>"

build: build-app build-extractor

build-app:
	@mkdir -p "$(BIN_DIR)"
	go build -o "$(APP_BIN)" .

build-extractor:
	@mkdir -p "$(BIN_DIR)"
	go build -o "$(EXTRACTOR_BIN)" ./cmd/github_extract

run:
	go run .

run-extractor:
	go run ./cmd/github_extract \
		-owner "aleph-ri" \
		-repo "advance" \
		-project "12" \
		-db "$(OUTPUT_DB)"

test:
	go test ./...

fmt:
	go fmt ./...

clean:
	rm -rf "$(BIN_DIR)"
