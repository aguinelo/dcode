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

**D52. Laranja: a cor do dcode é a do mascote** (substitui D29 na cor; emenda
D35). Decisão do dono, de 2026-10-13, junto com o mascote (D49): o Laranja
`#FF6A00` é a cor principal do dcode, no lugar do lima `#c6f36b` do grafite.
O resto de D29 fica: escuro por padrão, denso, um acento só — para o que está
trabalhando e para a marca —, um tom para o que espera você, verde e vermelho
só para o que os critérios mediram, a fonte do sistema e cantos de 4px.

- **De onde saem as cores.** Da paleta de 23 tons da folha: o acento
  `#FF6A00`; os neutros `#0A0A0A`, `#1A1A1A`, `#2D2D2D`, `#4B5563`,
  `#6B7280`, `#9CA3AF`, `#D1D5DB` e `#F5F5F2` para chão, linhas e texto; o
  sucesso `#22C55E`; o apoio `#FDE68A` para o que espera você. Os chãos que
  ficam entre dois neutros (`#0F0F0F`, `#141414`, `#111111`, `#ededea`,
  `#f9f9f7`…) são degraus entre eles, porque a janela precisa de mais camadas
  do que a folha tem.
- **Onde a paleta não se lê, um degrau do mesmo tom, e qual:**
  - *Acento no claro:* `#FF6A00` dá 2,87:1 no branco e 2,63:1 no `#F5F5F2`.
    Texto e marcas usam `#B33D00`, laranja escurecido (4,78:1 no pior chão,
    a seleção). O laranja vivo fica onde é preenchimento sob tinta escura: o
    botão de enviar e o "Nova" do Crew, com `#0A0A0A` sobre `#FF6A00` a
    6,90:1, nos dois temas. Branco sobre `#FF6A00` daria 2,87:1 e foi
    descartado.
  - *Espera:* o aviso da folha, `#F59E0B`, fica a poucos graus do laranja, e
    "rodando" e "esperando você" passariam a ser o mesmo tom na lista. A espera
    passa para o amarelo de apoio `#FDE68A` no escuro e, no claro, para
    `#7A5C00`, o mesmo amarelo escurecido até ser lido como texto.
  - *Perigo no escuro:* `#EF4444` dá 4,13:1 na seleção. Fica `#F87171`, um
    degrau mais claro do mesmo vermelho.
  - *Claro:* `#22C55E`, `#16A34A` e `#EF4444` não chegam a 4,5:1 no branco.
    Ficam `#137036` e `#B91C1C`, verde e vermelho escurecidos.
  - *Apagado no escuro da grade:* `#6B7280` dá 3,6:1, e `#9CA3AF` já é o
    `dim`. O `faint` é `#868B95`, um degrau entre os dois.
- **O terminal** (D37) segue o mesmo: a seleção em laranja, e o azul, o roxo e
  o ciano do ANSI no escuro passam a ser os da folha (`#3B82F6`, `#8B5CF6`,
  `#38BDF8`). O fundo da janela antes do primeiro quadro passa a ser
  `#0a0a0a`.

Contraste (WCAG, texto sobre chão; "sel" é a seleção sobre o painel). Todos
acima de 4,5:1:

| grade escura | chrome `#0a0a0a` | panel `#141414` | raised `#1a1a1a` | sunken `#0f0f0f` | sel |
|---|---|---|---|---|---|
| text `#d1d5db` | 13,44 | 12,50 | 11,81 | 13,01 | 10,53 |
| dim `#9ca3af` | 7,80 | 7,26 | 6,86 | 7,55 | 6,11 |
| faint `#868b95` | 5,79 | 5,39 | 5,09 | 5,61 | 4,54 |
| accent `#ff6a00` | 6,90 | 6,42 | 6,06 | 6,68 | 5,41 |
| ok `#22c55e` | 8,69 | 8,08 | 7,64 | 8,41 | 6,81 |
| err `#f87171` | 7,16 | 6,66 | 6,29 | 6,93 | 5,61 |
| warn `#fde68a` | 15,90 | 14,79 | 13,98 | 15,39 | 12,46 |

| grade clara | chrome `#ededea` | panel `#f9f9f7` | raised `#ffffff` | sunken `#f5f5f2` | sel |
|---|---|---|---|---|---|
| text `#1a1a1a` | 14,84 | 16,51 | 17,40 | 15,93 | 14,19 |
| dim `#2d2d2d` | 11,74 | 13,06 | 13,77 | 12,61 | 11,23 |
| faint `#4b5563` | 6,44 | 7,17 | 7,56 | 6,92 | 6,16 |
| accent `#b33d00` | 4,99 | 5,56 | 5,86 | 5,36 | 4,78 |
| ok `#137036` | 5,27 | 5,86 | 6,18 | 5,66 | 5,04 |
| err `#b91c1c` | 5,52 | 6,14 | 6,47 | 5,92 | 5,27 |
| warn `#7a5c00` | 5,33 | 5,93 | 6,25 | 5,72 | 5,10 |

| Crew escura | chrome `#0a0a0a` | panel `#111111` | raised `#1a1a1a` | sunken `#0d0d0d` | sel |
|---|---|---|---|---|---|
| text `#f5f5f2` | 18,13 | 17,29 | 15,93 | 17,79 | 13,85 |
| dim `#d1d5db` | 13,44 | 12,81 | 11,81 | 13,19 | 10,27 |
| faint `#9ca3af` | 7,80 | 7,44 | 6,86 | 7,66 | 5,96 |
| accent `#ff6a00` | 6,90 | 6,58 | 6,06 | 6,77 | 5,27 |
| ok `#22c55e` | 8,69 | 8,29 | 7,64 | 8,53 | 6,64 |
| err `#f87171` | 7,16 | 6,83 | 6,29 | 7,03 | 5,47 |
| warn `#fde68a` | 15,90 | 15,16 | 13,98 | 15,61 | 12,15 |

| Crew clara | chrome `#f5f5f2` | panel `#ffffff` | raised `#f0f0ed` | sunken `#fafaf8` | sel |
|---|---|---|---|---|---|
| text `#0a0a0a` | 18,13 | 19,80 | 17,34 | 18,94 | 16,91 |
| dim `#2d2d2d` | 12,61 | 13,77 | 12,06 | 13,18 | 11,76 |
| faint `#4b5563` | 6,92 | 7,56 | 6,62 | 7,23 | 6,46 |
| accent `#b33d00` | 5,36 | 5,86 | 5,13 | 5,61 | 5,00 |
| ok `#137036` | 5,66 | 6,18 | 5,41 | 5,91 | 5,28 |
| err `#b91c1c` | 5,92 | 6,47 | 5,67 | 6,19 | 5,53 |
| warn `#7a5c00` | 5,72 | 6,25 | 5,48 | 5,98 | 5,34 |

O botão: tinta `#0a0a0a` sobre `#ff6a00`, 6,90. O acento sobre o fundo suave
dele (`accent-soft` sobre o painel) dá 5,54 no escuro e 4,78 no claro. No aviso
de aprovação (o tom de espera a 7% ou 10% sobre o painel), o texto fica em
10,70 e o tom em 12,66 no escuro, e em 14,32 e 5,14 no claro.

**As réguas.** A janela inteira muda de cor, e as referências 02 e 03 mudam
com ela. O dono aceitou recapturá-las neste PR, e só neste: elas foram
capturadas pela própria verificação (`check:visual`), nas condições de sempre
(modo fixture, o Chromium do Playwright, escuro, menos movimento, o relógio
fixo, 1600×960) e enquadradas no anel de 1px das PNGs antigas, como #439 e
#447 fizeram. Antes da recaptura, a janela com o mascote (D49) e o Laranja
media 0,13% (02) e 0,15% (03) contra as capturas antigas: o limiar de cor do
pixelmatch acha o grafite e os novos neutros quase iguais, e o que muda de
verdade é o acento. Depois da recaptura, 0,00% nas duas.

Descartados: manter o lima ao lado do laranja (dois acentos); o laranja como
texto no claro (2,9:1); branco sobre o laranja nos botões (2,9:1); o
`#F59E0B` da folha para a espera (perto demais do acento); seguir os chãos do
grafite e trocar só o acento (a folha traz os neutros, e o dono pediu a paleta
inteira).
