BINARY     := bin/cora
CMD        := ./cmd/cora
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)
ARGS       := services list

# ── Build ─────────────────────────────────────────────────────────────────────

.PHONY: build
build:
	@mkdir -p $(dir $(BINARY))
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

.PHONY: build-prod
build-prod:
	@mkdir -p $(dir $(BINARY))
	CGO_ENABLED=0 go build -ldflags "-s -w $(LDFLAGS)" -o $(BINARY) $(CMD)

.PHONY: install
install:
	go install -ldflags "$(LDFLAGS)" $(CMD)

.PHONY: clean
clean:
	rm -rf bin/

# ── Test ──────────────────────────────────────────────────────────────────────

.PHONY: test
test:
	CGO_ENABLED=1 go test -race ./...

.PHONY: test-unit
test-unit:
	CGO_ENABLED=1 go test -race -short ./...

.PHONY: test-cover
test-cover:
	CGO_ENABLED=1 go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: test-cover-text
test-cover-text:
	CGO_ENABLED=1 go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# ── Lint & Format ─────────────────────────────────────────────────────────────

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: fmt
fmt:
	gofmt -w .
	goimports -w . 2>/dev/null || true

.PHONY: vet
vet:
	go vet ./...

# ── Dependencies ──────────────────────────────────────────────────────────────

.PHONY: deps
deps:
	go mod download

.PHONY: tidy
tidy:
	go mod tidy

# ── Docker ────────────────────────────────────────────────────────────────────

IMAGE ?= cora

.PHONY: docker-build
docker-build:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

.PHONY: docker-run
docker-run:
	docker run --rm \
		-v ./config.example.yaml:/root/.config/cora/config.yaml:ro \
		$(IMAGE):latest $(ARGS)

# ── Smoke Tests ───────────────────────────────────────────────────────────────

SMOKE_RUNNER := bin/smoke-runner

.PHONY: smoke-build
smoke-build:
	@mkdir -p bin
	go build -o $(SMOKE_RUNNER) ./cmd/smoke

.PHONY: smoke
smoke: build-prod smoke-build
	./$(SMOKE_RUNNER) \
		--cora-bin ./bin/cora \
		--config ./config/smoke-config.yaml \
		--scenarios-dir ./scenarios \
		--report-dir ./smoke-report \
		$(if $(VERBOSE),--verbose)
	@echo "Report: ./smoke-report/report.html"

.PHONY: smoke-filter
smoke-filter: build-prod smoke-build
	./$(SMOKE_RUNNER) \
		--cora-bin ./bin/cora \
		--config ./config/smoke-config.yaml \
		--scenarios-dir ./scenarios \
		--filter "$(FILTER)" \
		$(if $(VERBOSE),--verbose)

# ── Release ───────────────────────────────────────────────────────────────────

RELEASE_VERSION ?= $(VERSION)
DIST_DIR      := dist
RELEASE_DIR   := $(DIST_DIR)/cora-$(RELEASE_VERSION)
PLATFORMS     := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

.PHONY: release
release: clean build-prod
	@rm -rf $(DIST_DIR)
	@mkdir -p $(RELEASE_DIR)
	@# Copy source
	@cp -R go.mod go.sum Makefile cmd internal pkg assets config scenarios spec readme.md readme_en.md $(RELEASE_DIR)
	@rm -rf $(RELEASE_DIR)/config/config.yaml $(RELEASE_DIR)/config/smoke-config.yaml $(RELEASE_DIR)/config/views.yaml
	@rm -f $(RELEASE_DIR)/assets/img/cora.png
	@# Build per platform
	@set -e; for p in $(PLATFORMS); do \
		goos=$$(echo $$p | cut -d/ -f1); \
		goarch=$$(echo $$p | cut -d/ -f2); \
		out=$(DIST_DIR)/cora-$$goos-$$goarch; \
			if [ "$$goos" = "windows" ]; then out="$$out.exe"; fi; \
		CGO_ENABLED=0 GOOS=$$goos GOARCH=$$goarch go build -ldflags "$(LDFLAGS) -s -w" -o $$out $(CMD); \
		echo "  built: $$out"; \
	done
	@# Package source tarball (RELEASE_DIR has source only, no binaries).
	@echo "Creating source archive..."
	@tar -czf $(DIST_DIR)/cora-$(RELEASE_VERSION).src.tar.gz --exclude='.git' -C $(DIST_DIR) cora-$(RELEASE_VERSION)
	@# Package per-platform binary tarballs
	@set -e; for p in $(PLATFORMS); do \
		goos=$$(echo $$p | cut -d/ -f1); \
		goarch=$$(echo $$p | cut -d/ -f2); \
		binary=cora-$$goos-$$goarch; \
			if [ "$$goos" = "windows" ]; then binary="$$binary.exe"; fi; \
		tar -czf $(DIST_DIR)/cora-$(RELEASE_VERSION).$$goos-$$goarch.tar.gz -C $(DIST_DIR) $$binary; \
		echo "  archive: $(DIST_DIR)/cora-$(RELEASE_VERSION).$$goos-$$goarch.tar.gz"; \
	done
	@echo "Release $(RELEASE_VERSION) built in $(DIST_DIR)/"
	@ls -lh $(DIST_DIR)/*.tar.gz

# ── Help ──────────────────────────────────────────────────────────────────────

.PHONY: help
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Build:"
	@echo "  build          Build binary ($(BINARY))"
	@echo "  build-prod     Build optimised binary (CGO disabled, stripped)"
	@echo "  install        Install to GOPATH/bin"
	@echo "  clean          Remove build artefacts"
	@echo ""
	@echo "Test:"
	@echo "  test           Run all tests with race detector"
	@echo "  test-unit      Run short tests only (no integration)"
	@echo "  test-cover     Generate HTML coverage report"
	@echo "  test-cover-text Print coverage summary to stdout"
	@echo ""
	@echo "Quality:"
	@echo "  lint           Run golangci-lint"
	@echo "  fmt            Format code (gofmt + goimports)"
	@echo "  vet            Run go vet"
	@echo ""
	@echo "Smoke Tests:"
	@echo "  smoke-build    Build smoke-runner binary"
	@echo "  smoke          Run all smoke scenarios (requires SMOKE_* env vars)"
	@echo "  smoke-filter   Run filtered scenarios: make smoke-filter FILTER=gitcode"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps           Download modules"
	@echo "  tidy           Run go mod tidy"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build   Build Docker image (IMAGE=$(IMAGE) VERSION=$(VERSION))"
	@echo "  docker-run     Run CLI inside Docker (pass ARGS='forum posts list')"
