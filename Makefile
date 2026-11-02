MODULES := . grpc middleware/gin
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
GOVULNCHECK := go run golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: fmt-check vet lint vuln test tidy-check ci

fmt-check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

vet:
	@for m in $(MODULES); do (cd $$m && go vet ./...) || exit 1; done

lint:
	@for m in $(MODULES); do (cd $$m && $(GOLANGCI_LINT) run ./...) || exit 1; done

vuln:
	@for m in $(MODULES); do (cd $$m && $(GOVULNCHECK) ./...) || exit 1; done

test:
	@for m in $(MODULES); do (cd $$m && go test -race ./...) || exit 1; done

tidy-check:
	@for m in $(MODULES); do (cd $$m && go mod tidy && git diff --exit-code -- go.mod go.sum) || exit 1; done

ci: fmt-check vet lint vuln test tidy-check
