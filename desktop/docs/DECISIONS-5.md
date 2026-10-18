# Decisões do desktop — D62 em diante

A continuação de [`DECISIONS-4.md`](DECISIONS-4.md), que vai de D49 a D61, e
dos anteriores, a partir de [`DECISIONS.md`](DECISIONS.md), que tem a
introdução, D1 a D20, as decisões em aberto, as lacunas e os pedidos ao núcleo.
A numeração é uma só; este arquivo começa porque o DECISIONS-4 passaria de 500
linhas com D62.

**D62. A revisão mostra as propostas anteriores do turno, recolhidas embaixo da
escrita.** Decisão do dono, de 2026-10-16: construir o histórico que a
`done-qualifier` promete (§4) como defesa contra uma reproposta que troca um
critério por um trivial. O daemon guarda toda proposta do turno qualificador,
mede e escreve a última, e o `done.proposed` passa a levar `sequence`,
`proposed_at` e `earlier` (registro `202610161200`).

- **O card continua sendo o da proposta escrita** (D56). Quando o turno
  repropôs, o título ganha "· 3ª proposta do turno", e embaixo, antes dos
  caminhos protegidos da escrita, um `<details>` fechado: "2 propostas
  anteriores · o turno repropôs; só a última foi medida e escrita".
- **Aberto, cada anterior em ordem, da mais antiga:** "#1 · 3 critérios · 4m00s
  antes da última", e cada critério com o nome, o comando, o que o proponente
  disse dele ("disse que falharia" / "disse que passaria") e o porquê; os
  protegidos dela, se havia. Sem classe, sem exit, sem ✓/✗: as anteriores
  nunca rodaram, e desenhar uma medição que não aconteceu seria inventá-la.
- **O tempo é contado da última, não do relógio.** "4m00s antes da última" não
  depende do fuso da máquina e diz o que importa — quanto o turno pensou entre
  uma e outra.
- **Uma proposta só não muda nada:** sem sufixo no título e sem o `<details>`.
  O caso comum fica com a cara de antes.

As réguas: o card só aparece com um loop esperando revisão, que as telas 02 e 03
não têm; as duas ficam como estavam.

Descartados: as anteriores lado a lado com a escrita, aberto por padrão (o card
já é o mais alto da régua, e o olho deve cair primeiro no que vai ser usado);
um diff entre propostas (custa desenho próprio, e a lista em ordem já mostra a
troca); medir as anteriores para dar classe a elas (rodar critérios que ninguém
vai usar, só para enfeitar a lista).

**D63. A Crew é a tela principal e leva todo elemento novo; a grade fica como
alternativa** (emenda D32 e D33). Decisão do dono, de 2026-10-17, depois de ver
as capturas da régua estendida: as telas seguem o visual da Crew — a fonte, as
cores, a paleta e o arranjo de três painéis (Sessões com busca e grupos por
projeto à esquerda, a conversa no meio com o título, a linha de onde trabalha e
o chip do modelo, as abas Mudanças, Pronto e Sessão à direita) —, e cada
elemento feito desde D41 aparece nela.

- **O padrão é a Crew.** Sem escolha guardada, a janela abre na Crew; quem
  escolheu a grade continua nela, e o seletor da barra do topo alterna as duas
  como antes. A ordem do seletor fica "Grade · Crew": trocá-la mudaria a barra
  das telas 02 e 03, que a régua mede.
- **Onde cada elemento mora na Crew:**
  - a régua do loop, com as pastas e os botões de parar (D55), e o card da
    revisão com as propostas anteriores (D56, D62): no alto da conversa, abaixo
    do título, como na maximizada. A aba Pronto mantém a régua com as pastas
    abertas, sem um segundo card: diz que a revisão está no alto da conversa;
  - os critérios em "ciclo N · parado s de L" (D54): na mesma faixa, abaixo da
    régua, e de novo na aba Pronto; o "⟲ Ciclo N voltou" fica no fluxo;
  - o plano fixo como checklist (D57), os chips de atividade com "+N" (D58), os
    anexos no campo e na mensagem (D46, D59) e o card de aprovação com as quatro
    respostas (D39): já estavam no corpo da conversa que a Crew e a maximizada
    compartilham, e seguem com os tokens da Crew;
  - "↻ loop" no menu "Nova": escolher o projeto abre a folha "Começar loop",
    como na janela de Nova sessão da grade; o `/loop` no campo continua;
  - o ⌘K (D27) abre sobre a Crew, como sobre a grade;
  - o PR e o CI (D60, D61) continuam no alto da aba Mudanças.
- **A thread segue o loop.** A Crew tem uma conversa no meio, e um loop troca
  de sessão a cada pasta: a thread vai da sessão que o loop abriu para a que ele
  ocupa agora, e um loop começado nesta janela toma o meio quando a primeira
  sessão dele abre. Abrir a conversa de uma pasta pela régua a põe no meio, em
  vez de maximizá-la numa grade que não está desenhada.
- **O que a régua do daemon lê** fica igual nas duas telas: a linha da conversa
  com `data-session-id` e `data-state`, a janela da conversa com
  `data-pane-id`, o palco como `main`, e "Nova sessão" como nome do botão
  "Nova".
- **Só na grade:** várias conversas lado a lado e a maximizada com os pares no
  topo. A lateral da grade continua a continuar uma gravada ao clique (D21,
  D42); na Crew, clicar numa gravada a mostra, e escrever a continua (D33).

As réguas: as telas 02 e 03 são da grade maximizada e continuam abrindo nela
no modo fixture, a 0,00%. O `check:daemon` passa 13 de 13. O cenário
"continues-recorded" segue agora a Crew, com autorização do dono para mexer em
`tests/daemon/` só nisso: clicava numa conversa gravada e esperava a
continuação, o comportamento da lateral da grade; agora o clique mostra o
histórico, nada continua sem que se escreva, e a mensagem escrita continua a
conversa numa sessão nova, que a janela abre e onde o modelo responde. A
intenção fica — uma conversa terminada se continua pela janela —, porque o dono
manteve na Crew o que D33 decidiu: mostrar antes de continuar.

Descartados: a Crew continuar uma gravada ao clique só para o cenário passar
(trocaria a leitura do histórico por uma sessão que ninguém pediu, o que D42
recusou); trocar a ordem do seletor (moveria a régua da 02 e da 03); um segundo
card de revisão na aba Pronto (dois "Começar o trabalho" para a mesma
proposta); refazer a grade com os tokens da Crew (o pedido é a Crew como tela
principal, não a grade com outra cara).

**D64. A conversa orquestradora mostra a lista viva dos agentes na aba
Agentes da Crew.** Passo 14 da primeira fatia do orquestrador (registro
`202610132000`, RN-9, `.p` §11), feito antes do daemon que o alimenta, com
autorização do dono para paralelizar a interface. O desktop já lia e conferia
os eventos da orquestradora desde o passo 2; agora eles viram uma lista.

- **Onde mora.** No painel direito da Crew, numa quarta aba, **Agentes**, que
  aparece só quando a sessão diz `role: orchestrator` e abre por padrão nela;
  as outras três (Mudanças, Pronto, Sessão) continuam ao lado, como D63 as
  deixou. Uma conversa comum não ganha aba nenhuma. O contador da aba é o de
  agentes em curso — nem merged nem fechado.
- **A gramática é a da Mudanças** (D41): uma linha de resumo ("6 agentes em
  curso · 8 na lista") e uma linha por filho — a marca, o título e o estado
  em palavra à direita; o `owns` resumido ("internal/a/ +2") com o PR e o CI
  à direita; os tokens somados (entrada · saída · cache); e o detalhe do
  estado quando há ("revisando", "medindo o gate", "o turno terminou em
  max_iterations"). A árvore segue `parent_id`: os filhos da orquestradora no
  primeiro nível, um neto recuado embaixo do pai.
- **Os tons seguem D52 e D61.** Trabalhando, em revisão e PR no acento, porque
  estão em curso; esperando você — e um PR que espera a resposta do merge —
  no amarelo de espera; falhou em vermelho; despachado, merged e fechado em
  cinza. Estado que o app não conhece é dito em vermelho com o nome que o
  daemon mandou. O CI do PR é o de D61; o núcleo escreve na gramática do
  `GitChecks` e o golden diz "passing", então os dois são lidos.
- **Sem dinheiro.** Tokens de entrada, saída e cache, nada em dólar: preço e
  tabela são da segunda fatia (`.p` §2).
- **No topo**, "passo a passo" — o `Project.autonomy` desta fatia é sempre
  `step`; quando o desktop ler o `Project`, a palavra vem dele —, a saúde da
  main do último `main.health` e "◆ N perguntas", que abre a fila: o tipo, o
  filho, o texto, a recomendada e o porquê, e os caminhos sensíveis de um
  merge. A pergunta de um filho aparece também na linha dele. Responder daqui
  fica para quando o app falar com `GET/POST /questions`, e a fila diz isso.
- **Clicar abre o filho no meio**, como clicar na lista de sessões: ao vivo,
  continua acompanhando; gravado, mostra o histórico (D33, D63). Um filho cuja
  conversa não está na lista da janela — despachado sem sessão ainda, merged
  ou fechado — fica desabilitado e diz "sem conversa na lista desta janela".
  A grade não muda.
- **Só o fio (D8), num redutor puro.** `state/agents.ts` dobra `child.*`,
  `review.verdict`, `gate.measured`, `question.*`, `merge.*` e `main.health`
  dentro da visão da sessão, sobre a resposta do daemon a `/children` quando o
  app a ler. Um fechado sai quando essa resposta não o traz mais — a
  retenção é do daemon (`orchestrator.closed_retention`, 1h), nunca um relógio
  do cliente —, e um que falhou não sai sozinho. Evento sobre filho que a
  lista não conhece não inventa o filho.
- **Modo fixture.** Uma terceira gravação, `orchestra-recording.ts`, com uma
  orquestradora e um filho em cada um dos oito estados, duas perguntas e a
  main verde (`?fixture=crew-orquestra`), e uma orquestradora sem filhos
  (`crew-orquestra-vazia`).

As réguas: as telas medidas não têm orquestradora, e nenhuma mudou — 02 e 03
e as dezesseis da Crew a 0,00%. Nenhuma referência nova entra aqui: as
capturas do painel, nos dois temas, cheio e vazio, vão ao dono no PR, e a
régua só ganha o estado depois do visto dele. O cenário de `check:daemon` que
a `.i` pede para este passo (dois filhos, um esperando você, o merged saindo
depois da retenção curta) espera o daemon dos passos que produzem os eventos
e a rota `/children`.

Lacuna: os filhos trabalham cada um na sua worktree, e a lista de sessões
agrupa por pasta, então cada filho aparece como um projeto à parte. Agrupar
por `project_id` é decisão própria.

Descartados: a lista no lugar da aba Mudanças (a orquestradora também muda
arquivos, e a Mudanças dela continua valendo); um painel novo, quarto na
largura (a Crew tem três, e D63 manteve); contar o fechado no relógio do
cliente (o `.p` dá a retenção ao daemon, e dois relógios discordariam); o
custo em dólar estimado pelos tokens (inventaria uma tabela que a fatia não
tem).
