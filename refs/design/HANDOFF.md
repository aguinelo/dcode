# Handoff: DCode — cliente TUI (v3 · v4 · v5)

## Overview

Interface de terminal do **DCode** (binário, comandos e caminhos continuam
`dcode`, minúsculo — só o **nome do produto** é DCode). Três documentos de
design, do mais conservador ao mais novo:

| Arquivo | O que é | Estado |
|---|---|---|
| `DCode TUI v3.dc.html` | catálogo estático de **12 estados** (vazio, laço do turno, delegação, aprovação, erro, fila, <100 col, cópia, `-r`, full-access, ASCII) | referência de comportamento |
| `DCode TUI v4.dc.html` | uma tela **viva**: barra de atividade permanente com verbo + fato, transições suaves | referência de movimento |
| `DCode TUI v5.dc.html` | **direção proposta**: tarefa como card vivo, coluna lateral com árvore de arquivos + trilha de sessões | implementar esta |

Alvo: **Go + Bubble Tea v2 / Lip Gloss v2 / Bubbles v2**, no pacote
`internal/tui` que já existe (reducer puro em `model.go`, render puro em
`render.go`, papéis de cor em `style.go`).

## About the Design Files

Os `.dc.html` são **referências de design feitas em HTML** — maquetes do que o
terminal deve renderizar, **não** código para copiar. A tarefa é reproduzir os
estados em Go. Nada do HTML vai para produção; `support.js` está no pacote só
para os arquivos abrirem no navegador.

Cada bloco escuro é um terminal inteiro. **Larguras em px são ilustrativas**: a
unidade real é a célula de exibição, e toda medida no código usa
`lipgloss.Width` / `go-runewidth`, nunca `len()`. O v4 e o v5 usam sans no
"chrome" só para leitura do mock — no terminal tudo é a fonte do usuário.

Fonte canônica do comportamento: `docs/specs/architecture/client-tui/`,
`agent-loop/`, `delegated-writing/`, `task-ledger/` e `docs/brand/`. **Onde este
README e a spec divergirem, a spec vence.**

## Fidelity

**Alta fidelidade** em cores, glifos, ordem de campos, textos literais e
durações de animação. **Ilustrativo**: proporções em px, conteúdo de exemplo
(nomes de arquivo, saídas), fontes e o raio de borda dos cards (no terminal, um
card é uma borda de runas ou apenas espaçamento + regra fina).

---

## Layout do v5

```
┌──────────────┬──────────────────────────────────────────┬──────────────┐
│ ARQUIVOS ^E  │ status: ⏺ DCode  MiniMax-M3  ws-write    │ PLANO        │
│ ▾ dcode/     │──────────────────────────────────────────│ ✓ 1 …        │
│   ▾ internal/│ ┌ ⏺ grep  \.Save\(      184 varridos ─┐  │ ▸ 2 …        │
│     ◦ alpha/…│ └──────────────────── barra de progresso┘ │              │
│ SESSÕES ^R   │ ┌ ⏺ explore  4 filhos                  ┐  │ TURNO        │
│ ● catalogar… │ │  ✓ alpha  owns …  ▇▇▇▇  read 9·wrote1│  │ iteração 2/100│
│   extrair…   │ │  ⊘ bravo  owns …  ▇▇    não respondeu│  │ em vôo 2·teto4│
│──────────────│ └──────────────────────────────────────┘  │              │
│ feat/catalogo│ atividade: ⏺ Repartindo o trabalho  fato… │ ^Z desfaz    │
└──────────────┴──────────────────────────────────────────┴──────────────┘
  barra inferior: [feat/catalogo] +156 −0 · 2 arquivos · ⏵ docker … adotado
```

Composição: `JoinHorizontal(rail, JoinVertical(status, fluxo, atividade), painel)`
com a barra inferior por baixo de tudo. Regiões, alturas e ordem de queda:

| Região | Tamanho | Notas |
|---|---|---|
| Status | 1 linha | sessão: modelo, sandbox, política, medidor de contexto |
| Coluna lateral | `clamp(20, w/5, 30)` col. | duas seções dobráveis; **some abaixo de 100 col.** |
| Fluxo | resto | cards + linhas de texto, ancorado no fim (`follow`) |
| Atividade | 1 linha | **só enquanto o turno roda**: verbo + fato + tempo + tokens |
| Painel de plano | `clamp(16, w/4, 34)` col. | plano + fatos do turno |
| Barra inferior | 1 linha, **nunca duas** | worktree · diff · segundo plano · pendência |

---

## Screens / Views

### 1. Coluna lateral — ARQUIVOS (`^E`)

Árvore do workspace, **com pasta de filho único compactada** (`alpha/ARCH.md`
numa linha só). Cabeçalho: chevron `▾` (gira para `▸` quando dobrado, **não é
clicável** — o atalho é `^E`), rótulo `ARQUIVOS`, a tecla, e à direita
`n tocados`. Dobrado, o cabeçalho ainda responde: `11 arquivos · 3 tocados`.

Estado por linha, derivado do log de eventos (nunca de polling do disco):

| Estado | Glifo | Cor | Meta à direita | Animação |
|---|---|---|---|---|
| pasta | `▾` | `#a8b0b6` | `2 ativos` se algum filho ativo | herda o âmbar |
| lendo | `◦` | `#eef1f3` | `lendo` | pulso âmbar 1.7s |
| escrevendo | `◦` | `#eef1f3` | `escrevendo` | pulso âmbar 1.7s |
| varrendo (glob/grep) | `▾` | `#eef1f3` | `varrendo 184` | pulso na pasta |
| falhou (filho caiu) | `⊘` | `#d9787c` | `sem resposta` | barra esquerda vermelha |
| concluído | `✓` | `#74b98a` | `+38` | **para de animar** |
| inexistente | — | — | — | **a linha não é desenhada** |

Invariantes novas que isso exige:
- **Arquivo que o turno ainda não criou não aparece** — nem antes de escrever,
  nem entre a queda de um filho e a reescrita pelo pai.
- **A árvore é derivada do log**, como todo o resto do cliente: mesma sessão
  reaberta reproduz a mesma árvore.
- Movimento só onde há mudança. Concluído não pulsa.

### 2. Coluna lateral — SESSÕES (`^R`)

É o `dcode -r` promovido a coluna permanente. Uma linha por conversa **deste
workspace**: título derivado da primeira pergunta, idade, turnos, e a marca da
seleção. Três modos (tweak `railMode` no mock):

- **passiva** — a sessão aberta leva `●` âmbar e a linha viva do verbo.
- **navegando** (`^R`) — a coluna **toma o teclado**, como o modo de cópia.
  Cursor é `▸`, um **caractere, nunca só cor**. `↑↓` **não dá a volta**,
  `enter` continua, `esc` **começa do zero**. Filtro digitado acima da lista.
- **nomeando** (`^N`) — *acréscimo desta proposta*: nome dado pela pessoa vence
  o título derivado, ganha `· nomeada`, `esc` mantém o derivado.

Regras que vêm da spec e não mudam: sessão em que **nada foi perguntado não
entra** na lista; não há coluna de id; **um workspace só** — outro projeto é
outro DCode aberto naquele diretório (sandbox, política e cadeia de instruções
são ancorados num workspace).

`^B` dobra a coluna inteira. Abaixo de 100 colunas ela desaparece e a
**preferência explícita vence nos dois sentidos**.

### 3. Fluxo — a tarefa é um card

Card = borda 1px `#1e2126`, fundo `#15181c` (rodando) / `#121417` (terminado),
raio 8px, cabeçalho `⏺ <ferramenta> <alvo> … <meta>` e uma barra de progresso de
2px no rodapé que **desaparece ao terminar** (`opacity` 0, 0.8s).

Progresso e meta vêm **do metadado da ferramenta**, nunca de parsear a saída
livre:

| Ferramenta | Meta rodando | Meta pronta |
|---|---|---|
| `grep` / `glob` | `184/184 arquivos varridos` | `11 matches · 4 arquivos` |
| `read` | `n de 240 lines` | `240 lines` |
| `write` | `n de 38 lines` | `created, 38 lines` |
| `edit` | `…` | `+24 −2` |
| `bash` (testes) | `7/12 testes` | `exit 0 · 12 passam · 31.2s` |
| `bash` adotado | `adotado em segundo plano · ~40s` | continua vivo fora do turno |

**Duração só a partir de 500 ms.** Enquanto roda, `…` quando não há contagem.

### 4. Delegação — um card com os filhos dentro

Cabeçalho `⏺ explore · 4 filhos · propriedade disjunta`. Uma sub-linha por
filho: glifo (`⏵` rodando, `✓` pronto, `⊘` sem resposta), nome, `owns <caminho>`,
barra própria, resultado (`read 9 · wrote 1`). Regras da
`delegated-writing.r/.p`:

- **Filho nunca pergunta**: escrita fora do `owns` é **negada e reportada**.
- **Quem não respondeu é nomeado**, com o motivo, na própria linha — nunca
  resumido junto dos que responderam.
- Tokens do filho **debitados do pai**; teto de concorrência é da sessão.
- `^Z` (undo do turno) alcança **a delegação inteira** como unidade.
- A definição de pronto roda **uma vez, na árvore inteira, no turno do pai**.

### 5. Barra de atividade (v4 e v5)

`⏺ <verbo> <ferramenta> <alvo> … <tempo> <tokens> ^C interrompe`.

O **verbo** é um gerúndio curto que troca a cada 2.4s com crossfade de 420ms,
sorteado do conjunto da fase (`pensando`, `lendo`, `delegando`, `escrevendo`,
`conferindo`). Regra: **o verbo nunca aparece sozinho** — a ferramenta e o alvo
reais ficam do lado dele. Ocioso, o verbo dá lugar ao resumo do turno e o
spinner vira `✓`.

Nunca entra no histórico enviado ao modelo. Desligável junto de
`behavior.show_reasoning`? **Não** — é do cliente; sugerido
`tui.activity_verbs = true|false`.

### 6. Selo de conclusão

`✓ verified · 2 checks` (verde) · `⚠ unverified` (amarelo) ·
`!! NOT verified !!` (branco sobre vermelho). É o selo do `done.toml` depois da
última edição — **nunca uma frase do modelo**.

---

## Interactions & Behavior — teclado (superfície `stable`)

Tudo do `client-tui.p` continua valendo (`Enter`, `↑↓`, `PgUp/PgDn`,
`Home/End`, `Tab`, `Esc`, `^P`, `^C`, `^D`, `^A/^E`, `^W/^U/^K`, `?` só em linha
vazia). **Letra nunca é atalho.** Acrescenta-se:

| Tecla | Ação | Nota |
|---|---|---|
| `^E` | dobra/expande ARQUIVOS | preferência explícita persiste na sessão |
| `^R` | foca a trilha de sessões | dona do teclado enquanto aberta |
| `^N` | nomeia a sessão sob o cursor | só com a trilha focada |
| `^B` | dobra a coluna inteira | vence a regra de largura |
| `^Z` | desfaz o turno (inclui filhos) | já existe em `tools/undo.go` |

Colisão a checar na implementação: `^E` hoje é "fim da linha" (`^A`/`^E`). **Não
reaproveitar `^E`** — usar `^F` (files) para a árvore, ou `^E` só quando a linha
de entrada estiver vazia. Decidir na spec antes de codar; o mock usa `^E` como
rótulo ilustrativo.

## Animações e transições

| O que | Duração / easing |
|---|---|
| Card / linha entrando | 520ms `cubic-bezier(.16,.84,.28,1)`, `translateY(6px)` → 0, opacidade 0 → 1 |
| Barra de progresso | 900ms `cubic-bezier(.3,.8,.3,1)` na largura |
| Barra some ao concluir | 800ms opacidade |
| Cor mudando de estado | 500–700ms ease |
| Crossfade do verbo | 420ms; troca a cada 2.4s |
| Pulso do arquivo tocado | 1.7s ease-in-out infinito, fundo `rgba(215,175,95,.05→.13)` |
| Respiro do spinner | 2.4s ease-in-out infinito |
| Caret | 1.1s step-end |

**No terminal isso não é CSS.** É `tea.Tick` de ~120 ms **só enquanto o turno
roda** (ocioso, nenhum tick e nenhum quadro), com o quadro sendo função pura do
contador. Onde o mock usa opacidade, o terminal usa **degrau de cor** (âmbar →
âmbar escuro → dim) e onde usa `translateY`, o terminal simplesmente **imprime a
linha** — a entrada suave não existe em célula de texto.

## State Management

O cliente **não guarda estado de sessão**. Além do que já existe (`scroll`,
`panelVisible`, `queue`, `follow`), a proposta acrescenta:

```go
type Model struct {
    // … campos atuais
    tree      FileTree   // derivada do log: caminho → último evento que o tocou
    rail      RailState  // collapsed(files,sessions,column) + focus + cursor + filter
    sessions  []SessionRow // do protocolo, não do disco
    activity  Activity   // verbo atual, quando trocou, fase
    cards     []Card     // progresso por chamada, ancorado no índice de emissão
}
```

- **Modelo é redutor puro sobre o log; View é pura sobre modelo + geometria.**
  Nenhum teste precisa de terminal.
- `tea.WindowSizeMsg` só recalcula geometria (largura da coluna e do painel).
- Progresso de card vem de evento de ferramenta, **nunca** de parsear stdout.

## Design Tokens

### Marca (imutável — `docs/brand/`)
| Token | Hex | Papel |
|---|---|---|
| highlight | `#EFC066` | face de cima |
| body | `#E0A030` | face frontal — **primária**, o `⏺` |
| shadow | `#B87D1E` | base e aresta |
| eye | `#A8452A` | o olho, e nada mais |

### Papéis na TUI (ANSI é o que vale)
`accent 38;5;179` · `added/ok 32` · `removed/error 31` · `warn 33` ·
`danger 1;97;41` (**só** full-access) · `dim 2` · `bold 1` · `cursor 7` ·
`onAccent 48;5;179;38;5;234`.

Medidor de contexto: `dim` até 74%, `warn` 75–89%, `error` ≥ 90%; abaixo de 1%
escreve `ctx <1%`.

### Equivalentes do mock (só para leitura)
fundo `#101215` · superfície `#0d0f11` · card `#15181c` / `#121417` ·
borda `#1e2126` · divisor `#1a1d21` · texto `#cdd2d6` · ênfase `#eef1f3` ·
dim `#8b9298` · faint `#5b6167` · ghost `#3b4045` · accent `#d7af5f` ·
âmbar `#e0a030` · ok `#74b98a` · erro `#d9787c` · warn `#d9c07a`.

Tipografia do mock: `JetBrains Mono` (conteúdo) e `IBM Plex Sans` (chrome), 11–13px,
line-height 1.65. No terminal: a fonte do usuário, sempre.

### Escala tipográfica — todos os tamanhos, por elemento

O terminal tem **um** tamanho de fonte (o do usuário), então a hierarquia do
mock precisa ser traduzida em **peso, cor e caixa** — não em px. A coluna
"terminal" diz o que fazer com cada nível. Ajuste os px no mock e o mapeamento
aqui junto.

| px | Peso | Onde aparece (v5) | Terminal |
|---|---|---|---|
| **13** | 700 | nome do produto `DCode` (topo do mock e do status) | `bold` |
| **12.5 → 10** | 400 | corpo do fluxo: alvo e ferramenta do card, título de sessão, item de plano, verbo de atividade | tamanho base, cor `text` |
| **12 → 10** | 400 | nome de arquivo na árvore, modelo (`MiniMax-M3`), linhas de texto do fluxo (prosa, `⎯ iteração`), fatos do painel | base em `dim` |
| **11.5** | 400/500 | meta do card (`7/12 testes`, `+38`), nome do filho, pílulas `workspace-write` / `on-request`, barra inferior, filtro da trilha | base em `dim`/`faint` |
| **11** | 400 | idade e turnos da sessão, `owns …`, contadores do painel, caminho `~/work/dcode`, glifo do chevron | base em `faint` |
| **10.5** | 600 | rótulos de seção (`ARQUIVOS`, `SESSÕES`, `PLANO`, `TURNO`), teclas (`^E`, `^R`), notas de rodapé, `n tocados` | `dim` + **caixa alta** + `letter-spacing .14em` (no terminal: só caixa alta) |

v4, para referência: **13.5** (corpo do fluxo), 13 (nome do produto), 12.5
(verbo e prosa), 12 (status e painel), 11.5 (metadados e barra inferior), 11
(rótulos).

**Valores fechados no v5 (agosto/2026):** corpo e base em **10px**, o resto na
escala original — `fsProduto 13 · fsCorpo 10 · fsBase 10 · fsMeta 11,5 · fsSec 11 ·
fsRotulo 10,5`. Conteúdo menor que metadado é intenção: densidade de terminal
real, onde o peso e a cor — não o tamanho — fazem a hierarquia. Em Go isso
significa que **conteúdo e metadado têm o mesmo corpo**, e a diferença fica
inteira em `dim` vs `text`.

**Alturas de linha e ritmo:** 1.65–1.7 no fluxo, 1.35–1.4 em título de sessão e
item de plano (texto que quebra em duas linhas), 1.55–1.6 nas notas de rodapé.
No terminal, uma linha é uma linha — o que sobra do ritmo é a **linha vazia**
entre blocos, e o mock usa exatamente uma.

**Espaçamentos verticais do mock, para converter em linhas:** card = 9px de
padding no cabeçalho + 5–6px por sub-linha; gap de 8px entre blocos do fluxo
(≈ 1 linha em branco); linha da árvore = 2px de padding (linhas adjacentes, sem
respiro); seção da coluna lateral = 8–12px de padding (≈ 1 linha).

**Larguras fixas em `ch`** (essas sim vão para o Go, medidas em células):
ferramenta 6.5ch · glifo 1.2–2.4ch · nome do filho 14ch · `owns` 22ch · meta do
filho 15ch · indentação da árvore 11px por nível ≈ **2 células por nível**.

| Papel | Peso | Cor |
|---|---|---|
| produto | bold | `emph #eef1f3` |
| conteúdo | normal | `text #cdd2d6` |
| metadado | normal | `dim #8b9298` |
| secundário | normal | `faint #5b6167` |
| desligado | normal | `ghost #3b4045` |

### Duas invariantes que tornam a cor removível
1. **Estilo nunca altera largura medida.**
2. **Monocromático não emite escape algum**, nem de reset. `NO_COLOR`,
   `TERM=dumb`, `DCODE_ASCII=1`: glifos viram ASCII mantendo
   bloqueado (`!`) ≠ concluído (`x`), e `full-access` **muda o texto**.

No modo ASCII a árvore usa `+-` / `|` e o pulso vira o sufixo `*` na linha
tocada — movimento não pode ser a única pista.

## Assets

Nenhum binário. Mascote e logomarca são arte em meio-bloco desenhada em runas
(`█ ▀ ▄`); mapa canônico em `docs/brand/VOXELS.md`.

## Files

- `DCode TUI v5.dc.html` — a direção a implementar (cards, coluna lateral, atividade)
- `DCode TUI v4.dc.html` — referência de movimento (verbo + transições)
- `DCode TUI v3.dc.html` — os 12 estados, incluindo aprovação, erro, fila, <100 col, ASCII
- `support.js` — runtime só para abrir os mocks no navegador; **não** vai para produção

## O que precisa de spec antes de codar

Estes quatro itens **não existem** na spec hoje e são acréscimos desta proposta.
Cada um pede changelog em `docs/specs/architecture/client-tui/`:

1. **Árvore de arquivos na coluna lateral** — derivada do log, com as invariantes
   "arquivo inexistente não é desenhado" e "concluído não anima".
2. **Trilha de sessões permanente** (o `-r` como coluna) e **`^N` nomear**.
3. **Tarefa como card com progresso** — e a regra de que progresso vem do
   metadado da ferramenta.
4. **Verbo de atividade** — e a regra "nunca sozinho: sempre com ferramenta e
   alvo ao lado".

Colisão de tecla (`^E`) resolvida na spec, não no código.
