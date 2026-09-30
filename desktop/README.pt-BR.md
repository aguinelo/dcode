# DCode desktop

🇺🇸 [English version](README.md)

O cliente desktop do dcode: um segundo cliente do mesmo daemon com que o cliente
de terminal fala, em Electron, para ter a mesma cara no macOS, no Linux e no
Windows. macOS primeiro.

> **Estado.** 0.1.0, não publicado. Esta versão desenha a janela principal do
> design v2 — a lateral de projetos e sessões, uma sessão por vez, o card de
> aprovação, o composer e a barra inferior — a partir de eventos de protocolo
> **gravados**. Ainda não se conecta a um daemon; a barra inferior diz
> `gravação` para ninguém confundir com um ao vivo. Ligar o processo principal ao
> socket do daemon é a próxima versão.

## Rodando

```bash
cd desktop
npm ci
npm start             # o app, com o renderer no servidor de dev do Vite
npm run dev:renderer  # só o renderer, num navegador, em modo fixture
npm run build         # build de produção de main, preload e renderer em .vite/
```

O `npm start` baixa o binário do Electron no primeiro uso. A janela abre na
gravação. No navegador, `?fixture=02-janela-principal-rodando` ou
`?fixture=03-janela-principal-aprovacao` abre um dos estados de referência do
design, com os controles de janela desenhados onde o macOS os poria.

## Como é feito

- **Processo principal** (`src/main/`) é dono da janela e, na próxima versão, do
  socket Unix do daemon. A janela mantém os controles nativos do macOS dentro da
  faixa de título da lateral.
- **Preload** (`src/preload/`) entrega ao renderer uma API estreita — a
  plataforma e quem usa — e nada mais.
- **Renderer** (`src/renderer/`, React) nunca vê Node, o sistema de arquivos ou o
  socket: isolamento de contexto, sandbox e uma CSP que não o deixa conectar a
  lugar nenhum. Geist e Geist Mono vêm empacotadas; nada vem da rede.
- **Tipos do protocolo** (`src/protocol/generated.ts`) são gerados do
  `internal/protocol` do núcleo com o tygo — `npm run gen:protocol` —, commitados,
  e o CI confere se estão velhos. O desktop lê o núcleo; o núcleo nunca lê o
  desktop.
- **Estado** (`src/state/`) é um redutor puro de eventos do protocolo para o que a
  janela mostra, por sessão, mais as derivações de que a lateral e a barra
  inferior precisam. Lê só campos que o protocolo traz; onde o design pede mais,
  `docs/DECISIONS.md` lista a lacuna.

## O laço do design

A janela é medida contra as próprias capturas do design, para poder ser iterada
até bater — por uma pessoa, ou pelo dcode.

```bash
npm run check:visual
npm run check:visual -- --refs /caminho/para/refs/design/desktop/screenshots
```

Ele constrói o renderer, abre cada estado de referência no Chromium do Playwright
em 1600×960 (tema escuro, movimento reduzido, relógio fixo), compara com o PNG de
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

**Uma versão nova do Claude Design** é PNG de referência novo em
`refs/design/desktop/screenshots/`. O loop então mostra a distância até ele.
Meça o piso do próprio design novo antes de confiar no limite antigo (ver
`tests/visual/states.json`).

## Mais

- [`AGENTS.md`](AGENTS.md) — comandos, convenções, o que o CI roda
- [`docs/DECISIONS.md`](docs/DECISIONS.md) — por que a janela faz o que faz, e as lacunas do protocolo
- [`CHANGELOG.pt-BR.md`](CHANGELOG.pt-BR.md) — o registro e a versão próprios desta área
