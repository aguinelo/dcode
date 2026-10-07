# Changelog — desktop

🇺🇸 [English version](CHANGELOG.md)

O registro próprio do cliente desktop, versionado à parte do núcleo: tags
`desktop-vX.Y.Z`, a partir de 0.1.0. Nada foi publicado ainda. O que mudou e por
quê, uma entrada cada; o detalhe de uma decisão está em `docs/DECISIONS.md`.

---

## Não publicado

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
