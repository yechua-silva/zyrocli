VERSION ?= $(shell git describe --tags 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "unknown")

.PHONY: build build-all vet clean sync-mcptools

build:
	go build -ldflags="-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" \
		-o zyrocli ./cmd/zyrocli/

build-all:
	go build ./...

vet:
	go vet ./...

clean:
	rm -f zyrocli

sync-mcptools:
	for f in helix_client.py search_facts.py search_code.py search_skills.py helix_write.py task_context.py runner.py; do \
		cp internal/opencode/mcptools/$$$$f mcp-tools/$$$$f; \
	done
