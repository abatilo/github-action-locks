SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.ONESHELL:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

.PHONY: help
help: ## View help information
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: bootstrap
bootstrap:
	asdf plugin-add golang https://github.com/kennyp/asdf-golang.git || true
	asdf install

.PHONY: clean
clean: ## Delete local dev environment
	rm -rf dist

.PHONY: build
build: bootstrap ## Build binary for distribution
	mkdir -p dist/
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -ldflags="-w -s" -o dist/github-action-locks main.go
