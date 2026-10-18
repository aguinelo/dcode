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
- **Os cantos da Crew são os dela.** A grade fica nos 4px de D29 e D52
  (`--dc-radius`); na Crew, os cartões, os painéis e as folhas usam 8px
  (`--dc-radius-lg`, em `styles/crew.css`), a medida que o visual dela já
  tinha quando o dono o aprovou, e os controles pequenos continuam nos 4px e
  3px dos tokens. Fora dos tokens fica um valor escrito à mão: o botão de
  tentar de novo do histórico (D42), a 6px em `styles/history.css`.
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

**D65. O produto se chama DCode Box; a marca na barra do topo continua
"dcode"** (emenda D26, que tinha deixado o nome de fora). Registro de uma
decisão do dono, de 2026-10-13, feita em código antes de ter número aqui
(#468, #469): rodado a partir do código, o app aparecia no Dock e na barra de
tarefas como "Electron", e o dono pediu o ícone do mascote (D49) e o nome
"DCode Box".

- **Onde o nome aparece:** é o nome do produto — o `productName` do
  `package.json`, o `app.setName` do processo principal, o título da janela
  (`index.html`), o diálogo de quando o app não abre e a pergunta de fechar
  (D19). No macOS, rodado do código, o nome da barra de menus e o do Dock vêm
  do pacote do próprio Electron, e o `npm start` os escreve nele antes de
  abrir (`scripts/dev-name.mjs`); um app empacotado os tira do próprio pacote.
- **Por que um nome à parte:** foi o nome que o dono pediu, e ele separa o
  app do que já se chama `dcode` — o comando, o daemon e a TUI —, de que o app
  é uma janela a mais. O nome do produto é o que o sistema mostra do app; o do
  comando continua o de sempre.
- **A pasta dos dados não muda.** Ela nunca seguiu o nome do produto: continua
  `dcode-desktop` (D26), e trocar o nome não move as preferências de ninguém.
- **A marca na barra do topo continua "dcode"** (`TopBar.tsx`). Ela nomeia a
  ferramenta que a janela conduz — o mesmo nome que se digita no terminal —, e
  a barra está nas telas que a régua visual mede: trocá-la mudaria todas as
  referências sem mudar o que a janela faz.

Descartado: trocar também a marca da barra (move a régua inteira por um nome
que a pessoa já lê no título da janela e no Dock).

**D66. A janela age sobre a orquestradora: responde a fila, abre a do projeto
pelo "Nova" e agrupa os filhos sob o projeto** (fecha as lacunas de D64).
Passo 14 da primeira fatia do orquestrador (registro `202610132000`, decisão
4 da `.i`: o desktop ganha a opção de abrir a orquestradora do projeto, ao
lado da conversa comum, que continua o padrão), sobre as rotas que o daemon
já serve (`.p` §5.1).

- **Responder da aba Agentes.** "◆ N perguntas" abre a fila como antes, e cada
  pergunta ganha um botão por opção que o daemon mandou — merge, devolver e
  fechar num merge; continuar e parar no teto; as do desenho —, a recomendada
  no acento. Merge e desenho levam uma nota opcional (o porquê, o que falta);
  o teto não, porque as duas respostas dizem tudo. Nada é otimista: a
  pergunta sai quando o daemon toma a resposta, e o `question.answered` que
  chega de outra janela ou da TUI a tira do mesmo jeito. A primeira resposta
  vale: `question_already_answered` é dito na própria pergunta, com a
  mensagem do daemon, e a fila é relida para mostrar o que ele guarda.
- **Abrir a orquestradora do projeto, no menu "Nova".** O menu ganha um
  terceiro alternador, "◆ orquestradora", ao lado de "com modelo e modo" e de
  "↻ loop": ligado, o título vira "Abrir a orquestradora de", e escolher um
  projeto — ou "Outra pasta…" — faz `POST /projects` com a pasta, pede a
  sessão a `GET /projects/{id}/orchestrator` e a põe no meio, como uma sessão
  nova. Uma pasta que já tem projeto abre a mesma orquestradora.
  `orchestrator_unavailable` é dito com a razão do daemon, que nomeia a chave
  desligada, e com o que ligar: `orchestrator.enabled = true` e
  `record.enabled = true` (ou `DCODE_ORCHESTRATOR_ENABLED=true`), reiniciando
  o daemon. É o menor lugar coerente: a conversa comum continua sendo o que
  "Nova" abre sem escolha, e o que muda o tipo do que se abre já mora ali (o
  loop). O cabeçalho do projeto na lista foi descartado: está em toda tela
  medida da Crew.
- **Escondido enquanto o loop está ligado**, como "Outra pasta…" já fica. A
  folha do loop é um modo próprio, e o menu com o loop ligado é a referência
  11, nos dois temas: mostrar o alternador ali mudaria a régua sem mudar o
  que o loop faz. Se o dono quiser os três sempre visíveis, é recapturar a 11.
- **Os filhos sob o projeto na lista de Sessões.** Cada filho trabalha na sua
  worktree, e a lista agrupava por pasta: agora um filho vai para a pasta de
  quem o despachou, subindo pelos `parent_id`; com o pai fora da lista, para a
  da orquestradora do mesmo `project_id`; sem nenhum dos dois, para a própria
  pasta — nunca some. Dentro de cada grupo da Crew (Ativas, o do projeto,
  Mais antigas) um filho vem logo abaixo do pai, recuado 14px por nível, com
  "↳" antes da pasta; um filho cujo pai está em outro grupo fica na margem. O
  agrupamento vale também para a lateral da grade, que lê a mesma lista de
  projetos; o recuo é só da Crew, e a grade não muda de desenho.
- **`/children` e `/questions` alimentam a lista** ao abrir a conversa
  orquestradora, e de novo cada vez que o fluxo dela fica 1,5s quieto. A
  resposta do daemon substitui os filhos que os eventos tinham dobrado — é
  ela que segura a retenção de um fechado (D64) —, e os eventos que chegam
  depois dobram por cima. Reler no silêncio é o que mantém a lista na resposta
  do daemon sem um relógio da janela, e o que corrige a primeira leitura
  quando o replay de abertura ainda chegava. Uma leitura que falha é dita no
  alto da aba, e a lista dos eventos continua. Uma orquestradora terminada,
  lida do histórico, não lê nada: não há daemon que a guarde.
- **Só o fio (D8).** As respostas das rotas são conferidas na fronteira
  (`protocol/projects.ts`) contra os tipos gerados, e o processo principal
  confere o que a janela manda antes de enviar: a pasta absoluta, os ids no
  alfabeto do daemon, a resposta como id de opção de até 64 caracteres e a
  nota de até 2000.
- **Modo fixture.** A gravação da orquestra ganha uma pergunta de teto e a
  terceira opção do merge; responder diz que nada foi enviado, e abrir a
  orquestradora também. A lista agrupada aparece em `crew-orquestra`.

As réguas: as dezoito referências continuam a 0,00% — nenhuma tem filho de
orquestradora, e o menu medido (11) é o do loop ligado. O `check:daemon`
passava 13 de 13 quando esta decisão entrou; o cenário com dois filhos e a
retenção curta que a `.i` pede veio depois, com autorização do dono para
`tests/daemon/` (#553, `orchestra-lists`), e a régua passa 14 de 14. Ele achou
que um filho merged nunca saía da lista, corrigido no núcleo (#552). As capturas da
fila, da entrada no "Nova" e da lista agrupada, nos dois temas, vão ao dono
no PR.

Descartados: o ⌘K como lugar de abrir a orquestradora (a busca é a tela 12,
medida); a resposta otimista, tirando a pergunta antes do daemon (perder para
outra resposta deixaria a janela mostrando uma resposta que não valeu); nota
também no teto (continuar e parar não pedem porquê); reler `/children` num
relógio fixo (D64 deu a retenção ao daemon, e o silêncio do fluxo já diz
quando reler); recuar os filhos também na grade (a grade não muda de desenho
nesta fatia).

**D67. A pessoa decide na conversa da orquestradora, num cartão de perguntas
igual ao do Claude Code** (emenda D66). Decisão do dono, de 2026-10-18, que o
núcleo já serve (`.p` §5.5 da orquestração, #558): as perguntas do mesmo tipo
que esperam juntas são um cartão, e o cliente o desenha como o cartão de
pergunta do Claude Code que o dono usa.

- **O cartão mora na conversa**, no ponto do fluxo em que o `question.card`
  chegou pela primeira vez, e muda ali mesmo a cada versão nova — na Crew e na
  janela da grade, que compartilham o corpo da conversa. Um cartão que o fluxo
  não viu, lido de `GET /questions` ao abrir ou da resposta de uma
  confirmação, entra no fim. Uma versão mais velha que a desenhada não volta.
- **Um item por vez**, com "1 de N" e as setas ‹ › no alto à direita; a
  etiqueta (`header`) pequena ao lado do tipo, e o texto da pergunta como
  título. Num merge, embaixo do título, o `summary` da orquestradora e a
  `evidence` compacta — veredito da revisão, resultado do gate, o comando e o
  commit curto —, lida do log e nunca do que o filho disse.
- **As opções são linhas de rádio**: rótulo em negrito e `description`
  embaixo, a `selected` marcada, a `recommended` com "(Recomendado)"; por
  último "Outro", que abre um campo de texto, quando o item tem `allow_other`.
  As palavras vão como `answer: "other"` com a nota; o daemon diz o que fez
  delas (repassadas à orquestradora num merge, a resposta num desenho).
- **Rodapé**: "Aplicar a recomendação a todos" à esquerda; "Pular" e
  "Próximo ⌘↵" à direita. No último item "Próximo" vira "Confirmar", que manda
  **todas** as escolhas num `POST /questions/answers` só, com os pulados em
  `skipped`. Um item que a pessoa não chegou a ver vai como pulado: nada é
  respondido sem ser visto, e quem quer responder tudo sem olhar tem o
  "Aplicar a recomendação a todos", o mesmo `POST` com a recomendada de cada
  item. "Pular" no último item pula e confirma o resto.
- **Proposto**: o cartão que a orquestradora fez das palavras da pessoa abre
  com "Você pediu: …" e o porquê dela, e as escolhas propostas marcadas, a
  nota proposta junto. Só a confirmação da pessoa vira resposta.
- **Teclado**: ↑/↓ movem a escolha, ↵ e ⌘↵ avançam (e confirmam no último),
  Esc não fecha nada — o cartão fica no fluxo, e o Esc não chega à janela.
- **Nada é otimista.** O item muda quando o cartão do daemon diz; o resultado
  de cada resposta aparece embaixo do item — respondida, repassada, pulada ou
  recusada com a mensagem do daemon (`question_already_answered` dito como
  D66 já dizia). Um lote recusado inteiro é dito no cartão.
- **O selo "◆ Precisa da sua contribuição · N pendentes"**, no amarelo de
  espera de D52, fica embaixo do cartão enquanto `pending` passa de zero. Um
  cartão fechado vira a lista do que foi respondido, sem selo.
- **A aba Agentes leva ao cartão.** "◆ N perguntas" rola até o cartão mais
  recente que tem uma das perguntas e o foca, em vez de abrir a fila; sem
  cartão desenhado, abre a fila de D66 como antes.
- **Só o fio (D8).** O cartão é conferido na fronteira (`QUESTION_CARD`), a
  resposta do lote também (`decodeAnswers`), e o processo principal confere o
  lote antes de enviar: ids no alfabeto do daemon, cada resposta como em D66,
  pulo só com o id do cartão, nem vazio nem acima de 64 itens.
- **Modo fixture.** A quarta gravação, `questions-recording.ts`
  (`?fixture=crew-perguntas`): três merges num cartão e o cartão proposto das
  palavras da pessoa.

As réguas: as dezoito referências continuam a 0,00% — nenhuma tem
orquestradora. O `check:daemon` passa 14 de 14: o `orchestra-lists`, com
autorização do dono para mexer só nele, responde agora o merge pelo cartão
(escolhe Merge, Confirmar, um `POST /questions/answers`) depois de o "◆ 1
pergunta" focar o cartão, e confere o mesmo desfecho. As capturas — o cartão
de três merges no primeiro e no último item, e o proposto, nos dois temas —
vão ao dono no PR.

Descartados: o cartão só no painel da direita (o dono quer decidir na
conversa); todos os itens abertos de uma vez (o cartão do Claude Code mostra
um, e três merges com evidência não cabem na altura da conversa); responder
os itens não vistos com a seleção (uma resposta que a pessoa não leu); a
fila de D66 ao lado do cartão (duas maneiras de responder a mesma pergunta na
mesma tela).
