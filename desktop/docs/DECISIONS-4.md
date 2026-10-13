# Decisões do desktop — D49 em diante

A continuação de [`DECISIONS-3.md`](DECISIONS-3.md), que vai de D39 a D53 (sem
D49 e D52), de [`DECISIONS-2.md`](DECISIONS-2.md), de D21 a D40, e de
[`DECISIONS.md`](DECISIONS.md), que tem a introdução, D1 a D20, as decisões em
aberto, as lacunas e os pedidos ao núcleo. A numeração é uma só; D49 veio para cá
porque o DECISIONS-3 passaria de 500 linhas com ela.

**D49. O mascote entra no app: o ícone e o estado de cada conversa.** Decisão
do dono, de 2026-10-13: o mascote da folha de referência
(`refs/design/brand/mascote.webp`) — um robô de blocos arredondados empilhados,
a cabeça com um olho grande só, braços e pés pretos e as marcas de movimento —
entra agora, no ícone do app, nos estados e na paleta. A paleta é a decisão
seguinte (D52); esta é o desenho e onde ele aparece.

- **Vetor desenhado à mão, nunca recorte da folha.** O ícone é
  `assets/icon/icon.svg`: o mascote de frente, em Laranja (`#FF6A00`, o realce
  `#FB923C`, o olho `#1A1A1A`/`#F5F5F2`, braços e pés `#0A0A0A`), sobre um
  ladrilho grafite `#2D2D2D` — no preto puro os braços somem. `npm run
  gen:icons` (`scripts/icons.mjs`) rasteriza o SVG no mesmo Chromium da régua
  visual e monta o `.icns` (16 a 1024), o `.ico` (16 a 256) e o `.png` (512)
  sem ferramenta de plataforma, para que o conjunto saia igual em qualquer
  máquina. O `forge.config.ts` passa `assets/icon/icon` ao `packagerConfig.icon`:
  o empacotador pega o `.icns` no macOS e o `.ico` no Windows; no Linux ele não
  lê ícone, e o `.png` espera o maker que vier.
- **O estado, em pose.** Onde a janela já mostrava o estado de uma conversa com
  um caractere (`●`, `◆`, `✓`, `✗`, `⚠`, `·`) — a janela da grade, a
  maximizada, a thread e as linhas de sessão do Crew —, agora mostra a cabeça do
  mascote, no tom de antes (a cor continua dizendo o que dizia), e o olho e uma
  marca ao lado dizem de novo em forma, para quem não lê a cor:

  | estado | quando | olho e marca |
  |---|---|---|
  | Planejando… | rodando, sem chamada ou lendo | olha para cima, dois pontos subindo |
  | Codando… | rodando, escrevendo, delegando ou outra | as duas marcas de movimento da folha |
  | Verificando… | rodando um comando (`bash`, `process`) | olha para o lado, uma lupa |
  | Deu certo! | último turno `passed` | um ✓ no olho, as marcas |
  | Bug encontrado… | último turno `failed` | o olho riscado, um zigue-zague |
  | Esperando você | aprovação pendente | olha para você, um ! |
  | Hum… | `stale` ou `unavailable` | a pálpebra a meio |
  | Parado | nada medido | só o olho |

  Lido só do que a janela já dobra (`src/state/mascot.ts`): o estado da
  conversa, a fase da chamada que roda agora (D14 — `conferindo` é o verbo da
  TUI para comando) e o selo do último turno (D13). Nenhum campo novo: o
  protocolo não diz "verificando", e a janela não finge que diz.
- **Legível pequeno.** Num quadro de 16, a cabeça ocupa o canto de baixo à
  esquerda e a marca o de cima à direita; desenhado a 12, 14 e 16 px (linhas
  do Crew, janela, thread). Cada glifo tem o nome do estado como rótulo e
  título. O que respira continua sendo o `dc-breath` de antes, que o
  `prefers-reduced-motion` desliga; a pose nunca depende do movimento.
- **Tela vazia e aviso.** A grade sem nada e a thread do Crew sem conversa em
  foco mostram o mascote inteiro, de frente. Os avisos (toasts) — que são
  sempre uma falha ou uma recusa — levam a cabeça de "Bug encontrado…", como o
  aviso da folha.
- **As réguas.** A janela maximizada da referência 02 e a 03 mostram o glifo da
  conversa em foco, então elas mudam com isto: 0,07% (02) e 0,04% (03) contra
  as capturas atuais, dentro do limite de 1,25% mas não os 0,00% de sempre. As
  referências não são recapturadas aqui: o PR da paleta (D52), empilhado sobre
  este, muda a janela inteira e recaptura 02 e 03 uma vez só, com a
  autorização do dono. Os dois entram juntos, este primeiro.

Descartados: recortar o mascote da folha (raster, e a folha é renderização 3D
que não fica legível a 16 px); glifos diferentes só na cor (quem não lê a cor
perderia o estado, que hoje a forma do caractere dá); inventar um estado
"verificando" que o protocolo não carrega; o mascote inteiro nas linhas (a 12
px ele vira uma mancha laranja). Ainda não: a marca da Crew (o "d" do trilho) e
as bolinhas da lateral e das abas continuam como estão.
