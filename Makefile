APP_NAME := xp_2077
BIN_DIR := bin
APP_BIN := $(BIN_DIR)/$(APP_NAME)

# La app extrae de GitHub al arrancar. El contexto (owner/repo/project) está fijo en código.
OUTPUT_DB ?= ./tmp/github_extract.db

.PHONY: help build run run-skip-extract test fmt clean

help:
	@echo "Targets disponibles:"
	@echo "  make build            - Compila la app ($(APP_BIN))"
	@echo "  make run              - Ejecuta la app (extrae de GitHub y muestra el ranking)"
	@echo "  make run-skip-extract - Ejecuta la app usando solo el SQLite existente"
	@echo "  make test             - Corre tests"
	@echo "  make fmt              - Formatea codigo Go"
	@echo "  make clean            - Elimina artefactos generados"
	@echo ""
	@echo "Variables:"
	@echo "  OUTPUT_DB=<ruta>   (default $(OUTPUT_DB))"
	@echo "  GITHUB_TOKEN=...   (requerida salvo con run-skip-extract)"

build:
	@mkdir -p "$(BIN_DIR)"
	go build -o "$(APP_BIN)" .

run:
	OUTPUT_DB="$(OUTPUT_DB)" go run .

run-skip-extract:
	OUTPUT_DB="$(OUTPUT_DB)" go run . -skip-extract

test:
	go test ./...

fmt:
	go fmt ./...

clean:
	rm -rf "$(BIN_DIR)"
