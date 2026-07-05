HYPHAE_DIR := hyphae

.DEFAULT_GOAL := check

.PHONY: check analyze setup

check:
	@echo "==> Building..."
	@cd $(HYPHAE_DIR) && go build ./...

	@echo "==> Vetting..."
	@cd $(HYPHAE_DIR) && go vet ./...

	@echo "==> Testing..."
	@cd $(HYPHAE_DIR) && go test ./...

	@echo "==> All checks passed."

analyze:
	@echo "==> Running go vet..."
	@cd $(HYPHAE_DIR) && go vet ./...

	@echo "==> Running tests with race detector..."
	@cd $(HYPHAE_DIR) && go test -race ./...

	@echo "==> Generating coverage report..."
	@cd $(HYPHAE_DIR) && go test -coverprofile=coverage.out ./...
	@cd $(HYPHAE_DIR) && go tool cover -func=coverage.out

	@echo "==> Running golangci-lint..."
	@cd $(HYPHAE_DIR) && golangci-lint run ./...

	@echo "==> Running govulncheck..."
	@cd $(HYPHAE_DIR) && govulncheck ./...

	@echo "==> Analysis complete."

setup:
	@echo "==> Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

	@echo "==> Installing govulncheck..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest

	@echo "==> Done. Ensure $$(go env GOPATH)/bin is on your PATH."
