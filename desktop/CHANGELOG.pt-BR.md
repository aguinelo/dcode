# Changelog — desktop

🇺🇸 [English version](CHANGELOG.md)

O registro próprio do cliente desktop, versionado à parte do núcleo: tags
`desktop-vX.Y.Z`, a partir de 0.1.0. Nada foi publicado ainda. O que mudou e por
quê, uma entrada cada; o detalhe de uma decisão está em `docs/DECISIONS.md`.

---

## Não publicado

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
