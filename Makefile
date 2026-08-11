.PHONY: test race cover lint build install install-fast uninstall clean tidy check eval

GO      ?= go
PKGS    := ./...
COVER   := coverage.out
MODULE  := github.com/aguinelo/dcode/internal/version

# Onde `make install` põe o binário. Mesmo default do install.sh, e gravável
# pelo usuário — instalar não deve pedir privilégio.
DCODE_INSTALL_DIR ?= $(HOME)/.local/bin

# A versão de um build local diz que é local, no próprio texto. Um binário que
# se apresenta igual a um release publicado é como um relato de bug vira uma
# hora perdida descobrindo que nunca era o código publicado.
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
GIT_DIRTY  := $(shell git diff --quiet HEAD 2>/dev/null || echo .dirty)
GIT_TAG    := $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
BASE_VER   := $(if $(GIT_TAG),$(GIT_TAG),0.0.0)
VERSION    := $(BASE_VER)-dev+$(GIT_COMMIT)$(GIT_DIRTY)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X $(MODULE).Version=$(VERSION) \
           -X $(MODULE).Commit=$(GIT_COMMIT) \
           -X $(MODULE).Date=$(BUILD_DATE) \
           -X $(MODULE).Source=local

test:
	$(GO) test $(PKGS)

race:
	$(GO) test -race $(PKGS)

# -coverpkg counts code exercised from another package's tests. pkg/client is
# driven entirely by the server integration tests, and per-package coverage
# would report it as untested while the assertions that matter run through it.
cover:
	$(GO) test -race -coverprofile=$(COVER) -covermode=atomic -coverpkg=./... $(PKGS)
	@./scripts/coverage.sh $(COVER)

lint:
	$(GO) vet $(PKGS)
	@gofmt -l . | grep -v '^$$' && { echo "gofmt: arquivos não formatados acima"; exit 1; } || true

build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o bin/dcode ./cmd/dcode

# Instala o build local no PATH. Depende de `check` de propósito: instalar algo
# que não passou no gate é como um defeito local vira "o dcode quebrou".
install: check
	@mkdir -p "$(DCODE_INSTALL_DIR)"
	@install -m 0755 bin/dcode "$(DCODE_INSTALL_DIR)/dcode"
	@echo "instalado: $$("$(DCODE_INSTALL_DIR)/dcode" --version | head -1)"
	@echo "em:        $(DCODE_INSTALL_DIR)/dcode"
	@case ":$$PATH:" in \
	  *":$(DCODE_INSTALL_DIR):"*) ;; \
	  *) echo ""; echo "  $(DCODE_INSTALL_DIR) não está no PATH. Adicione:"; \
	     echo "  export PATH=\"$(DCODE_INSTALL_DIR):\$$PATH\"" ;; \
	esac

# Instala sem rodar o gate. Para o laço de edição, onde a suíte já rodou há
# trinta segundos e rodar de novo é só espera.
install-fast: build
	@mkdir -p "$(DCODE_INSTALL_DIR)"
	@install -m 0755 bin/dcode "$(DCODE_INSTALL_DIR)/dcode"
	@echo "instalado (sem gate): $$("$(DCODE_INSTALL_DIR)/dcode" --version | head -1)"

uninstall:
	@rm -f "$(DCODE_INSTALL_DIR)/dcode"
	@echo "removido: $(DCODE_INSTALL_DIR)/dcode"

tidy:
	$(GO) mod tidy

check: lint race cover build

# Contratos comportamentais. Fora do `check` de propósito: cada cenário roda
# DCODE_EVAL_RUNS vezes contra um modelo de verdade, e uma suíte que gasta
# dinheiro a cada commit é uma suíte que alguém desliga.
#
# Sem DCODE_EVAL_ENABLED=true e DCODE_EVAL_MODEL, cada cenário se pula dizendo
# o que falta. -count=1 porque medição em cache não é medição.
eval:
	$(GO) test -tags eval -count=1 -v ./internal/evals/...

# Compila os cenários sem executá-los. É o que impede a suíte de eval apodrecer
# em silêncio enquanto o código que ela mede muda por baixo — o modo de falha
# de todo teste que vive atrás de build tag.
eval-build:
	$(GO) vet -tags eval ./internal/evals/...

clean:
	rm -rf bin $(COVER)
