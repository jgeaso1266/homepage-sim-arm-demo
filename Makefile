.PHONY: help bake dev build test test-web e2e check

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

bake: ## Plan both arms and write static trajectory assets (web/static/trajectories)
	go run ./cmd/bake

dev: ## Run the web dev server (http://localhost:5173)
	pnpm -C web dev

build: ## Build the static web app to web/build
	pnpm -C web build

test: ## Run Go tests (planning, scene, baker)
	go test ./...

test-web: ## Run web unit tests (vitest)
	pnpm -C web test

e2e: ## Run Playwright end-to-end tests against the production build
	pnpm -C web test:e2e

check: test test-web ## Run Go + web unit tests
