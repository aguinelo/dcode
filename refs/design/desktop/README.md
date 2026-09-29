# Handoff: DCode Desktop — v1 (janela, navegação, loop)

## Overview

Cliente **desktop** do DCode. Fala com o **mesmo daemon** da TUI e da CLI: uma
sessão aberta em qualquer cliente aparece nos outros. Esta primeira versão tem
três telas:

1. **Janela principal:** coluna lateral com projetos → sessões, e **uma sessão
   por vez** ocupando a área toda. O mock mostra dois estados, *rodando com
   delegação* e *esperando aprovação*, alternados ao clicar na lateral.
2. **Navegar entre sessões (`⌘K`):** seletor com filtro, escopos e prévia.
3. **Loop de execução:** sessão que repete o turno até os checks do
   `done.toml` passarem.

Binário, comandos e caminhos continuam `dcode`, minúsculo. Só o **nome do
produto** é DCode.

## About the Design Files

`DCode Desktop v2.dc.html` é uma **referência de design feita em HTML**: mostra
a aparência e o comportamento pretendidos, **não** é código para copiar. A
tarefa é **reproduzir as telas no ambiente do cliente desktop**, com os padrões
dele. Se esse cliente ainda não existe, escolha o framework adequado para um app
que conversa com um daemon Go local (por exemplo Wails ou Tauri com frontend
web) e registre a escolha como decisão de arquitetura. `support.js` está no
pacote só para o arquivo abrir no navegador.

Abra o arquivo no navegador. As telas ficam empilhadas, de cima para baixo:
**Loop de execução → Janela principal → Navegar entre sessões.** O tweak
`theme` (`dark`/`light`) troca o tema.

`DCode Desktop.dc.html` (v1) é uma direção **descartada**. Não implemente.

**Fonte canônica de comportamento:** `docs/specs/` e `docs/brand/`. **Onde este
README e a spec divergirem, a spec vence.** Várias partes desta proposta ainda
**não têm spec**. Elas estão listadas em `SPEC_GAPS.md` e precisam de changelog
de spec **antes** do código.

## Fidelity

**Alta fidelidade** em cores, tipografia, espaçamentos, raios, textos literais,
glifos de estado e durações de animação. **Ilustrativo:** conteúdo de exemplo
(nomes de sessão, arquivos, números), modelos (`MiniMax-M3`, `Opus 5.5`) e o
tamanho fixo de 1600×960 da janela. O app real é redimensionável: a coluna
lateral é fixa e a área da sessão flexível.

---

## Screens / Views

### Janela (comum a todas as telas)

- Janela: fundo `--dc-chrome`, borda 1px `--dc-line`, raio 12px.
- Layout: `row [ lateral 264px fixa | área da sessão flex ]`, e embaixo uma
  **barra inferior de 30px**.
- A área da sessão é um **painel** inset: margem `8px 8px 0 0`, fundo
  `--dc-panel`, borda 1px `--dc-line`, raio 10px. A lateral **não** tem painel,
  fica direto no chrome.
- Topo da lateral (48px): semáforos macOS 12px com gap 8px, padding `0 16px`.
  À direita: alternar lateral e voltar/avançar (`←`/`→`), `--dc-faint`.

### 1. Coluna lateral: projetos → sessões

**Navegação fixa** (padding `4px 8px 10px`, itens com gap 1px):

| Ícone | Rótulo | Tecla |
|---|---|---|
| `+` | Nova sessão | `⌘N` |
| `⌕` | Procurar | `⌘K` |
| `◷` | Rotinas | — |

Item: padding `7px 10px`, raio 7px, 13.5px, ícone numa coluna de 16px
`--dc-dim`, tecla 11px `--dc-faint`. Hover: `--dc-hover`.

**Cabeçalho "Projetos"** (12px `--dc-faint`, padding `14px 10px 6px`): rótulo,
depois `recolher tudo`/`expandir tudo` (11.5px, hover `--dc-text`), depois `+`
(adicionar projeto).

**Linha de projeto** (padding `7px 10px`, raio 7px, 13.5px peso 500
`--dc-text`):
- chevron `▾` aberto / `▸` fechado (10px, coluna de 16px);
- nome, com reticências;
- meta à direita (11px): **só quando recolhido**. Mostra `n espera` em
  `--dc-warn` se houver sessão esperando, senão `n rodando`, senão o total de
  sessões em `--dc-faint`;
- `⋯` (18px), **visível só no hover** ou com o menu aberto.

**Linha de sessão** (padding `6px 10px 6px 16px`, raio 7px, 13px):

| Estado | Marca (6px, coluna 16px) | Texto | À direita |
|---|---|---|---|
| rodando | círculo `--dc-accent`, pulso `dcBreath` 2.4s | `--dc-text` | — |
| loop | círculo `--dc-accent`, pulso | `--dc-text` | `↻ 8/100` `--dc-faint` |
| esperando aprovação | losango (quadrado 45°, raio 1.5px) `--dc-warn`, **estático** | `--dc-text` | `aprovar` `--dc-warn` |
| concluída / parada | nenhuma (espaço reservado) | `--dc-dim` | idade (`2h`, `ontem`, `1sem`) |
| selecionada | + fundo `--dc-sel` | `--dc-emph` | igual |

**Rodapé da lateral:** avatar 26px (iniciais 10.5px/600), nome 13px, `⚙`.
Borda superior `--dc-line-soft`, padding `12px 16px`.

#### Ações de projeto

- **Recolher/expandir:** clique na linha do projeto. Também pelo menu `⋯` e
  por "recolher tudo".
- **Renomear:** clique duplo no nome, ou `⋯ → Renomear`. Vira um input
  inline (borda 1px `--dc-accent`, raio 6px, padding `4px 7px`, 13.5px/500).
  `↵` ou blur salva; `esc` cancela; texto vazio mantém o nome anterior.
  **Renomear muda só o rótulo de exibição, nunca o diretório.**
- **Reordenar:** arrastar a linha do projeto. Durante o arraste o item fica em
  opacidade 0.4, e o alvo mostra uma **borda superior de 2px `--dc-accent`**.
  Soltar insere o item antes do alvo. Também pelo menu: `Mover para cima` e
  `Mover para baixo` (aparecem só quando fazem sentido).
- **Menu `⋯`:** popover de 196px, top 34px/right 6px, fundo `--dc-panel`,
  borda `--dc-line`, raio 9px, sombra `0 12px 32px rgba(0,0,0,.28)`, padding
  5px. Itens: padding `7px 9px`, raio 6px, 13px. Itens: *Renomear* (dica
  `2× clique`), *Recolher sessões*/*Expandir sessões*, *Mover para cima*, *Mover
  para baixo*.

Ordem, nomes e estado recolhido são **preferências do cliente**: persistir
localmente, não no daemon.

### 2. Sessão (área principal)

**Uma sessão por vez, ocupando a área toda.** Não há split.

**Cabeçalho (48px, padding `0 18px`):** título 14px/500 `--dc-emph` com
reticências, `⌄` (menu da sessão), espaço, e o chip de branch: mono 11.5px,
borda `--dc-line`, raio 6px, padding `4px 9px`, com `+38` em `--dc-ok` e `−0`
em `--dc-faint` (ou `--dc-err` quando houver remoções).

**Fluxo:** coluna centralizada, `max-width: 720px`, padding `0 24px 16px`, gap
16px, **ancorada no fim** (auto-follow).

- **Mensagem do usuário:** alinhada à direita, max 85%, padding `10px 14px`,
  raio 14px, fundo `--dc-raised`, 14px/1.55 `--dc-emph`.
- **Linha de ferramenta** (compacta, não é card): `✓` `--dc-ok` (coluna 14px),
  nome da ferramenta em mono 12.5px `--dc-text`, alvo em mono 12.5px
  `--dc-dim`, espaço, e à direita o resultado resumido em 13px `--dc-dim`
  (`11 matches · 4 arquivos`, `96 linhas`, `criado, 54 linhas`).
- **Texto do modelo:** 14px/1.65 `--dc-text`, `text-wrap: pretty`. Código
  inline: mono 12.5px, padding `1px 5px`, raio 4px, fundo `--dc-raised`.
- **Card de delegação:** borda `--dc-line`, raio 10px.
  - Cabeçalho (padding `11px 14px`, 13px): ponto âmbar pulsando, `Delegou a 4
    filhos` 500 `--dc-emph`, `propriedade disjunta` `--dc-faint`, espaço, `2 de
    4` `--dc-dim`.
  - Linhas: grid `14px 64px 1fr 56px 118px`, gap 10px, padding `8px 14px`,
    borda superior `--dc-line-soft`, 12.5px. Colunas: glifo, nome (500),
    diretório dono (mono 11.5px `--dc-faint`), barra de progresso de 3px, meta.
  - Glifos: `✓` `--dc-ok` concluído · `●` `--dc-accent` rodando · `⊘`
    `--dc-err` sem resposta (a meta também fica em `--dc-err`: `não
    respondeu`). A barra usa a cor do glifo e anima a largura em `.9s
    cubic-bezier(.3,.8,.3,1)`.
- **Linha de atividade** (só enquanto o turno roda): `●` âmbar pulsando, verbo
  capitalizado + `…` (500 `--dc-emph`), o fato atual em `--dc-dim` com
  reticências (`charlie lendo internal/store/sqlite.go`), e à direita tempo e
  tokens (12px `--dc-faint`, `1m42s · 18.4k`). O verbo roda a cada 2.4s entre
  `delegando`, `lendo`, `conferindo`, `pensando` e depois vem de
  `tui.activity_verbs`.

**Card de aprovação:** borda `--dc-warn-line`, fundo `--dc-warn-bg`, raio 12px.
- Cabeçalho: losango `--dc-warn`, `Rodar este comando?` 14px/500, espaço,
  `esperando · 0:48` 12px `--dc-warn` (**relógio ao vivo**, conta desde o
  pedido).
- Comando: bloco mono 12.5px `--dc-emph`, fundo `--dc-panel`, borda
  `--dc-line`, raio 8px, padding `10px 12px`, margem `0 16px`. Prefixo `$` em
  `--dc-faint`.
- Motivo: 12.5px/1.55 `--dc-dim`, com os recursos pedidos em `--dc-text` (ex.:
  *rede*, `package.json`, `node_modules/`) e a política violada.
- Opções (borda superior `--dc-warn-line`, padding `4px 8px 8px`): linhas
  numeradas com caixinha de 18px e raio 5px. `1 Permitir uma vez ↵` (pré-
  selecionada, `--dc-sel`), `2 Permitir nesta sessão`, `3 Negar esc`. Teclas
  `1`/`2`/`3`, `↵` e `esc`.

**Composer** (max 720px, padding `13px 14px 10px`, raio 14px, fundo
`--dc-sunken`, borda `--dc-line`):
- Placeholder conforme o estado: rodando → *Mensagem entra na fila depois deste
  turno*; esperando aprovação → *Ou diga o que fazer em vez disso* (com cursor
  âmbar piscando, `dcCaret` 1.1s step-end); loop → *Orientar a próxima iteração
  — entra no início da iteração N*.
- Linha inferior (12.5px `--dc-dim`, gap 10px): `+` anexar, chip `workspace-write
  · on-request` (borda `--dc-line`, raio 6px, padding `3px 8px`), espaço, modelo
  + `⌄`, medidor de contexto (`41%` `--dc-faint`), e o botão redondo de 28px.
  Rodando: **parar** (quadrado de 9px, fundo `--dc-raised`). Idle: **enviar**
  `↑` (fundo `--dc-btn`, texto `--dc-btn-ink`).

**Barra inferior (30px, 12px `--dc-dim`, padding `0 16px`, gap 16px):**
- Janela principal: `● daemon` (ponto 6px `--dc-ok`), `v0.21.1` `--dc-faint`,
  espaço, `1 rodando`, `1 esperando você` `--dc-warn`.
- Loop: o mesmo à esquerda; à direita `<Verbo>… iteração 8 · 15m22s`.

### 3. Navegar entre sessões (`⌘K`)

Overlay sobre a janela: o fundo fica com `blur(3px)` e opacidade 0.55, mais um
véu `rgba(0,0,0,.28)`. Modal de 920×560, top 120px, fundo `--dc-panel`, borda
`--dc-line`, raio 14px, sombra `0 24px 70px rgba(0,0,0,.45)`.

- **Topo:** `⌕`, input *Ir para sessão…* (15px `--dc-emph`, sem borda, com
  autofocus) e os chips de escopo `todas` · `ativas` · `<projeto atual>`
  (ativo: fundo `--dc-sel`, borda `--dc-line`, texto `--dc-emph`).
- **Lista (520px, borda direita `--dc-line-soft`):** grupo **Ativas** primeiro
  (rodando, loop, esperando), depois um grupo por projeto. Cabeçalho de grupo
  12px `--dc-faint` com a contagem à direita. Linha: marca de estado (mesma
  regra da lateral), título, estado (`rodando`/`aprovar`/idade) e tecla `⌘1…⌘9`
  das ativas. Selecionada: `--dc-sel`.
- **Prévia (flex, fundo `--dc-sunken`, padding `20px 22px`):** título 15px/500,
  estado longo colorido (ex.: `✓ verified · 2 checks` `--dc-ok`, `Esperando
  aprovação · bash` `--dc-warn`), grid `84px 1fr` com workspace, branch,
  modelo e turnos (valores de caminho/branch em mono), e o **último evento**.
- **Rodapé:** `↑↓ navegar` · `↵ abrir` · `tab escopo` … `esc fechar`.
- **Vazio:** *Nenhuma sessão com "<q>". ↵ cria uma nova com esse pedido.*

Teclado: `↑↓` movem sem dar a volta; hover também seleciona; `tab` gira o
escopo; digitar filtra pelo título e volta o cursor ao topo; `esc` limpa o
filtro e, com filtro vazio, fecha.

### 4. Loop de execução

Especificação funcional completa (comando, flags, controles, regras de parada):
**`LOOP.md`**, neste pacote. Aqui fica só o layout.

**Cabeçalho:** `↻` `--dc-accent`, objetivo curto 14px/500, `loop · iteração 8
de 100` 12px `--dc-dim`, e o chip de branch. Borda inferior `--dc-line-soft`.

**Corpo:** `row [ coluna principal flex | painel 340px ]`.

**Faixa superior** (padding `20px 28px 18px`, borda inferior):
- `objetivo`: rótulo 12px `--dc-faint` numa coluna de 58px + o texto completo
  em 14px `--dc-emph`.
- **Placar de iterações:** uma coluna por iteração (flex 1, max 84px, gap 6px,
  raio 6px, clicável, selecionada em `--dc-sel`). Rótulos à esquerda numa
  coluna de 72px.
  - Barra: altura = testes passando / total × 100px, raio `3px 3px 0 0`, número
    em cima (11px). Cor: `--dc-dim` com progresso, **`--dc-warn` sem
    progresso**, `--dc-ok` quando tudo passa, `--dc-accent` pulsando na
    iteração atual (8px, número `…`). Iterações não selecionadas ficam em
    opacidade 0.55. Linha tracejada `--dc-line` marca o total (`12 testes`).
  - Linhas de check abaixo (18px cada): `go vet`, `specguard` → `✓` `--dc-ok`
    / `✗` `--dc-err` / `·` `--dc-faint` na iteração atual. **Uma linha por
    check do `done.toml`**, sem nomes fixos.
  - Linha `iteração`: números 11.5px, o selecionado em 600 `--dc-emph`, e as
    iterações sem progresso levam `=` `--dc-warn` ao lado.
- **Detalhe da iteração selecionada** (padding `18px 28px`, gap 14px): `Iteração
  N` 15px/500, delta (`+2 passando · 10/12` `--dc-ok` / `sem progresso · 8/12`
  `--dc-warn` / `em andamento` `--dc-accent`), e à direita `2m02s · 26k tok ·
  +31 −12`. Depois o resumo (14px/1.65) e um card com as linhas de ferramenta.
  A última linha é sempre `done  go test · go vet · specguard  pronto | não
  pronto → próxima`.
- **Composer** no rodapé da coluna principal, com o placeholder do loop.

**Painel (340px, fundo `--dc-sunken`, borda esquerda, padding 20px, gap 22px):**
- **Pronto quando** (`done.toml` em mono à direita): uma linha por check, com
  estado atual (`●` `--dc-warn` parcial `11/12`, `✓` `--dc-ok` passa). Nota
  12px `--dc-faint`: *Roda no fim de cada iteração, na árvore inteira. O selo
  vem daqui, nunca de uma frase do modelo.*
- **Limites:** Iterações `8 de 100` · Tokens `203k de 1M` · Tempo `15m de 2h`,
  cada um com barra de 3px `--dc-dim`.
- **Para quando:** `✓` *Todos os checks passam* → conclui; `=` *3 iterações
  seguidas sem progresso* → pausa e pergunta; `■` *Algum limite acaba* → para.
- No fim do painel, dois botões: **Parar depois desta iteração** (borda
  `--dc-line`, fundo `--dc-panel`, raio 8px) e **Interromper agora `esc`**
  (ghost).

---

## Interactions & Behavior

- **Selecionar sessão na lateral** troca a área principal; a sessão anterior
  **continua rodando** no daemon.
- **`⌘1…⌘9`** abrem as sessões ativas na ordem do grupo Ativas.
- **Auto-follow** no fluxo: rolar para cima desliga, voltar ao fim religa.
- **Animações** (sempre discretas; **concluído nunca anima**):
  - `dcBreath`: opacidade 1 → .35 → 1, 2.4s ease-in-out, infinita. Usada no
    ponto de rodando e loop, na atividade e na iteração atual.
  - `dcCaret`: opacidade 0 em 50%, 1.1s step-end. Cursor do composer.
  - Barras de progresso: `width .9s cubic-bezier(.3,.8,.3,1)`.
  - Respeitar `prefers-reduced-motion`: sem pulso; o estado continua legível
    pela forma do glifo.
- **Cor nunca é o único sinal:** rodando = círculo, esperando = losango,
  falha = `⊘`, sem progresso = `=`. O cursor do seletor é um fundo **e** a
  posição do teclado.
- **Hover:** `--dc-hover` em linhas e itens clicáveis; `⋯` só aparece no
  hover.
- **Relógios ao vivo:** espera de aprovação (`m:ss`) e tempo do loop
  (`XmYYs`), atualizando a cada 1s.

## State Management

Estado vindo do **daemon, derivado do log de eventos** (nunca de polling do
disco):
- sessões: `id, título, projeto, estado (running | loop | awaiting_approval |
  idle | interrupted), branch, diff (+/−), modelo, turnos, último evento, selo
  (verified | unverified | not_verified), idade`;
- turno: eventos de ferramenta, filhos da delegação (`nome, dono, progresso,
  estado, leituras/escritas`), verbo e fato de atividade, tempo e tokens,
  contexto (%);
- aprovação pendente: `comando, recursos pedidos, política, desde`;
- loop: `objetivo, iteração atual, limites e consumo, checks do done.toml e,
  **por iteração**, o resultado de cada check com contagem (ex.: go test
  11/12)`, resumo, ferramentas, duração, tokens, diff. ← **pedido de
  protocolo** (ver `SPEC_GAPS.md`).

Estado **local do cliente** (persistir no app):
- `order: string[]` dos projetos, `labels: {projectId: string}`,
  `collapsed: {projectId: bool}`;
- sessão ativa na janela, iteração selecionada no loop, escopo e filtro do
  `⌘K` (efêmeros).

## Design Tokens

Fontes: **Geist** (400/500/600/700) para a interface, **Geist Mono** (400/500)
para caminhos, comandos, branches e diffs.

| Token | Escuro | Claro | Uso |
|---|---|---|---|
| `--dc-canvas` | `#0a0a0a` | `#d9d7d2` | fundo atrás da janela (só no mock) |
| `--dc-chrome` | `#161616` | `#f1f0ed` | janela, lateral |
| `--dc-panel` | `#1c1c1c` | `#ffffff` | painel da sessão, modais, popovers |
| `--dc-raised` | `#232323` | `#f7f6f4` | bolha do usuário, código inline |
| `--dc-sunken` | `#191919` | `#fafaf9` | composer, prévia, painel do loop |
| `--dc-line` | `#2a2a2a` | `#e4e2dd` | bordas |
| `--dc-line-soft` | `#242424` | `#eceae6` | divisórias internas |
| `--dc-text` | `#e6e6e6` | `#262626` | texto |
| `--dc-emph` | `#fafafa` | `#0f0f0f` | títulos, ênfase |
| `--dc-dim` | `#a1a1a1` | `#5f5f5f` | secundário |
| `--dc-faint` | `#6f6f6f` | `#8f8f8f` | terciário, teclas |
| `--dc-hover` | `rgba(255,255,255,.045)` | `rgba(0,0,0,.04)` | hover |
| `--dc-sel` | `rgba(255,255,255,.07)` | `rgba(0,0,0,.06)` | seleção |
| `--dc-accent` | `#e0a030` | `#b87d1e` | **só "rodando"**: pulso, loop, caret |
| `--dc-ok` | `#6fbf87` | `#2f7a48` | concluído, verified, adições |
| `--dc-err` | `#e07b7f` | `#b3434a` | falha, remoções |
| `--dc-warn` | `#e3c26b` | `#8a6a0c` | **só "esperando você"** e sem progresso |
| `--dc-warn-bg` | `rgba(227,194,107,.06)` | `rgba(184,125,30,.06)` | card de aprovação |
| `--dc-warn-line` | `rgba(227,194,107,.35)` | `rgba(184,125,30,.35)` | borda do card de aprovação |
| `--dc-btn` / `--dc-btn-ink` | `#fafafa` / `#111` | `#111` / `#fff` | botão enviar |

Regra de cor: **âmbar = trabalhando, amarelo = precisa de você**; todo o
resto é neutro. O vermelho `danger` fica reservado ao modo full-access (tela
ainda não desenhada).

**Tipografia:** 15 (títulos de modal e iteração) · 14 (título da sessão, corpo,
composer) · 13.5 (itens da lateral) · 13 (linhas de ferramenta, sessões) ·
12.5 (metas, código em bloco) · 12 (rótulos, barra inferior) · 11.5 · 11
(teclas, contagens). Corpo com line-height 1.55–1.65. Pesos 400/500 (600 só na
iteração selecionada e no avatar).

**Raios:** 14 (composer, bolha, modal) · 12 (janela, card de aprovação) · 10
(painel da sessão, cards) · 9 (popover) · 8 (bloco de comando, botões) · 7
(linhas da lateral) · 6 (chips, inputs, colunas do placar) · 5 (caixinha
numerada) · 4 (código inline).

**Espaçamento:** gaps 1 · 6 · 8 · 10 · 14 · 16 · 22 px; paddings de linha
`6–8px 10px`; cabeçalhos com 48px de altura; barra inferior de 30px.

**Sombras:** modal `0 24px 70px rgba(0,0,0,.45)`; popover `0 12px 32px
rgba(0,0,0,.28)`.

## Assets

Nenhuma imagem. Os glifos são caracteres Unicode (`● ◆ ✓ ✗ ⊘ = ↻ ■ ▾ ▸ ⋯ ⌕ ◷ ⌘ ↵
⎋`). Fontes Geist e Geist Mono (Google Fonts, licença OFL). No app, empacotar
localmente.

## Files

- `screenshots/`: referência visual, 1600×960, tema escuro.
  - `01-loop-de-execucao.png`: loop na iteração 8, com a 7 selecionada.
  - `02-janela-principal-rodando.png`: sessão rodando com delegação a 4 filhos.
  - `03-janela-principal-aprovacao.png`: sessão esperando aprovação.
  - `04-navegar-entre-sessoes.png`: seletor `⌘K`.
- `DCode Desktop v2.dc.html`: as três telas (abrir no navegador).
- `support.js`: runtime para o arquivo HTML abrir. **Não** vai para produção.
- `LOOP.md`: especificação funcional do loop (entrada, flags, controles,
  regras de parada, telas faltantes).
- `SPEC_GAPS.md`: o que este design assume e ainda **não está em
  `docs/specs/`**. Resolver antes de codar.
