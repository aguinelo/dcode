# Changelog — desktop

🇺🇸 [English version](CHANGELOG.md)

O registro próprio do cliente desktop, versionado à parte do núcleo: tags
`desktop-vX.Y.Z`, a partir de 0.1.0. Nada foi publicado ainda. O que mudou e por
quê, uma entrada cada; o detalhe de uma decisão está em `docs/DECISIONS.md`.

---

## Não publicado

- **O plano fica fixo como checklist enquanto o turno roda (D57).** O plano
  que o turno em curso revisou por último (`plan.updated`) fica acima do
  fluxo, fora da rolagem, e o fluxo o deixa de fora: uma janela da grade dá a
  ele uma linha ("passo 2 de 3 · ..."), a conversa maximizada e a Crew a
  checklist inteira, com o passo corrente num anel girando. Quando o turno
  termina, ele volta ao fluxo, onde estava. Antes, o plano rolava para fora da
  vista justamente enquanto andava. O bloco do plano agora guarda o turno que o
  revisou, porque o evento não traz nenhum. A gravação não tem plano; as telas
  02 e 03 ficam a 0,00%.
- **A régua do daemon roda um loop (passo 9 da proposta do loop).** Dois
  cenários entram no `check:daemon`, que passa a ter 13: `loop-qualifies`
  começa um loop por "Nova sessão" → "↻ loop" numa pasta que não declara
  critério ("0 · qualifica antes"), vê a régua do loop na janela ler
  `qualifying`, o card do `done.proposed` esperar em `awaiting_review` sem nada
  trabalhado, "Começar o trabalho" abrir a sessão de trabalho, as luzes dos
  critérios andarem com o primeiro `done.checked` (✓ ola, ✗ tchau) antes de o
  turno terminar, e o veredito "1 pasta trabalhada · 3 de 3 critérios";
  `loop-stops` digita `/loop specs/ola` num campo (nenhum turno daquela sessão
  começa) e "Parar agora" interrompe o turno de trabalho e encerra o loop
  `stopped`, sem nada escrito. Cada passo é conferido contra o `/v1/loops` e o
  log de eventos do daemon, com o modelo roteirizado segurando as respostas
  entre as quais a janela é lida. Antes, nada conduzia um loop por um daemon de
  verdade a partir da janela. `tests/daemon/**` é caminho protegido do
  `docs/loop/done.toml`; o dono autorizou mudá-lo só neste PR (2026-10-15). O
  `docs/loop/tasks.md`, "O que a régua lê", também é protegido e não foi
  autorizado: os atributos que os cenários do loop leem (`data-pane-id`,
  `data-loop-id`, `data-loop-state`, `data-loop-review`) estão listados no topo
  de `tests/daemon/loop.mjs` até aquela tabela ser atualizada.
- **O `generated.ts` traz os campos novos do loop.** O `LoopRun` do núcleo
  ganha `args`, e cada `LoopSpecRun` o que a pergunta achou (`criteria`,
  `unmet`, `unavailable`, `error`) e o que a proposta escreveu
  (`proposal_criteria`, `proposal_summary`), para a folha do loop e o card de
  revisão os desenharem sem medir de novo. Regenerado, e o decodificador do
  loop os aceita como opcionais (`criteria` ausente é zero); nada no app os
  desenha ainda.
- **A pergunta ao fechar conta os loops vivos (D55, emenda D19).** Fechar o
  app para o daemon que ele subiu, e um loop vive nesse daemon: termina junto
  e não volta. A pergunta antes de fechar agora conta os loops vivos ao lado
  das sessões que trabalham e diz isso ("Fechar o DCode Box e encerrar 1
  loop?"); antes, um loop esperando revisão, sem sessão trabalhando, era
  encerrado sem uma palavra.
- **A definição de pronto proposta é revisada antes do trabalho (D56).**
  Enquanto um loop espera em `awaiting_review`, a régua dele mostra um card
  feito do `done.proposed` como o daemon mediu: o arquivo, cada critério com a
  classe (aceitação, regressão, quebrado), o que rodá-lo fez, o comando, o
  porquê do proponente e a saída; a discordância entre o proponente e a
  medição, critério quebrado e conjunto sem nenhum vermelho são avisados; os
  caminhos protegidos aparecem. Um objetivo que qualificou várias pastas ganha
  uma proposta por pasta. "Abrir no editor" pede ao processo principal, que só
  abre um arquivo que o daemon listou como proposta daquele loop. "Começar o
  trabalho" continua o loop (`POST /v1/loops/{id}/continue`); quando outro
  cliente respondeu antes, `loop_not_awaiting_review` aparece com a razão do
  daemon. A aba Pronto da Crew ganha uma seção Loop. Antes, a revisão só
  existia na TUI. As telas 02 e 03 não têm loop e ficam a 0,00%.
- **Um loop começa e para pela janela (D55).** `/loop <pasta|objetivo>
  [--protect glob]` em qualquer campo vai ao daemon como foi digitado
  (`POST /v1/loops`), e está no menu "/" e na ajuda; "↻ loop" na janela de
  sessão nova abre a folha "Começar loop", com as pastas que o daemon lista e o
  que cada uma declara, ou um objetivo, e o que proteger, mandando o mesmo texto
  que a pessoa digitaria. Um loop ocupa uma janela da grade que segue a sessão
  corrente dele e fica no lugar de pasta em pasta (Q5 A, emenda D31); uma régua
  no topo — só quando há loop — diz o objetivo, o estado, em que pasta está de
  quantas, a fase de cada pasta a um clique da conversa dela, o veredito, o erro
  e os avisos, com "Parar agora" e "Parar depois desta pasta". O processo
  principal guarda o fluxo de loops ao lado do de conversas. Recusas
  (`loop_already_running` e as outras) aparecem com a razão do daemon, e um
  loop que falha é dito uma vez. Antes, o daemon rodava um loop e a janela não
  conseguia começar, ver nem parar. As telas 02 e 03 não têm loop e ficam a
  0,00%.
- **Sessões e conversas são lidas com `loop_id`.** O núcleo agora roda o
  `/loop` e marca as sessões que um loop abre (passo 4 de
  `docs/proposals/loop.md`); o `generated.ts` é regenerado com os tipos do
  loop, e os decodificadores de sessão e de conversa aceitam o campo. Nada no
  app o usa ainda: começar e acompanhar um loop são os passos 7 e 8.
- **As luzes dos critérios andam a cada conferência (D54).** Enquanto o turno
  roda, as luzes da janela da grade, da maximizada e da aba Pronto da Crew são
  as do último `done.checked` dele, não as do último turno que terminou, com
  "ciclo N" ao lado e, quando para, "parado s de L" contra o teto que o evento
  traz. Quando o turno termina, o `turn.completed` volta a mandar. Conferência
  que quebrou um critério e fez o laço devolver trabalho diz isso no fluxo
  ("⟲ Ciclo 2 voltou: e2e quebrou, desfeito src/cart.ts"); antes, as luzes
  ficavam paradas até o fim do turno e um desfazer era invisível. As telas 02 e
  03 não têm conferência em curso e ficam a 0,00%.
- **O desktop lê `done.checked`, e ainda não o mostra.** O núcleo agora anuncia
  cada conferência da definição de pronto dentro do turno (passo 3 de
  `docs/proposals/loop.md`), e tipo que o protocolo acrescenta reprova os
  testes deste app até ser decodificado. Ele é decodificado e conferido contra
  os golden files do núcleo; as luzes dos critérios continuam a mudar no
  `turn.completed`, e movê-las a cada conferência é o passo 6, uma mudança
  própria.
- **O `done.proposed` é lido na forma que o núcleo agora emite.** O daemon
  anuncia uma proposta escrita (passo 2 de `docs/proposals/loop.md`) com o
  caminho do `done.toml` e sem `turn_id`, `round` e `expires_at`, porque nada
  espera resposta. O decodificador segue o `generated.ts` regenerado; a nota no
  fluxo não muda, e o card de revisão vem num passo próprio.
- **O desktop deixa de ler `done.signed`.** O núcleo apagou o evento e o
  `SignDoneRequest` (passo 1 de `docs/proposals/loop.md`, decisão Q1 C): a
  assinatura é o `done.toml` revisado, e nada nunca emitiu nenhum dos dois. O
  `generated.ts` regenerado perde os dois, e o decodificador e a nota
  "Definição de pronto assinada" saem junto; o `done.proposed` fica.
- **Um desenho para rodar o `/loop` pelo desktop, aprovado pelo dono em
  2026-10-13 com todas as recomendações; é construído antes do orquestrador.**
  `docs/proposals/loop.md` desenha começar, acompanhar, revisar e parar um
  loop pela grade e pela Crew, o pedido ao núcleo em que ele se apoia (N4: o
  daemon conduz a sequência do `/loop`, para os dois clientes verem o mesmo
  loop, e um evento por conferência dentro do turno), e o que fazer com
  `done.proposed`, `done.signed` e `SignDoneRequest`, que nada emite. Escrita
  antes de qualquer código porque a metade do núcleo muda um protocolo
  versionado e o dono pediu para aprovar o desenho antes; nada muda no app ainda.
- **Rodando a partir do código no macOS, a barra de menus e o Dock dizem DCode
  Box.** O `npm start` agora roda antes o `scripts/dev-name.mjs`, que grava o
  nome no Info.plist do Electron de desenvolvimento (em node_modules) e assina
  esse pacote de novo, ad hoc, porque o macOS lê esses dois lugares do pacote e
  não do `app.setName`. Não faz nada quando o nome já está lá, e um `npm ci` que
  o desfaça é corrigido no próximo início. Uma falha é dita e nunca impede o
  app de abrir. O app empacotado não muda.
- **O app se chama DCode Box e mostra o ícone do mascote quando roda a partir
  do código.** Com `npm start`, o binário é o do próprio Electron, então o Dock
  e a barra de tarefas mostravam o ícone do Electron. O ícone agora é definido
  em tempo de execução, e o nome do produto (janelas, diálogos, pacote) é
  "DCode Box". No macOS, o nome do app na barra de menus e o rótulo no Dock
  continuam "Electron" nesse modo: vêm do pacote do próprio Electron (a entrada
  acima também os troca).
- **A cor do dcode é o Laranja do mascote.** O dono escolheu o laranja
  `#FF6A00` como cor principal do dcode, no lugar do lima `#c6f36b` de D29; os
  neutros e as semânticas saem dos 23 tons da folha do mascote, na Grade e na
  Crew, escuro e claro. Onde um tom não se lê como texto, entra um degrau do
  mesmo tom, com as razões WCAG em D52: o acento no claro é `#B33D00` (o
  laranja dá 2,9:1 no branco), o laranja vivo fica como preenchimento sob
  tinta `#0A0A0A` (6,9:1), a espera passa ao amarelo da folha `#FDE68A`
  (escuro) e `#7A5C00` (claro) para não se confundir com rodando, e vermelho e
  verde escurecem ou clareiam para ficar acima de 4,5:1. A seleção e as cores
  ANSI do terminal e o chão do primeiro quadro acompanham. As referências 02 e
  03 foram recapturadas pela própria check:visual, como o dono autorizou para
  esta mudança, e as duas medem 0,00% (D52).
- **O mascote entra no app: o ícone e o estado de cada conversa.** O dono
  escolheu o mascote de `refs/design/brand/mascote.webp` como a cara do dcode.
  Ele é desenhado à mão em vetor, nunca recortado da folha: o ícone do app
  (`assets/icon/icon.svg`, de frente, em Laranja, num ladrilho grafite) e o
  `npm run gen:icons`, que o rasteriza no Chromium do Playwright e monta o
  `.icns`, o `.ico` e o `.png` sem ferramenta de plataforma; o
  `forge.config.ts` o entrega ao empacotador. Onde a janela mostrava o estado
  de uma conversa com um caractere — janela da grade, a maximizada, a thread e
  as linhas de sessão do Crew —, agora mostra a cabeça do mascote no mesmo tom,
  e o olho e uma marca dizem o estado de novo em forma: Planejando…, Codando…,
  Verificando…, Deu certo!, Bug encontrado…, Esperando você, Hum…, Parado. A
  pose sai do que a janela já dobra (o estado da conversa, a fase da chamada
  que roda, o último selo), nunca de um campo que o protocolo não tem. Telas
  vazias mostram o mascote inteiro; avisos levam a cabeça de "Bug
  encontrado…". Menos movimento continua parando a respiração. As referências
  02 e 03 mudam 0,07% e 0,04%, dentro do limite; a troca de paleta empilhada
  sobre esta as recaptura uma vez só (D49).
- **A memória e as skills de um projeto se alcançam da grade.** Só a Crew as
  mostrava. "Memória e skills…" no menu ⋯ de uma conversa agora abre, por cima
  da janela, as próprias seções Memória e Skills da Crew, uma aba cada, no
  projeto daquela conversa (o seletor de projeto continua trocando). Os mesmos
  componentes, então o que a Crew diz — o que não veio, um daemon antigo — a
  grade diz igual. Oferecido no meio de um turno, porque só lê; Esc, um clique
  fora ou "Fechar" fecham. Nada em repouso muda: as duas referências visuais
  medem 0,00%, e é por isso que fica no menu ⋯ e não numa entrada sempre à
  vista (D53).
- **Uma sessão pode ser encerrada pelo menu ⋯.** A janela só fechava uma sessão
  por conta própria (a nunca usada, e a deixada para trás ao trocar de modelo),
  então a pessoa não tinha como encerrar uma. "Encerrar sessão" agora a fecha
  com o mesmo `DELETE /v1/sessions/{id}`; o daemon guarda o registro, a lista
  mostra a conversa terminada, e escrever nela a continua numa sessão nova. No
  meio de um turno, ou com uma aprovação esperando, o menu pergunta antes,
  dizendo o que se perderia, com o foco em "Cancelar". Uma recusa vira aviso com
  o motivo do daemon. Nada em repouso muda: as duas referências visuais medem
  0,00% (D51).
- **Os comandos que a pessoa escreveu aparecem e vão como na TUI.** A TUI
  oferece os comandos de barra de `commands/*.md`, os do usuário e os do
  projeto; a janela não conseguia lê-los. Agora ela pergunta ao núcleo
  (`GET /v1/commands`) uma vez por projeto de que tem conversa, como a TUI lê
  uma vez ao abrir, e o menu de `/` os lista depois dos do campo, com a
  descrição e de quem são. Digitado, um comando é expandido como o
  `config.Expand` da TUI expande (`$ARGUMENTS`, ou o resto acrescentado) e vai
  como mensagem; um arquivo com nome de comando do campo nunca o encobre. A
  ajuda os lista em "Deste projeto". Uma lista que não veio é dita, com o motivo
  do daemon, no pé do menu, na ajuda e num `/nome` desconhecido; nunca uma lista
  vazia. Nada em repouso muda: as duas referências visuais medem 0,00% (D50).
- **O núcleo lista os comandos do usuário num workspace.** `GET /v1/commands`
  responde os comandos de barra que o usuário e o projeto escreveram, como a TUI
  os acha, cada um com o texto que manda. É o que o menu de `/` do campo vai
  oferecer ao lado dos comandos dele, e mandar como a TUI manda. Os tipos do
  protocolo foram regenerados.
- **Os comandos do campo aparecem, e uma ajuda os lista com as teclas.** O
  campo atendia `/undo`, `/compact`, `/mode` e `!`, e nada dizia isso. Um `/` no
  começo do campo de uma conversa agora abre um menu com todos os comandos,
  filtrado enquanto o nome é digitado (↑ ↓ escolhem, ⇥ completa, ↵ envia o que
  não pede mais nada, esc fecha), como a completação da TUI. `/help`, ou `?` no
  campo vazio, abre por cima da janela uma ajuda com os comandos e as teclas da
  janela, do campo e da aprovação. Dois comandos da TUI chegam: `/init` manda o
  turno do DCODE.md da própria TUI, palavra por palavra (um teste compara com
  `internal/tui/commands.go`), e `/plan` mostra o plano atual ou, com texto,
  pede um novo como a TUI pede. Os dois começam um turno, então no meio de um a
  janela recusa e diz por quê, em vez de redirecionar o trabalho com outro
  pedido. Nada em repouso muda: as duas referências visuais medem 0,00% (D48).
- **Imagens vão com o turno.** O daemon aceita imagens num turno desde
  `SubmitTurnRequest.images`, e a TUI as manda; a janela mandava só o texto, e o
  "+" dela dizia que anexar ainda não existia. Agora o "+" abre o seletor do
  sistema (só imagens), colar com uma imagem na área de transferência a anexa,
  e arquivos soltos sobre o campo são anexados. O processo principal lê o
  arquivo escolhido e confere do mesmo jeito os bytes colados ou soltos — o tipo
  pelos bytes (PNG, JPEG, GIF, WebP), até 10 MiB cada e 20 por turno — e confere
  de novo quando o turno é enviado; o renderer nunca vê um caminho. As
  miniaturas ficam acima do campo, cada uma com seu ×, guardadas por conversa
  como o rascunho. Uma recusa vira um aviso e uma linha abaixo das miniaturas,
  com o motivo. Uma correção leva só texto, então uma mensagem com imagens
  durante um turno fica guardada, com o porquê, e nunca vira correção nem entra
  na fila. A mensagem no fluxo mostra as imagens que foram com ela; os eventos
  não as carregam, então uma conversa lida do registro mostra só o texto. Nada
  em repouso muda: as duas referências visuais medem 0,00% (D46).
- **O campo lembra o que se pediu e guarda o que vem depois.** A TUI traz de
  volta o que foi pedido com ↑/↓ e põe uma mensagem na fila até o turno acabar;
  a janela não fazia nenhum dos dois, então pedir de novo era redigitar, e toda
  mensagem durante um turno virava redirecionamento. Agora ↑/↓ no campo de uma
  conversa percorrem o que ela já pediu, da mais nova para a mais velha, e
  devolvem o rascunho depois da mais nova — só nas bordas do campo, então uma
  mensagem de várias linhas continua editada como texto. O que se pediu vem dos
  próprios eventos `turn.started` e `turn.steered` da conversa, e sobrevive a
  reiniciar o app sem um segundo armazenamento; uma conversa continuada lê o
  registro da de origem na primeira vez que ↑ passa da mais velha. Durante o
  turno ↵ continua redirecionando (D23) e ⌥↵ põe na fila: a fila aparece em
  fichas acima do campo, cada uma com ×, ⌃X tira a última, e cada uma é enviada
  quando a conversa volta a ficar parada, uma por turno. Um envio recusado para
  a fila, diz por quê e espera "Retomar"; só mensagens entram, e um comando é
  recusado com o motivo. As duas referências visuais seguem medindo 0,00% (D47).
- **Uma chamada de ferramenta abre no que ela fez.** O fluxo mostrava cada
  chamada do modelo numa linha de resumo, embora o `tool.completed` sempre
  trouxesse a saída inteira e, de quem mudou arquivo, o diff unificado; só um
  comando digitado com `!` aparecia aberto. Agora um clique na linha, ou foco e
  ↵/espaço, abre a chamada: uma mudança mostra o diff com + e − nas cores do
  painel do git (o mesmo desenho), qualquer outra a saída em monoespaçado sem
  os códigos ANSI, e uma falha o erro inteiro. Saída de mais de 40 linhas abre
  nas 40 primeiras, com "mostrar tudo"; o corte da ferramenta e o corte de 400
  linhas do diff no daemon são ditos, nunca calados. Vale para a sessão viva e
  para o histórico gravado. Fechada por padrão e desenhada como antes: as duas
  referências visuais medem 0,00% (D45).
- **A janela diz onde um turno está contra os tetos dele, e quanto o último
  custou.** As rodadas e as chamadas em voo chegavam e eram dobradas, mas nada
  as mostrava, e do `usage` do `turn.completed` só o contexto ficava. Agora a
  barra do composer — o rodapé da janela da grade e o da maximizada — mostra
  "rodada N/M" a partir da metade do teto (aviso, e erro a partir de três
  quartos) e "N/M em voo" quando todas as vagas estão ocupadas, como a barra da
  TUI; longe dos tetos não desenha nada, e a contagem inteira fica no
  passar do mouse do botão de parar. A aba Sessão do Crew mostra o turno
  corrente e, do último que terminou, saída, entrada somada pelas rodadas e
  cache. Os tokens só viajam no fim do turno, então nada é estimado enquanto
  ele corre, e um turno sem `usage` diz que o provedor não informou. Nada em
  repouso muda: as duas referências visuais medem 0,00% (D44).
- **Uma conversa pode ser renomeada pela janela.** O daemon nomeia conversas,
  vivas e terminadas, desde `POST /v1/sessions/{id}/name`, e a janela nunca
  pedia: o título era o que a primeira pergunta fosse. Agora um duplo clique no
  título (janela da grade, a maximizada, a thread do Crew) vira um campo, e
  "Renomear" no menu ⋯ abre o mesmo campo. Enter ou sair do campo salva, Esc
  desiste, um nome vazio devolve o título automático, e as teclas da janela
  ficam fora do campo. O título muda quando o daemon diz, não quando o nome é
  digitado: o `session.renamed` de uma conversa viva ou, para uma terminada
  (cujo nome nenhum stream anuncia), a linha dela relida de
  `GET /v1/conversations`. Uma recusa vira um aviso com o motivo do daemon, e o
  campo guarda o texto. O nome passa por um canal novo do processo principal:
  id no alfabeto do núcleo, nome aparado e recusado, não cortado, acima dos 120
  caracteres do núcleo. Nada em repouso muda: as duas referências visuais medem
  0,00% (D43).
- **Uma conversa terminada abre no histórico dela.** Abrir uma mostrava a
  thread vazia, e o painel da direita do Crew dizia que uma conversa terminada
  não traz os eventos dela: a pessoa achou que as sessões tinham se perdido.
  Uma conversa gravada sem sessão viva, numa janela da grade, maximizada ou na
  thread do Crew, agora é lida página por página da rota
  `GET /v1/conversations/{id}/events` do núcleo, por um canal novo do processo
  principal (o id conferido no alfabeto do núcleo, a posição a partir de 1).
  Os eventos são dobrados como os de uma sessão viva e desenhados só para
  leitura pelo mesmo fluxo, acima do mesmo campo "Escrever continua a conversa
  numa sessão nova". As Mudanças do Crew listam os arquivos que ela mudou.
  Enquanto lê, a janela diz que está lendo. Se a leitura falha, o motivo do
  daemon aparece na thread com "Tentar de novo" e também num aviso; nunca
  parece uma conversa vazia. Passando de 20 páginas, a janela diz quanto ficou
  sem ler. As duas referências visuais continuam a 0,00% (D42).
- **Fica claro onde a conversa trabalha.** A conversa em foco mostrava o título
  e o nome curto da pasta, a branch do início da sessão e só o que as
  ferramentas disseram ter escrito. Agora, na Crew, debaixo do título: o id da
  sessão, a pasta, a branch de agora, se a pasta é uma worktree e de qual
  checkout, e +/− da pasta. Mudanças lista cada arquivo mudado desde o último
  commit com as linhas, e abrir um mostra o diff; Sessão traz id e pasta com
  copiar, viva ou terminada, em que branch a sessão começou se a pasta mudou, e o
  commit. Na Grade, o chip da branch e o rodapé de cada janela abrem um cartão
  com o mesmo, sem mudar o que a grade desenha (02 e 03 a 0,00%). Lido do
  daemon, pela conversa, nunca por um caminho que a janela escolhe; pasta sem
  git, git ausente e daemon antigo são ditos, nunca uma lista vazia (D41).
- **Os tipos do protocolo trazem a página de uma conversa gravada.**
  `src/protocol/generated.ts` foi regenerado para a nova rota do núcleo,
  `GET /v1/conversations/{id}/events` (`ConversationEventsResponse`,
  `ConversationEventsLimit`). Nada na janela a lê ainda; isso é a próxima
  mudança, que abre uma conversa terminada no histórico dela em vez de numa
  thread vazia.
- **O núcleo lê o git de um workspace como está agora.** Duas rotas respondem,
  para uma pasta, a branch de agora, o commit, se é uma worktree ligada e de
  qual checkout, cada arquivo mudado desde o último commit com as linhas, e o
  diff de um arquivo. O que o painel de contexto vai ler, em vez da branch
  congelada no início da sessão e do que os eventos das ferramentas disseram ter
  escrito. Os tipos do protocolo foram regenerados.
- **A aprovação oferece as respostas que a TUI oferece.** O cartão oferecia
  sempre uma vez, nesta sessão e negar; o protocolo tem cinco respostas, e a TUI
  escolhe pela travessia. Uma travessia de rede agora oferece uma vez, neste
  projeto, sempre e negar (sem "nesta sessão", como na TUI); qualquer outra, uma
  vez, nesta sessão e negar. Teclas de 1 a n, ↵ e esc negam, negar continua
  destacada; as respostas que ficam gravadas além da sessão vêm no tom de aviso.
  A razão do daemon fica no cartão, não só no título. A tela de referência 03 é
  uma travessia de rede e mudou com isto (0,73%); o dono a viu e autorizou
  recapturar só a 03 no mesmo PR, e as duas medem 0,00% (D39).
- **Desfazer, resumir e rodar um comando pela janela.** `/undo`, `/compact` e
  `!` eram só da TUI, e um texto começando com `/` ia ao modelo como mensagem. O
  ⋯ da conversa (janela da grade, conversa na Crew) agora oferece Desfazer o
  último turno — dizendo o que voltou e o que ficou porque mudou depois do turno
  —, Resumir o contexto — dizendo quando não havia o que resumir, o que nenhum
  evento diz — e Rodar um comando…; `!comando` no campo roda na sessão, e a
  saída aparece aberta. `/undo`, `/compact` e `/mode <m>` funcionam no campo;
  qualquer outra barra é "comando desconhecido" e fica lá (D40).
- **Um comando que o agente propôs pode ser rodado pela pessoa, num terminal
  abaixo da conversa, e o resultado volta ao agente.** Pedido depois de uma
  aprovação negada que a pessoa queria rodar ela mesma, talvez com sudo. Todo
  comando proposto — o do cartão de aprovação e cada linha `bash` do fluxo —
  ganha Copiar e Rodar no terminal, que aparecem ao passar o ponteiro ou com o
  foco e ficam fora do layout, e as duas referências visuais seguem a 0,00%.
  Rodar no terminal abre um shell de verdade (um PTY no processo principal,
  `node-pty`, desenhado pelo xterm.js) abaixo do fluxo — na janela da grade, na
  conversa maximizada e na conversa da Crew —, na pasta da sessão como o daemon
  a lista, com o comando digitado e não rodado: a pessoa edita e aperta Enter,
  e o sudo pede a senha dentro do terminal. zsh e bash marcam o prompt (OSC
  133), e quando o comando termina a janela oferece o resultado ao agente,
  mostrando antes a mensagem — comando, código de saída, as últimas 200 linhas
  sem ANSI, cortadas e dito que foram, sem as linhas de prompt de senha — e só
  manda com um clique, como turno ou correção. Canais nomeados conferidos no
  processo principal; cada terminal responde só à janela que o abriu; nada do
  que passa por ele vai a log. Dependências, em versões exatas: `node-pty`
  1.2.0-beta.15 (a 1.1.0 publica o spawn-helper do macOS sem permissão de
  execução), `@xterm/xterm` 6.0.0, `@xterm/addon-fit` 0.11.0, carregados só
  quando um terminal abre (D37).
- **O modelo e o modo de uma sessão nova são escolhidos antes de ela abrir.** A
  Nova sessão abria sempre no padrão do daemon, e trocar depois custava uma
  sessão nova ou um turno no modo errado. A janela que pergunta onde, na grade, e
  o "Nova ▾" da Crew ganham "+ modelo e modo": ligado, escolher o projeto (ou
  outra pasta) leva aos modelos que o daemon lista para ele e aos quatro modos,
  e abre ali. O que fica no padrão não é enviado. O modo vai como o sandbox que
  ele mapeia, e a janela troca o modo logo depois quando a sessão abriu em
  outro; uma recusa deixa a sessão aberta e diz por quê (D38).
- **O modo da sessão muda pela janela.** Plano, assistir e auto eram só da TUI
  (`/mode`, ⇧⇥); a janela mostrava o modo e não o mudava. O chip do modo — no
  campo da janela da grade, no da conversa maximizada e na aba Sessão da Crew —
  abre um menu com os três, e o ⇧⇥ passa de um ao outro na conversa em foco. O
  chip muda quando a sessão anuncia a troca, nunca antes, e uma recusa nunca
  aparece como aceita: vem com a razão do daemon. Auto vem no tom de aviso, como
  na barra da TUI, e um par que não é nenhum dos três é chamado de personalizado,
  não do modo mais próximo (D36).
- **Um tema claro, sóbrio, escolhido por janela; o escuro continua o padrão.**
  O grafite era só escuro (D29), o que deixa de fora a luz do dia e quem lê
  melhor no claro. Sistema (segue o sistema), Claro ou Escuro, nas
  Configurações da Crew e, na grade, no ⚙ do pé da lateral — que antes dizia
  "ainda não" e não custa nenhum pixel que as réguas medem. O claro é um bloco
  de tokens para cada versão da tela: chão neutro quase branco, texto quase
  preto, o cinza mais apagado acima de 4,5:1 em todo lugar, um acento
  oliva-lima para texto e marcas (o lima vivo fica só no preenchimento do botão
  de enviar, sob tinta escura), e âmbar, verde e vermelho escurecidos para serem
  lidos como texto. As cores que ainda estavam escritas nos estilos viraram
  tokens. A janela abre escondida até o primeiro quadro pintado, e nenhuma
  paleta pisca a outra. Sem escolha, é Escuro, e as duas referências visuais
  continuam a 0,00% (D35).
- **O texto do modelo é desenhado como o markdown que ele é.** Respostas com
  títulos em negrito e listas mostravam os asteriscos e os traços crus. O texto
  do modelo agora desenha parágrafos, negrito, itálico, código em linha, blocos
  de código cercados (mono, com rolagem horizontal), títulos modestos, listas
  com e sem número e aninhadas, citações, linhas, tabelas e links. O `marked`
  18.0.14 (MIT, sem dependências) só separa os tokens, e a árvore vira elementos
  React: nenhuma string de HTML é montada, então o HTML que o modelo escreve
  aparece como texto e a CSP não muda. Só link `https` é link, aberto no
  navegador pelo processo principal; qualquer outro endereço fica como texto, com
  o endereço no título, e imagem é o texto alternativo. Durante o streaming, um
  `**` sem par fica como chegou e uma cerca sem fim é código até onde o texto
  chegou. Os estilos usam só os tokens que existem, e valem na grade, nas janelas
  compactas e na Crew (D34).
- **As referências visuais são capturas da janela aprovada.** O dono aprovou o
  visual novo — grafite, as janelas da grade, a Crew — e o `check:visual` o media
  contra o mock do handoff, com o qual ele já não se parece. Os dois estados
  medidos (02, 03) agora são capturados da janela pela própria verificação, nas
  condições dela; o piso é 0 px nos dois, e o limite continua 1,25%. Uma mudança
  visual futura os troca num PR próprio, depois do visto. As decisões de D21 em
  diante vão para `docs/DECISIONS-2.md`, e cada arquivo fica abaixo de 500 linhas
  (D18).
- **Os tipos de memória e skills da Crew vêm do núcleo.** Escritos à mão
  enquanto o #432 estava aberto, agora são os que o tygo gera a partir de
  `internal/protocol`; `src/protocol/crew.ts` fica só com o decodificador que
  confere uma resposta contra eles, e uma mudança de forma no núcleo reprova o
  `check:protocol` em vez de divergir (N6).
- **A Crew vira um painel com um trilho de seções, sobre os dados do daemon.** A
  primeira Crew pôs um terceiro painel ao lado da lateral da grade; estudada
  contra um painel desse tipo, faltava a forma que faz um: seções, e uma lista de
  sessões própria. A Crew agora tem um trilho estreito de ícones — Painel, Agenda,
  Memória, Skills, Apps, Conhecimento, Configurações — e o Painel tem três
  painéis: toda conversa que o daemon lista, com busca e grupos (ativas, por
  projeto, mais antigas), a em foco inteira, e o contexto dela em abas —
  Mudanças, Pronto, Sessão. Skills e Memória leem as rotas novas do daemon (N6,
  #432): cada skill com a origem, o arquivo, os gatilhos e o que a faz pedir
  permissão, e os arquivos recusados; cada memória por tipo, marcando a velha e a
  que a sessão não lê, com os blocos que não são memória. Um daemon anterior às
  rotas aparece como "ainda não expõe", com a resposta dele. Configurações mostra
  a versão da tela, o daemon e os modelos de cada projeto. Rotinas, apps e
  conhecimento ainda não têm nada no núcleo e dizem isso em vez de mostrar uma
  lista vazia. A grade fica intocada e continua sendo o padrão. Ícones do `lucide-react` (MIT)
  (D33).
- **Uma segunda versão da tela, Crew, ao lado da grade.** Depois do Kiro Crew, e
  sem desmanchar a grade: um seletor na barra do topo alterna Grade e Crew, e a
  grade continua sendo o padrão. A Crew tem três painéis — a lista, a conversa em
  foco inteira e, à direita, o selo e os critérios, os arquivos que mudou, onde
  roda e o resto da tripulação —, em alto contraste (o cinza mais apagado em cerca
  de 6:1), com Geist e Fragment Mono; a Diatype do Kiro é licenciada e não pode vir
  no app. As fontes agora são sempre arquivos próprios no build, para a CSP nunca
  recusar uma em silêncio (D32).
- **Cada janela da grade é uma conversa em que se trabalha.** Pedido depois de ver
  a grade: cada quadrante uma janela isolada, como uma conversa própria. A janela
  agora traz a conversa inteira — selo e critérios, o fluxo com rolagem própria, a
  aprovação e o seu próprio campo —, e se trabalha em várias ao mesmo tempo sem
  abrir nenhuma. Aberta, clicada ou escrita, uma conversa fica até a janela dela
  ser fechada; o que roda ou espera mantém o lugar. Nova sessão pergunta onde numa
  janela própria, com os projetos que a lista já conhece, e vira a sessão ali; sem
  nenhum conhecido, o seletor de pastas vem direto. Escrever numa conversa
  terminada a continua na mesma janela. ⌘1–9 põe o cursor na janela n, ⌘↵
  maximiza e restaura, Esc sai do campo (D31).
- **A janela abre numa grade de atenção, em grafite.** Uma conversa por vez lia
  como clone de app de chat, e quem roda vários agentes quer ver todos. Agora a
  janela abre na grade das conversas que pedem lugar — esperando você, fixadas,
  rodando, ou terminadas e ainda não vistas —, em lugares que não pulam enquanto o
  trabalho anda, até 6 painéis, ou 9 numa janela grande. Cada painel traz o selo
  do último turno e cada critério de pronto, o fim do trabalho e a aprovação
  respondida ali mesmo; ↵ abre a conversa, com os outros painéis como pares no
  topo, e Esc ou ⌘0 voltam. Grafite é escuro, com um acento lima, a fonte do
  sistema e mono no que é código, e cantos de 4px; os selos e as palavras deles
  estão em português. As referências visuais do handoff deixam de descrever a
  janela, então o `check:visual` reprova contra elas até referências novas serem
  aprovadas (D18, D29, D30).
- **Um turno que escreveu onde o trabalho é medido diz isso.** O daemon já
  mandava os caminhos protegidos que um turno escreveu (`touched_protected`), e a
  janela nunca os lia. Agora o selo diz "· tocou a régua: …" e deixa de ser verde,
  e um painel diz "⚠ tocou a régua" ao lado dos critérios, com os caminhos no
  título: um "verificado" medido com a régua que o próprio turno mudou não é um
  verificado comum (D13, D30).
- **Um painel diz quando a família do modelo nunca foi medida.** O selo diz o que
  os critérios de pronto conferiram, não se alguém mediu o modelo que fez o
  trabalho. Ao lado dos critérios, um painel e a conversa aberta dizem "sem
  medição" quando a lista de modelos do daemon diz que a família não tem medição
  por trás; lista que não veio não marca nada (D30).
- **O núcleo lista a memória e as skills de um workspace.** Duas rotas
  respondem, para um projeto, o que as sessões dele leem como memória — cada
  entrada, se está velha e se a sessão a mostra, e os blocos que não são
  memória, com o motivo — e quais skills elas têm, as do usuário e as do
  projeto, com a origem de cada uma e se espera por uma pessoa. O que a visão de
  equipe vai ler, em vez de interpretar o `.dcode/memory.md` e as pastas de
  skills com uma cópia das regras do núcleo. Os tipos do protocolo foram
  regenerados.
- **Uma conversa continua em outro modelo.** O botão do modelo, no composer, abre
  o que o daemon diz que uma sessão daquele projeto pode pedir (`GET
  /v1/models`): o padrão do projeto e cada perfil do `models.toml`, cada um
  marcado como medido — os contratos de comportamento do dcode já rodaram contra
  a família — ou não, com o aviso do próprio daemon. Escolher um continua a
  conversa numa sessão nova nele, com todo o histórico, e encerra a que ficou para
  trás, cujo registro permanece; aberta, cada troca deixaria mais uma, até o
  daemon recusar novas. Com um turno rodando ou esperando resposta, o menu diz
  para parar ou esperar. Daemon sem a rota diz isso no menu, e nenhum modelo é
  adivinhado no lugar (D28).
- **Uma conversa que termina não é mais dada como falha.** Quando uma sessão
  fecha, o daemon encerra o fluxo dela e a lista a mostra terminada; a janela
  tomava o fim do fluxo por defeito e dizia que os eventos tinham parado de chegar.
  Agora espera um instante pela lista, que chega por uma conexão própria, e só
  avisa quando a conversa continua viva nela.
- **O núcleo lista os modelos que uma sessão pode pedir.** Uma rota responde, para
  o workspace de uma sessão, o modelo que ela recebe por padrão e cada perfil para
  o qual pode trocar, cada um com a família e se essa família tem medição por trás
  — o que o menu de modelo por sessão vai ler, em vez de ler o `models.toml` e
  guardar uma cópia da lista de famílias medidas do núcleo. Os tipos do protocolo
  foram regenerados.
- **Um turno que dá outra volta fala numa mensagem nova.** Quando a checagem de
  pronto manda um turno de volta ao trabalho, as palavras seguintes do modelo
  saíam coladas às últimas — "…o parser.O teste…" —, porque o texto que chega em
  partes se juntava a qualquer mensagem aberta do mesmo turno. O daemon avisa cada
  rodada nova com um evento de progresso; agora o texto só se junta à mensagem do
  mesmo turno e da mesma rodada.
- **O ⌘K acha qualquer conversa.** A busca que o design desenha, sobre a lista
  do daemon: cada palavra digitada precisa estar no título ou no projeto, sem
  distinguir maiúsculas nem acentos; as conversas rodando ou esperando você
  primeiro, depois um grupo por projeto; `tab` troca o escopo, as setas movem o
  cursor, `esc` limpa e depois fecha. O `↵` abre a do cursor como um clique na
  linha dela abriria, então uma terminada é continuada. A prévia mostra o que a
  lista traz — o estado, o selo, o workspace, a branch, o modelo, os turnos e o
  último evento —, e o `docs/DECISIONS.md` registra o que o design pede e a lista
  não traz (D27, L17).
- **A janela roda sobre um daemon de verdade.** Dentro do Electron, o processo
  principal anexa ao `dcode serve` que responde no socket — `DCODE_SOCKET`, senão
  o que o `dcode socket` imprimir — ou sobe um ali como filho, de `DCODE_BIN`,
  `~/.local/bin/dcode` ou do `PATH`; encerra só o que subiu, e pergunta antes
  quando fechar pararia uma sessão rodando ou esperando você (D19). A barra
  inferior diz conectando, conectado com a versão do daemon, caiu ou não subiu,
  com o porquê. A lateral é a lista de conversas do daemon (N3), vivas e
  terminadas, por projeto: uma viva abre o fluxo dela desde o primeiro evento e o
  retoma do último quando ele cai; uma terminada é continuada numa sessão nova,
  que a janela abre (D21). O campo envia um turno, corrige um que roda (D23) e o
  para; uma aprovação se responde com `1`, `2`, `3`, `esc` e `↵`, que nega (D22);
  Nova sessão e ⌘N pedem uma pasta e abrem uma sessão nela. Toda recusa mostra a
  mensagem do daemon, e nada enviado a um daemon que morreu parece aceito. O
  renderer continua sem ver Node nem o socket: o preload lhe entrega um canal com
  nome por pedido e por notícia (`src/shared/api.ts`). Um navegador, ou o
  `?fixture=`, continua mostrando a gravação, e a régua visual lê 1,20% e 1,21% —
  os números em que D22 e D23 foram medidas.
- **O app guarda os dados numa pasta só dele.** O padrão do Electron é o nome do
  produto na pasta de dados do sistema, `~/Library/Application Support/DCode` no
  macOS — e o sistema de arquivos ali não distingue `DCode` de `dcode`, a pasta em
  que o dcode guarda configuração, registros e estado. Cada `npm start` escrevia o
  armazenamento, os caches e os cookies do Chromium ao lado do `models.toml`, e
  apagar os dados do app levaria a configuração do dcode junto. O app agora usa
  `dcode-desktop` ali, e `DCODE_DESKTOP_USER_DATA` escolhe outra pasta, absoluta —
  que a régua já passava e o app ignorava. O que as execuções anteriores deixaram
  na pasta do dcode é do Chromium e pode ser apagado à mão; o que é do dcode ali é
  `doctrine`, `models.toml`, `sessions` e `update-check.json`. `docs/DECISIONS.md`
  D26.
- **O loop agora leva a janela até ficar usável.** Com N1–N3 no núcleo, o
  `docs/loop/tasks.md` acrescenta o que o trabalho do dia a dia pede além da
  conexão: a lateral listando todas as conversas, vivas e terminadas, a partir da
  lista do núcleo e do fluxo único dela; abrir uma terminada continuando-a numa
  sessão nova; e o ⌘K achando qualquer conversa pelo título. A régua ganha um
  cenário para cada — `lists-recorded`, `continues-recorded`, `searches` —, e o
  `check:daemon` passa a ter onze. O lado do daemon deles foi provado contra um
  `dcode serve` de verdade antes do commit; reprovam de propósito, na janela, até
  ela conectar.
- **O núcleo lista todas as conversas para a lateral (N3).** Uma rota com as
  conversas vivas e as gravadas, cada uma com título, projeto, estado, branch,
  modelo, turnos, selo, diff e última atividade, e um fluxo só do que muda — o que
  a lateral e a busca vão ler em vez de um fluxo por sessão. Os tipos do protocolo
  foram regenerados, e o `docs/DECISIONS.md` marca o N3 como feito.
- **Uma sessão aberta pela janela lê a configuração do seu projeto (N2).** O
  daemon agora resolve a cadeia de configuração no workspace de cada sessão, então
  uma sessão aberta num projeto roda sob o `.dcode/config.toml` daquele projeto,
  mesmo com o daemon subindo fora de todos eles, como o desktop o sobe.
  Configuração de projeto que não se lê recusa a sessão com o motivo. O
  `docs/DECISIONS.md` marca o N2 como feito.
- **A verificação visual fica fora do CI pelo motivo que vale.** Os documentos a
  deixavam local porque as imagens de referência não estavam no repositório;
  estão desde que o handoff do design entrou. Rodada uma vez no `ubuntu-latest`
  do workflow, ela mede 1,76% e 1,79% contra o limite de 1,25%, igual em quatro
  runners: o Chromium do Linux aplica hinting nos glifos e os suaviza em
  subpixels LCD, e nem o macOS — onde o limite foi medido — nem as referências
  fazem isso. O limite fica: afrouxado para caber o Linux, deixaria ao macOS
  mais de 9 mil pixels de folga, espaço para passar um elemento faltando. O
  `AGENTS.md`, o workflow, a nota da régua e o changelog da raiz param de dizer
  que as referências faltam, e o `AGENTS.md` dá o motivo que vale, com as flags
  de inicialização que levaram o Linux a 1,19% e 1,20% — uma mudança de régua,
  deixada para uma decisão própria.
- **O núcleo agora diz onde está o daemon (N1).** `dcode socket` imprime o
  caminho do socket do daemon — um por usuário, qualquer que seja o ambiente —
  ou falha dizendo por quê, e o loop da conexão pergunta a ele em vez de guardar
  uma cópia da regra do núcleo. O `docs/DECISIONS.md` e o `docs/loop/tasks.md`
  nomeiam o comando.
- **O próximo loop liga a janela a um daemon de verdade, e a régua dele foi escrita
  antes do código.** O `npm run check:daemon` compila o app e o `dcode` do
  checkout e roda oito cenários, cada um contra um `dcode serve` novo com um
  modelo roteirizado: anexar, subir o próprio, dizer por que não subiu, uma
  sessão nova com conversa, `↵` negando uma aprovação, `1` permitindo, redirecionar
  um turno em andamento, e um daemon que morre. Cada um é conferido contra o que o
  daemon registrou, não só contra o que a janela desenhou. O `docs/loop/` agora
  aponta o `/loop` para ela, com a régua protegida. Reprova de propósito: a janela
  ainda não conecta, e cada cenário diz onde para. Escrita antes porque um loop
  que escreve a própria régua mede o que construiu, não o que foi decidido.
- **Sete decisões para falar com o daemon, no `docs/DECISIONS.md` (D19–D25).**
  - O desktop sobe o daemon quando nenhum responde, e esse daemon para quando o
    app fecha.
  - Um daemon atende todos os projetos, com a configuração resolvida por sessão.
  - A lista de sessões vem do protocolo.
  - `↵` nega a aprovação e a mensagem digitada durante o turno o redireciona, como
    na TUI.
  - O loop é uma tela sobre o mecanismo que existe.
  - Tokens ao vivo ficam para depois.

  Tomadas antes de ligar qualquer coisa, porque cada uma muda o que é ligar. O que
  elas pedem ao núcleo está listado como N1–N4, cada um um PR do núcleo com a spec
  antes.
- **A área do desktop existe, à parte do núcleo.** Electron Forge com Vite,
  TypeScript e React, em `desktop/`, com README, changelog, versão e `AGENTS.md`
  próprios, e um `go.mod` de fachada para o `go … ./...` do núcleo parar na porta
  em vez de entrar no `node_modules`. À parte porque o desktop depende do núcleo e
  nunca o contrário, e um cliente que sai no próprio ritmo não deve pegar
  emprestada a versão do núcleo.
- **A janela principal do design v2, sobre eventos de protocolo gravados.** A
  lateral de projetos e sessões com as marcas de estado, uma sessão por vez com o
  seu fluxo — a pergunta, linhas de ferramenta compactas, o texto do modelo, o
  card de delegação, a linha de atividade —, o card de aprovação com as três
  respostas e o relógio ao vivo, o composer e a barra inferior, nos dois temas,
  seguindo o sistema. Gravados, e não ao vivo, porque a janela tinha de estar
  certa antes de ser ligada: a próxima versão conecta o processo principal ao
  socket do daemon. A barra inferior diz `gravação` para uma gravação nunca
  passar por daemon, e toda ação que precisaria de um avisa que nada foi enviado.
- **Um redutor puro de eventos do protocolo para o que a janela mostra**, testado
  com vitest. Lê só campos que o protocolo traz, validados na fronteira: payload
  que não bate é dito no fluxo da sessão, evento repetido não muda nada, evento
  que falta é nomeado. Onde o design pede o que o fio não traz — o progresso de
  um filho, tokens ao vivo, os caminhos pedidos na aprovação, o loop —, desenha o
  que existe e o `docs/DECISIONS.md` lista a lacuna, porque campo inventado é
  promessa que o daemon nunca fez.
- **Tipos do protocolo gerados do `internal/protocol`** com o tygo, commitados, e
  conferidos no CI; todo tipo de evento que o núcleo declara precisa de leitor
  aqui, e toda forma de payload precisa concordar com o tipo gerado, ou os testes
  e o typecheck reprovam. Uma definição só do fio, e um desktop que percebe quando
  ele se move.
- **Ordem, rótulos e recolhimento dos projetos são preferências locais**, guardadas
  no armazenamento do próprio app e nunca mandadas ao daemon: recolher e expandir,
  recolher tudo, renomear no lugar, mover para cima e para baixo, e arrastar para
  reordenar. Armazenamento que não se lê ou não se grava é dito, e a lateral volta
  ao arranjo inicial.
- **O renderer não alcança nada.** Isolamento de contexto, sandbox, nenhum Node,
  um preload que expõe a plataforma e o nome de quem usa e mais nada, uma CSP que
  não o deixa conectar a lugar nenhum, fontes empacotadas, nenhuma porta aberta —
  e o lint reprova um import de Node em código do renderer.
- **O design pode ser conferido num loop.** `npm run check:visual` compara a
  janela em modo fixture com as capturas de referência do design (Chromium do
  Playwright, pixelmatch) e reprova além de um limite cujo raciocínio fica ao
  lado dele: o próprio mock do design, desenhado pelo mesmo Chromium, fica a
  1,15% e 1,19% das referências, e a janela mede 1,17% e 1,19%. O
  `desktop/docs/loop/` deixa o `/loop` do próprio dcode iterar contra ele, com as
  referências e a régua protegidas. Só local: o limite foi medido no macOS e
  vale lá.
