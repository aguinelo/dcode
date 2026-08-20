# Handoff: dcode — cliente TUI

## Overview
Interface de terminal do **dcode**, um agente de codificação que planeja antes de escrever, executa por lista de tarefas e não entrega código sem teste e diff revisado. O alvo de implementação é **Go + Bubble Tea / Lip Gloss / Bubbles**.

O documento de design cobre nove estados: estado vazio, turno rodando, modal de aprovação, erro com item bloqueado, fila de entrada + rolagem pausada, layout abaixo de 100 colunas, autocomplete de comandos, `full-access`, e degradação `NO_COLOR` + `DCODE_ASCII=1`.

## About the Design Files
`dcode TUI v2.dc.html` é uma **referência de design feita em HTML** — uma maquete do que o terminal deve renderizar, não código para copiar. A tarefa é **reproduzir esses estados em Go**, com Bubble Tea para o loop de eventos, Lip Gloss para composição e estilo, e os componentes de `bubbles`. Nada do HTML vai para a produção.

Cada bloco escuro no HTML representa um terminal inteiro. Larguras em px são ilustrativas: a unidade real é a **célula de exibição**, e todo cálculo de largura no código deve usar `lipgloss.Width` / `go-runewidth`, nunca `len()`.

Fonte canônica do comportamento: `docs/specs/architecture/client-tui/` e `docs/brand/`. Onde este README e a spec divergirem, a spec vence.

## Fidelity
**Alta fidelidade.** Cores, glifos, ordem de campos e textos literais são finais. O que é ilustrativo: proporções em pixels, o conteúdo de exemplo (nomes de arquivo, saídas de teste) e a fonte (JetBrains Mono no HTML; no terminal é a fonte do usuário).

---

## Layout

```
┌──────────────────────────────────────────┬────────────────────┐
│ ✓ dcode  MiniMax-M3  workspace-write  ctx 34%│ PLAN            │  status
│                                          │                    │
│ > adicionar validação de CPF             │ ✓ 1 Mapear o fluxo │  fluxo
│   ⏺ read  handler.go        240 lines    │ ▸ 2 Implementar    │
│   ⏺ edit  validate.go       +24 −2  1.2s │   3 Cobrir teste   │
│                                          │ ⊘ 4 Rodar a suíte  │
│                                          │     └ falta dep    │
│                                          │ 2 of 4 · 1 blocked │
│                                          │ [^p] hide panel    │
│ ⠹ bash go test ./...  4.2s  1.2k tok  ^C │                    │  trabalho
│ > _                    ↓ 12 lines below  │                    │  entrada
└──────────────────────────────────────────┴────────────────────┘
```

| Região | Altura | Notas |
|---|---|---|
| Status | 1 linha | sempre visível |
| Fluxo | resto | `bubbles/viewport` |
| Linha de trabalho | 1 linha | **só enquanto um turno roda** |
| Entrada | 1 linha | `bubbles/textinput` |
| Painel de plano | altura do fluxo | à direita, `clamp(16, w/4, 34)` colunas |

Composição: `JoinVertical(status, JoinHorizontal(fluxo, painel), trabalho, entrada)`.

---

## Screens / Views

### 1. Estado vazio (`stateEmpty`)
Sessão nova sem nenhum turno. Mascote em meio-bloco à esquerda; à direita, em coluna: nome `dcode` em bold, `MiniMax-M3`, o modo de sandbox, e a linha de atalhos `? help    ^C interrupt`.

```
    ▄▄▄▄
    █▀▀█        dcode
    ████
  ▄▄▄▄▄▄▄▄      MiniMax-M3
  ████████      workspace-write
▄▄▄▄▄▄▄▄▄▄▄▄
████████████    ? help    ^C interrupt
 ▀▀      ▀▀
```

Cores por linha: linhas `▄` = highlight `#EFC066`; linhas `█` = body `#E0A030`; a última (`▀▀      ▀▀`, os pés) = shadow `#B87D1E`; o `▀▀` dentro de `█▀▀█` = eye `#A8452A`.

- Desaparece no primeiro turno e **não volta** enquanto a sessão viver.
- **Nunca aparece em sessão retomada.**

### 2. Turno rodando (`stateRunning`)
Status com spinner. Fluxo com: linha do usuário (`>` âmbar + texto em bold), prosa do agente indentada 2 colunas, chamadas de ferramenta, diff.

**Chamada de ferramenta, uma linha:** `⏺ <ferramenta>  <alvo>  <resumo>  <duração>`

| Ferramenta | Alvo | Resumo |
|---|---|---|
| `read` | caminho | `240 lines` / `240 lines (truncated)` |
| `edit` | caminho | `+24 −2` |
| `write` | caminho | `created, 120 lines` / `+120 −118` |
| `glob` | padrão | `18 files` |
| `grep` | padrão | `18 matches in 4 files` |
| `bash` | comando | `exit 0` / `exit 1` |
| `plan` | — | `5 items` |

- O resumo vem **do metadado da ferramenta**, nunca de parsear o texto de saída.
- **Duração só a partir de 500 ms.**
- Enquanto roda, o resumo é `…`.
- Colunas alinhadas: ferramenta 6 células, alvo 26 células, depois resumo e duração.

**Diff:** `+` verde, `−` vermelho, cabeçalho e `@@` em dim. Corpo indentado 4 colunas com uma barra de continuação. Truncamento com marca explícita: `⋯ 19 lines · Tab expande`.

**Linha de trabalho:** `⠹ bash go test ./internal/domain  4.2s  1.2k tok  ^C interrompe` — spinner animado, o que roda, tempo medido **pelo cliente**, tokens acumulados, como parar.

### 3. Modal de aprovação (`stateApproval`)
Sobreposto e centralizado (largura ~560px no mock, ~64 colunas), borda âmbar, fluxo atrás esmaecido, **entrada bloqueada**.

```
┌─ Approval needed ──────────────────────────┐
│   bash crosses: network                    │
│     curl -X POST https://…                 │
│   [d] deny   [a] allow   [A] whole session │
│   Enter denies.                            │
└────────────────────────────────────────────┘
```

- Toma a tela; bloqueia toda outra tecla.
- **Enter nega**; deny listado primeiro.
- Mostra o **comando renderizado**, não uma descrição.
- `A` maiúsculo para a sessão inteira.
- **Não mostra o plano.**

### 4. Erro e item bloqueado (`stateError`)
Falha vem **expandida**; sucesso vem recolhido. Saída do erro com barra vermelha à esquerda, truncada com `⋯ 34 lines truncated · Tab expande`. Abaixo, a explicação do agente e o motivo do bloqueio.

No painel: `⊘ 5 Rodar a suíte de integração` com `└ falta testcontainers` na linha seguinte; rodapé `3 of 5 · 1 blocked`.

### 5. Fila de entrada e rolagem pausada (`stateQueued`)
- Aviso de rolagem parada, alinhado à direita, no fim do fluxo: `↓ 42 lines below · End to follow` (fundo `#303030`).
- Itens enfileirados acima da entrada: `⇥ 1 <texto>` + `^X remove`; contador `2 na fila` à direita da entrada.
- Comportamento: acompanha a saída nova por padrão; rolar para cima para de acompanhar e **saída nova não move a tela**; voltar ao fim religa. A fila drena como **um único turno**, na ordem digitada. Fila cheia **recusa em voz alta**.

### 6. Abaixo de 100 colunas
Painel escondido; o contador **e a tecla** migram para o status: `3 of 5 · 1 blocked · ^p`. O painel encolhe até o piso de 16 colunas antes de sumir. **Preferência explícita (`^P`) vence nos dois sentidos.** Sem plano, sem painel.

### 7. Autocomplete de comandos
Lista acima da entrada, item selecionado com fundo `#303030`, nome do comando em âmbar, descrição em dim, rodapé `5 de 7 · ↑↓ navegar · ⇥ completar · esc fechar`.

### 8. `full-access`
`!! FULL-ACCESS !!` em branco bold sobre vermelho (`1;97;41`), sempre em destaque — **muda o texto**, não só a cor, para sobreviver ao modo monocromático.

### 9. `NO_COLOR` / `DCODE_ASCII=1`
Nenhum escape emitido, **nem o de reset**. Glifos viram ASCII mantendo bloqueado ≠ concluído:

| Unicode | ASCII | Estado |
|---|---|---|
| (espaço) | (espaço) | pendente |
| `▸` | `>` | ativo |
| `✓` | `x` | concluído |
| `⊘` | `!` | bloqueado |

---

## Interactions & Behavior — teclado (superfície `stable`)

| Tecla | Ação |
|---|---|
| `Enter` | enviar (enfileira se um turno roda) |
| `↑` `↓` | linha vazia: histórico · linha com texto: cursor no fluxo |
| `PgUp` `PgDn` | rolar uma tela, com 2 linhas de sobreposição |
| `Home` `End` | início da sessão · fim, religando o acompanhamento |
| `Tab` | expandir/recolher o item sob o cursor |
| `Esc` | fecha a expansão, depois a seleção, depois o modal |
| `^P` | mostrar/esconder o painel |
| `^C` | interrompe o turno; ocioso, sai |
| `^D` | sair |
| `^A` `^E` | início e fim da linha |
| `^W` `^U` `^K` | apagar palavra · limpar linha · cortar até o fim |
| `←` `→` `Del` | mover o caret · apagar sob ele |
| `?` | ajuda, **só em linha vazia** |

- **Letra nunca é atalho** — atalho é tecla de controle ou pontuação.
- Teclas de rolagem nunca dependem do que está digitado.
- `^C` interrompe antes de sair.
- O cursor puxa a janela para si.

## Comandos embutidos

| Comando | Ação |
|---|---|
| `/help` | atalhos e comandos |
| `/init` | escreve `DCODE.md` a partir do repositório |
| `/clear` | encerra a sessão e abre outra |
| `/plan` | mostra o plano completo; com argumento, força replanejamento |
| `/config <chave>` | valor efetivo **e** procedência |
| `/model <nome>` | troca de modelo — abre sessão nova |
| `/resume` | lista sessões e reconecta |

`/clear` e `/model` **abrem sessão nova, não limpam a tela**. Comando embutido vence comando do usuário de mesmo nome, e a colisão é reportada.

## State Management

O cliente **não guarda estado de sessão**. Só: posição de rolagem, visibilidade do painel e fila de entrada. Fechar e reabrir replica a sessão a partir do log e cai na mesma tela.

```go
type Model struct {
    state    State          // empty | running | approval | idle
    log      []Event        // append-only, vindo do protocolo
    plan     Plan
    queue    []string
    geom     Geometry       // width, height, panelVisible, panelPref
    follow   bool           // viewport acompanha o fim
    spinner  spinner.Model
    vp       viewport.Model
    input    textinput.Model
    theme    Theme
}
```

- **Modelo é redutor puro sobre o log; View é pura sobre modelo + geometria.** Nenhum teste precisa de terminal.
- `tea.WindowSizeMsg` atualiza só a geometria e recalcula a largura do painel.
- `tea.Tick` de ~120 ms **só enquanto há turno rodando** — move spinner e tempo decorrido. Ocioso, nenhum tick.
- Tokens/eventos chegam como `tea.Msg` por canal; `Update` anexa ao log e chama `viewport.GotoBottom()` **apenas se `follow == true`**.

## Design Tokens

### Marca
| Token | Hex | Papel |
|---|---|---|
| highlight | `#EFC066` | face de cima de cada caixa |
| **body** | **`#E0A030`** | face frontal — **primária** |
| shadow | `#B87D1E` | base e aresta direita |
| **eye** | **`#A8452A`** | o olho, e nada mais |

### Papéis na TUI (não cores)
| Papel | ANSI | Uso |
|---|---|---|
| `accent` | `38;5;179` | âmbar da marca — spinner, bullet, cursor de seleção, `>` do prompt |
| `added` / `ok` | `32` | `+` no diff, `exit 0` |
| `removed` / `error` | `31` | `−` no diff, falha |
| `warn` | `33` | contexto ≥ 75% |
| `danger` | `1;97;41` | **só** `full-access` |
| `dim` | `2` | metadados, durações, dicas |
| `bold` | `1` | nome do produto, linha do usuário |
| `cursor` | `7` | o caret |

Medidor de contexto: `dim` até 74%, `warn` de 75 a 89%, `error` a partir de 90%. Abaixo de 1% escreve `ctx <1%`, nunca arredonda para zero.

Equivalentes usados no HTML (só para leitura do mock): accent `#d7af5f`, ok `#5faf5f`, error `#d75f5f`, warn `#d7d75f`, danger fundo `#af0000`, dim `#767676`, texto `#d0d0d0`, ênfase `#eeeeee`, fundo `#1c1c1c`, borda `#303030`, separador `#262626`.

### Duas invariantes que tornam a cor removível
1. **Estilo nunca altera largura medida** — toda medida em células, com escapes descontados.
2. **Monocromático não emite escape algum**, nem de reset.

Nada de conteúdo se perde com a cor desligada: `full-access` muda o texto, erro tem `!`, diff tem `+`/`−`.

### Degradação
| Situação | O que acontece |
|---|---|
| `NO_COLOR` ou `TERM=dumb` | desliga a cor |
| `DCODE_COLOR=always/never` | decide sobre os dois |
| Sem Unicode | glifos viram ASCII, mantendo bloqueado ≠ concluído |
| `DCODE_ASCII=1` | força ASCII |

## Bibliotecas sugeridas
`bubbles/viewport` (fluxo e diff), `bubbles/textinput` (entrada), `bubbles/spinner` (Dot — status e linha de trabalho), `bubbles/help` (`?`), `bubbles/list` (autocomplete). O modal é overlay desenhado por cima, com `Update` curto-circuitando toda tecla.

## Assets
Nenhum binário. O mascote e a logomarca D-1 são arte em meio-bloco desenhada em runas (`█ ▀ ▄`); o mapa de voxels canônico está em `VOXELS.md` no repositório de marca. Fonte: a do terminal do usuário.

## Files
- `dcode TUI v2.dc.html` — os nove estados, a paleta de marca e as notas de implementação.

## Princípios que atravessam tudo
1. O cliente não guarda estado de sessão.
2. Puro é testável.
3. Responsivo antes de configurável — mas preferência explícita vence sempre.
4. Segurança nunca é silenciosa.
5. Recusar em voz alta.
