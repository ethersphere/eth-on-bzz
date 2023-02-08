GO ?= go
GOBIN ?= $$($(GO) env GOPATH)/bin
GOLANGCI_LINT ?= $(GOBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v1.50.0

.PHONY: get-golangcilint
get-golangcilint:
	test -f $(GOLANGCI_LINT) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$($(GO) env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

# Runs lint on entire repo
.PHONY: lint
lint: get-golangcilint
	$(GOLANGCI_LINT) run ./...

# Runs tests on entire repo
.PHONY: test
test: 
	go test  ./... -race -failfast -timeout=5s -count=50 

# Runs tests with integration components on entire repo
.PHONY: test-integration
test-integration: 	
	go test ./... -race -failfast -tags=integration -count=1
	
# Code tidy
.PHONY: tidy
tidy:
	go mod tidy
	go fmt ./...
