# Checklist de revisão — Go (CLI / harness / daemon)

🇬🇧 [English version](GO-CODE-REVIEW.md)

Convenção deste repositório. Teste: `go test -race ./...`. Build principal: `CGO_ENABLED=0`.
Dirs compartilhados: `internal/**`, `pkg/**`.

Checks reutilizáveis de alta taxa de acerto. Não é exaustivo; aplique o que couber ao diff.

## Concorrência (a maior fonte de bug em Go)

- **Goroutine sem dono:** toda `go func()` precisa de caminho de término claro. Disparada sem `context`, `WaitGroup` ou canal de parada → vazamento. Pergunte: quem cancela isso?
- **`context` como primeiro parâmetro:** função que faz I/O sem receber `ctx` não é cancelável. No caminho de turno do agente isso vira sessão que não morre no Ctrl-C → **blocker**.
- **`ctx` ignorado:** receber `ctx` e nunca checar `ctx.Done()` nem passar adiante é pior que não receber — dá falsa garantia.
- **Captura de variável de loop:** o clássico `for _, v := range` + `go func(){ use(v) }` em Go < 1.22. Confirme a versão em `go.mod` antes de apontar.
- **Canal sem buffer em caminho de escrita:** produtor bloqueia se o consumidor sumiu. Em fluxo de eventos, sempre `select` com `ctx.Done()` no envio, nunca `ch <- x` cru.
- **`sync.Mutex` copiado:** struct com mutex passada por valor. `go vet` pega — confirme que roda na CI.
- **Data race:** mudança que introduz estado compartilhado sem lock exige `go test -race` na CI. Se o PR mexe em concorrência e a CI não roda `-race`, é achado de processo.

## Erros

- **`err` engolido:** `_ = f()` ou `if err != nil { return nil }` sem contexto. Em Go o erro é o contrato — descartar é perda de informação.
- **Wrapping:** `fmt.Errorf("...: %w", err)` para preservar a cadeia. Sem `%w`, `errors.Is`/`errors.As` param de funcionar acima.
- **Sentinela vs string:** comparar erro por `strings.Contains(err.Error(), ...)` → **sempre** achado. Deve ser `errors.Is` com sentinela exportada.
- **`panic` em biblioteca:** aceitável só em erro de programação irrecuperável na inicialização. Em caminho de request ou de turno → blocker.
- **`recover` sem re-log:** engolir panic silenciosamente esconde bug estrutural.

## Recursos e I/O

- **`defer` em laço:** `defer f.Close()` dentro de `for` só executa no fim da função — acumula descritores. Extrair para função ou fechar explicitamente.
- **Body de HTTP não fechado:** `resp.Body.Close()` obrigatório mesmo em erro, e **ler até EOF** antes de fechar para reusar a conexão.
- **Timeout ausente:** `http.Client{}` sem `Timeout` nunca desiste. Toda chamada externa (provider, MCP) precisa de timeout **e** de `ctx`.
- **`io.ReadAll` em corpo não confiável:** resposta de provider ou saída de ferramenta sem limite → OOM. Use `io.LimitReader`.

## Específico deste produto

- **Mutação de prefixo de contexto:** código que edita, reordena ou remove mensagem já enviada viola o append-only e invalida cache KV → **blocker**.
- **Timestamp ou contador no prompt do sistema:** mesma consequência — invalida cache a cada turno.
- **Schema de ferramenta montado tarde:** definição de tool que só existe após conectar MCP em runtime invalida o prefixo. Deve vir de cache de startup.
- **Montagem de contexto deve ser função pura:** `(estado da sessão) → []Message` sem I/O e sem relógio dentro. É o que permite golden test exato — efeito colateral aí é achado de arquitetura, não de estilo.
- **Alocação no caminho quente:** alocar por token ou por evento vira pressão de GC sob swarm. Se o PR mexe no loop de turno, pergunte: isso aloca por delta? Buscar reuso de buffer.
- **`exec.Command` sem política:** execução fora da fronteira definida na spec de permissão → **blocker**. Deve passar pelo executor com política, nunca `exec` direto.
- **cgo no núcleo:** `import "C"` fora do pacote isolado por build tag quebra binário estático e compilação cruzada. A CI valida `CGO_ENABLED=0` no build principal.

## SOLID / DRY / estrutura

- **Interface definida no produtor:** em Go a interface pertence ao **consumidor**. Pacote que exporta interface + única implementação junto → geralmente abstração prematura.
- **Interface larga:** mais de 3–4 métodos costuma indicar responsabilidade misturada. Prefira interfaces pequenas e composição.
- **`internal/` vs `pkg/`:** o que não é contrato público **deve** estar em `internal/`. Tipo exportado em `pkg/` vira compromisso de compatibilidade — confirme se foi intencional e se o `.p.spec.md` declara o nível de estabilidade.
- **`any` em API pública:** perde a tipagem que é a razão de usar Go. Genéricos ou tipo concreto quase sempre cabem.
- **Dependência global:** `var db *sql.DB` de pacote, singleton implícito, `init()` com efeito colateral → dificulta teste e esconde ordem de inicialização.

## Testes

- **Table-driven:** é o idioma da linguagem. Sequência de `t.Run` copiada com corpo idêntico → refatorar para tabela.
- **Golden files:** mudança em saída serializada (evento, contexto montado, render de TUI) sem `testdata/` atualizado → cobertura ausente.
- **`t.Parallel()` com estado compartilhado:** teste paralelo que escreve em variável de pacote ou no mesmo diretório temporário → flake.
- **Teste que depende de rede ou de modelo real:** atrás de build tag ou `testing.Short()`. Comportamento mediado por modelo é medido por limiar na seção de contratos comportamentais do `.p.spec.md`, **não** no `go test` determinístico.
