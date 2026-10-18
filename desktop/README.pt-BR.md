# DCode Box — o desktop do dcode

🇺🇸 [English version](README.md)

O cliente desktop do dcode, que se chama **DCode Box** (D65; o comando continua
`dcode`): um segundo cliente do mesmo daemon com que o cliente de terminal fala,
em Electron, para ter a mesma cara no macOS, no Linux e no Windows. macOS
primeiro.

> **Estado.** 0.1.0, não publicado. A janela roda sobre um `dcode serve` de
> verdade: anexa ao que responde no socket, ou sobe um. Ela abre na **Crew**
> (D63): à esquerda as conversas do daemon, vivas e terminadas, por projeto e
> com busca; no meio a conversa; à direita o que ela estabeleceu — Mudanças (os
> arquivos, o diff do git, o PR e o CI dele), Pronto (o selo e os critérios) e
> Sessão (pasta, branch, modelo, modo, tokens). A conversa orquestradora ganha
> a aba **Agentes**, a lista viva dos filhos (D64). A **grade** de atenção
> (D30, D31) fica a um seletor na barra do topo: várias conversas lado a lado,
> cada uma trabalhada na sua janela. Em qualquer das duas a sessão é conduzida
> de ponta a ponta — uma nova numa pasta escolhida, um turno com imagens e
> documentos, uma correção, parar, uma aprovação com as respostas da TUI, a
> troca de modelo ou de modo, desfazer, resumir, um comando rodado no terminal
> logo abaixo —, e um `/loop` começa, é acompanhado e para (D54–D56). O ⌘K acha
> qualquer conversa pelo título ou pelo projeto. Escuro por padrão, claro à
> escolha (D35), no laranja do mascote (D49, D52). No navegador ela mostra uma
> gravação, e a janela diz `gravação`.

## Rodando

```bash
cd desktop
npm ci
npm start             # o app, com o renderer no servidor de dev do Vite
npm run dev:renderer  # só o renderer, num navegador, em modo fixture
npm run build         # build de produção de main, preload e renderer em .vite/
```

O `npm start` baixa o binário do Electron no primeiro uso. A janela procura o
daemon em `DCODE_SOCKET`, senão onde o `dcode socket` disser, e anexa a ele; sem
nada respondendo ali, sobe o `dcode serve` — de `DCODE_BIN`, senão
`~/.local/bin/dcode`, senão do `PATH` — e o encerra quando o app fecha. Um daemon
que ela não subiu continua rodando. O `dcode` instalado precisa ser 0.23.0 ou
mais novo, que serve a lista de conversas. No navegador, `?fixture=` abre um
dos estados gravados de `src/fixtures/scenes.ts`, com os controles de janela
desenhados onde o macOS os poria: `crew`, `crew-vazia`, `crew-loop`,
`crew-orquestra`, `crew-orquestra-vazia` e `crew-perguntas` na Crew, e
`02-janela-principal-rodando` e `03-janela-principal-aprovacao` na janela
maximizada da grade.

## Como é feito

- **Processo principal** (`src/main/`) é dono da janela e é o cliente do daemon:
  HTTP e os fluxos de eventos pelo socket Unix, o daemon que subiu, e a pergunta
  antes de fechar com sessão trabalhando. A janela mantém os controles nativos do
  macOS dentro da faixa de título da lateral.
- **Preload** (`src/preload/`) entrega ao renderer uma API estreita — um canal com
  nome por pedido e por notícia, em `src/shared/api.ts` — e nada mais.
- **Renderer** (`src/renderer/`, React) nunca vê Node, o sistema de arquivos ou o
  socket: isolamento de contexto, sandbox e uma CSP que não o deixa conectar a
  lugar nenhum. Geist, Geist Mono e Fragment Mono vêm empacotadas; nada vem da
  rede.
- **Tipos do protocolo** (`src/protocol/generated.ts`) são gerados do
  `internal/protocol` do núcleo com o tygo — `npm run gen:protocol` —, commitados,
  e o CI confere se estão velhos. O desktop lê o núcleo; o núcleo nunca lê o
  desktop.
- **Estado** (`src/state/`) é um redutor puro de eventos do protocolo para o que a
  janela mostra, por sessão, mais as derivações de que a lateral e a barra
  inferior precisam. Lê só campos que o protocolo traz; onde o design pede mais,
  `docs/DECISIONS.md` lista a lacuna.

## O laço do design

A janela é medida contra capturas da janela aprovada, para poder ser iterada
sem se afastar dela — por uma pessoa, ou pelo dcode.

```bash
npm run check:visual
npm run check:visual -- --refs /caminho/para/refs/design/desktop/screenshots
```

Ele constrói o renderer, abre cada estado de referência no Chromium do Playwright
em 1600×960 (movimento reduzido, relógio fixo; os dois estados da grade no
escuro, os da Crew nos dois temas), compara com o PNG de
mesmo nome em `refs/design/desktop/screenshots/` usando o pixelmatch, e escreve a
captura, o diff e o `report.json` em `test-results/visual/`. Imprime uma linha
por estado e reprova quando algum passa do limite. Os estados, o limite e o
raciocínio por trás dele moram em `tests/visual/states.json`. O Chromium do
Playwright se instala uma vez com `npx playwright install chromium`.

Se a janela fala com um daemon é medido do mesmo jeito:

```bash
npm run check:daemon
npm run check:daemon -- steers
```

Ele compila o app e o `dcode` deste checkout e roda cada cenário de
`tests/daemon/` contra um `dcode serve` novo com um modelo roteirizado, abrindo o
app como Electron e dirigindo-o por cliques e teclas. Cada cenário é conferido
contra o que o daemon registrou, não só contra o que a janela desenhou. As falhas
deixam uma captura e os logs em `test-results/daemon/`.

Para o dcode iterar sobre ele, a partir da raiz do repositório:

```
dcode
/loop desktop/docs/loop
```

O loop trabalha até o `desktop/docs/loop/done.toml` ser cumprido — typecheck,
lint, testes, os tipos do protocolo, a verificação visual e a do daemon. Trata
como protegidos as imagens de referência, o `tests/visual/`, o `tests/daemon/` e
a própria pasta do loop: são a régua, e mudá-los não é progresso. O que o loop
está construindo agora, e por que ele precisa de uma sessão com acesso total,
está em `docs/loop/tasks.md`.

**As referências são capturas da janela aprovada** (desde 2026-10-13; antes,
o mock do handoff): a 02 e a 03 da grade e, desde 2026-10-17, de 05 a 12 da
Crew, cada uma nos dois temas. Um visual novo é PNG novo em
`refs/design/desktop/screenshots/`, num PR próprio depois de aprovado — nunca
no PR que muda a janela. Meça o piso das referências novas antes de confiar no
limite antigo (ver `tests/visual/states.json` e D18).

## Mais

- [`AGENTS.md`](AGENTS.md) — comandos, convenções, o que o CI roda
- [`docs/DECISIONS.md`](docs/DECISIONS.md) — por que a janela faz o que faz, e as lacunas do protocolo
- [`CHANGELOG.pt-BR.md`](CHANGELOG.pt-BR.md) — o registro e a versão próprios desta área
