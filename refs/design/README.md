# Referências de design

Material de design do dcode: maquetes e handoffs. **Nada aqui vai para
produção** — os `.dc.html` são maquetes em HTML do que o terminal deve
renderizar, e `support.js` existe só para que elas abram no navegador.

A fonte canônica do comportamento continua sendo `docs/specs/architecture/`.
**Onde um handoff e a spec divergirem, a spec vence.**

## O que vale

| Arquivo | O que é |
|---|---|
| `HANDOFF.md` | **a referência atual** — v3, v4 e v5 do cliente TUI |
| `DCode TUI v5.dc.html` | a direção a implementar: card de tarefa, coluna lateral, atividade |
| `DCode TUI v4.dc.html` | referência de movimento: barra de atividade com verbo |
| `DCode TUI v3.dc.html` | catálogo de 12 estados (aprovação, erro, fila, <100 col, ASCII) |
| `HANDOFF-v2.md` · `dcode TUI v2.dc.html` | histórico, superado pelo `HANDOFF.md` |
| `support.js` | runtime dos mocks, só para abrir no navegador |

## Conferido contra o repositório — 2026-08-20

O `HANDOFF.md` está **verbatim, como entregue**. As divergências abaixo foram
encontradas conferindo as afirmações dele contra o código, e ficam aqui em vez
de virarem edição no handoff: quem escreveu o design não tinha o repositório à
mão, e reescrever o handoff apagaria a diferença entre o que foi projetado e o
que foi descoberto depois.

**Confere:** Bubble Tea v2, `internal/tui` com `model.go`/`render.go`/`style.go`,
`docs/brand/VOXELS.md`, e as quatro famílias de spec citadas.

**Diverge:**

1. **A colisão de teclas são três, não uma.** Além do `^E` (fim de linha,
   `internal/tui/program.go:494`) que o handoff pegou: **`^N` já é "descer"** em
   `internal/tui/picker.go:173`, junto de `j` e `↓`; e **`^Z` não existe no
   cliente** — o handoff diz "já existe em `tools/undo.go`", mas ali está a
   *ferramenta* que o modelo chama, não a tecla, e `^Z` é controle de job do
   shell, que hoje suspende o processo. `^R`, `^B` e `^F` estão livres.

2. **O handoff afirma "letra nunca é atalho", e `picker.go:173` usa `j`.** Ou o
   picker é exceção declarada na spec, ou a afirmação precisa de correção.

3. **Progresso de ferramenta não existe no protocolo.** Há
   `tool.requested` e `tool.completed`, e nada entre os dois — a coluna "meta
   rodando" do handoff (`184/184 varridos`, `7/12 testes`) não tem de onde vir.
   Isso torna "tarefa como card com progresso" uma mudança de **quatro camadas**
   — evento novo no protocolo, ferramentas emitindo, servidor repassando,
   cliente desenhando —, com changelog em `client-server-protocol` e **MINOR no
   mínimo**, não um turno de TUI.

   A metade "meta pronta", porém, **já existe**: `protocol.ToolCompleted` carrega
   `Lines`, `Files`, `Added`, `Removed`, `ExitCode`, `DurationMS` e `Diff` — e o
   comentário no código já diz a mesma regra que o handoff escreveu, *"a client
   that parses Output to rebuild these numbers breaks silently"*.

4. **A trilha de sessões contradiz a si mesma.** O handoff anota `sessions
   []SessionRow // do protocolo, não do disco`, mas `client.ListSessions`
   devolve **sessões vivas** e o `dcode -r` lê do **disco**
   (`pickSession(ctx, recordDir(…), …)` em `cmd/dcode/tui.go:113`). Ou entra RPC
   novo — outra mudança de protocolo —, ou a nota está errada.

5. **Dois itens já estão prontos e só precisam ser renderizados no lugar novo.**
   `Model.Verification` é exatamente o selo de conclusão da seção 6, e
   `Model.DiffAdded/DiffRemoved/DiffFiles` são a barra inferior, já somados do
   que cada ferramenta reportou em vez de parseados da saída.

Uma nota de forma, não de conteúdo: `Model` não guarda o log — ele reduz para
`Entries []Entry`. "Árvore derivada do log" significa um campo redutor novo
alimentado pelo mesmo fluxo, que é o que o handoff quer; só não é ler um log que
já esteja ali.
