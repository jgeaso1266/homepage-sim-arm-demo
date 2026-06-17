.PHONY: help install run bake build test test-web e2e check

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

install: ## Install the web app's dependencies (one-time)
	pnpm -C web install

run: ## Build and serve the demo locally (prints a http://localhost URL)
	pnpm -C web build && pnpm -C web preview

bake: ## (Optional) re-plan both arms and rewrite the static trajectory assets — needs Go + rdk
	go run ./cmd/bake

build: ## Build the static web app to web/build
	pnpm -C web build

test: ## Run Go tests (planning, scene, baker)
	go test ./...

test-web: ## Run web unit tests (vitest)
	pnpm -C web test

e2e: ## Run Playwright end-to-end tests against the production build
	pnpm -C web test:e2e

check: test test-web ## Run Go + web unit tests
