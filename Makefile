.PHONY: test race cover lint build clean tidy check

GO      ?= go
PKGS    := ./...
COVER   := coverage.out

test:
	$(GO) test $(PKGS)

race:
	$(GO) test -race $(PKGS)

cover:
	$(GO) test -race -coverprofile=$(COVER) -covermode=atomic $(PKGS)
	@./scripts/coverage.sh $(COVER)

lint:
	$(GO) vet $(PKGS)
	@gofmt -l . | grep -v '^$$' && { echo "gofmt: arquivos não formatados acima"; exit 1; } || true

build:
	CGO_ENABLED=0 $(GO) build -trimpath -o bin/dcode ./cmd/dcode

tidy:
	$(GO) mod tidy

check: lint race cover build

clean:
	rm -rf bin $(COVER)
