# Decisões do desktop

O porquê do que o cliente desktop faz. Decisões de interface moram aqui, e não em
specs SDD: o comportamento do agente e o protocolo continuam com a disciplina de
spec do núcleo (`docs/specs/architecture/`), e o desktop **não muda o
protocolo** — o que ele precisa do núcleo vira pedido, com spec lá. O design de
referência é o handoff v2 (`refs/design/desktop/`).

Quatro partes: o que foi decidido, o que ficou em aberto, o que o design pede e o
protocolo não traz, e o que o desktop pede ao núcleo.

## Decisões

**D1. Electron, com Electron Forge (Vite + TypeScript), React e npm.** O Chromium
embutido faz o app ter a mesma cara no macOS, no Linux e no Windows. Mac primeiro.

**D2. Segundo cliente do daemon, como a TUI.** O processo principal fala com o
socket Unix — os pedidos HTTP e os fluxos SSE —, e o renderer não vê Node nem o
socket (`contextIsolation` ligado, `nodeIntegration` desligado, `sandbox` ligado).
O preload expõe a API de `src/shared/api.ts`: um canal com nome para cada pedido e
para cada notícia — o estado do daemon, a lista de conversas, os eventos das
sessões abertas —, nenhum genérico, e o processo principal confere cada argumento
antes de usá-lo. Nenhuma porta local é aberta. O renderer
construído roda sob uma CSP que não deixa conectar a lugar nenhum (`connect-src
'none'`) e carrega fontes só do próprio pacote.

**D3. Os tipos do protocolo são gerados, não escritos.** `tygo` lê
`internal/protocol` e escreve `src/protocol/generated.ts`, que é commitado; o CI
regenera e reprova quando o arquivo difere. A dependência corre num sentido só: o
desktop lê o núcleo, o núcleo nunca lê o desktop, e nada entra no `go.mod` da
raiz. O que é escrito à mão é só o pareamento tipo de evento → payload, e ele não
pode divergir: cada forma (`Shape<T>`) precisa listar todos os campos do tipo
gerado com a mesma opcionalidade, ou o typecheck reprova; e um teste confere que
todo `Event*` declarado no núcleo tem leitor aqui.

**D4. O evento é validado na fronteira.** Payload que não bate com a forma vira
uma nota visível no fluxo da sessão ("Evento ilegível: …"), nunca é descartado.
Seq repetido é sobreposição de replay e não muda nada; seq que pula é o log com
buraco, e a lacuna é dita no fluxo.

**D5. No modo fixture os dados vêm de uma gravação.** Dentro do Electron a janela
é cliente do daemon (D2, D19). No navegador — `npm run dev:renderer` e a régua
visual, com `?fixture=` — ela mostra eventos no formato exato do
fio (`src/fixtures/recording.ts`), com só os campos e valores que um daemon
mandaria. A gravação é reproduzida como se tivesse acabado de acontecer: ao abrir,
todos os instantes andam o mesmo tanto até agora, e os relógios seguem dali. A
barra inferior diz **gravação**, nunca "daemon" — um ponto verde sem conexão
seria um estado que parece sucesso. Enviar, parar e responder aprovação avisam
que nada foi enviado.

**D6. Tema segue o sistema.** Tokens do handoff nos dois temas, como variáveis
CSS; `prefers-color-scheme` decide. Geist e Geist Mono vêm do `@fontsource-variable`
(OFL), empacotadas; nada vem da rede em tempo de execução. Com
`prefers-reduced-motion`, nada pulsa nem pisca e o verbo da atividade não roda: o
estado continua legível pela forma do glifo.

**D7. Preferências locais em `localStorage`:** ordem, rótulos e recolhimento dos
projetos, por caminho do workspace. Renomear muda o rótulo, nunca o diretório; um
nome vazio mantém o anterior. Preferência ilegível é dita num aviso e a lateral
começa do arranjo inicial. Sem ordem guardada, os projetos vêm por atividade mais
recente; dentro do projeto, a sessão mais nova primeiro, por criação — as linhas
não trocam de lugar sob o ponteiro enquanto as sessões trabalham.

**D8. Resumo da linha de ferramenta sai dos campos do `tool.completed`**
(`lines`, `files`, `added`, `removed`, `exit_code`), nunca do texto da saída. Os
textos seguem o design literalmente, inclusive "11 matches · 4 arquivos". Falha
mostra a primeira linha da saída, que é a mensagem do erro, não um número
reconstruído.

**D9. Delegação é o que a TUI já lê:** duas ou mais chamadas `explore`
adjacentes. O nome do filho é o último segmento do `path` da entrada (a TUI usa o
`path` inteiro); a coluna de dono é o `owns`. "Propriedade disjunta" só quando
todo filho declarou `owns` e nenhum caminho se sobrepõe. "n de N" conta os filhos
que voltaram, com ou sem resposta.

**D10. Uma chamada que pede aprovação é desenhada pelo card enquanto espera.**
Respondida, o card fica no lugar com a resposta, e a linha da ferramenta aparece
depois dele — a história se lê na ordem em que aconteceu.

**D11. O relógio "esperando · m:ss" conta do `at` do envelope do
`tool.approval_required`.** O `expires_at` chega preenchido desde o #403 e não é
desenhado: o design conta o tempo esperando, não o prazo.

**D12. A fronteira vem traduzida do `boundary_crossed`** ("Pede **rede**"); a
regra, quando existe, aparece; o `reason` do daemon (em inglês) fica no título do
elemento. Código de fronteira desconhecido aparece como veio.

**D13. Selo do turno:** `passed` → "✓ verified · n checks" (ok); `failed` → "✗ not
verified · …" (erro, nunca a cor de perigo, que é só do acesso total); `stale` e
`unavailable` → "⚠ unverified" (esmaecido); `clean` ou ausente → nada.

**D14. Verbo da atividade:** o catálogo por fase da TUI (`internal/tui/activity.go`),
em português — sinônimos da fase da ferramenta que roda, trocando a cada 2,4 s.
Sem ferramenta rodando, a palavra fixa "trabalhando". O verbo nunca aparece
sozinho: ao lado dele vai o fato (a ferramenta e o alvo, ou os filhos que ainda
rodam) e o tempo desde o `turn.started`.

**D15. Medidor de contexto** só depois do primeiro `turn.completed`, calculado de
`usage.context_tokens` sobre `context_window`, com as faixas do brief (esmaecido
até 74%, aviso até 89%, erro a partir de 90%). Antes disso não se desenha nada —
segmento sem dado não desenha.

**D16. Chip de diff (`+n −m`)** só quando alguma ferramenta reportou mudança,
somando `added`/`removed`, como a barra da TUI. "+0 −0" afirmaria que nada mudou
onde só não houve relato (ver L7).

**D17. Os textos da interface são em português**, os do design, literais; um
segundo idioma é um arquivo (`src/renderer/text.ts`), não uma caça.

**D18. Verificação visual.** `npm run check:visual` compara, com pixelmatch, a
janela em modo fixture com os PNGs do handoff, em Chromium, escuro, movimento
reduzido, relógio fixo. Os controles de janela do macOS são desenhados pelo
renderer no modo fixture (o navegador não tem os nativos), então a faixa do
título é comparada, não mascarada. As referências vêm com a borda de 1px da
janela do mock (1602×962) e a comparação usa o miolo de 1600×960. O limite e o
raciocínio estão em `tests/visual/states.json`. O que a medição mostrou: o mock do
próprio design, desenhado por este Chromium, fica a 1,15% e 1,19% das referências
— elas foram rasterizadas por um motor que não arredonda a altura de linha da
Geist como o Chrome no macOS, e as linhas de texto caem um pixel acima, linha a
linha. Esse é o piso; a janela mede 1,17% e 1,19%.

**D19. O desktop sobe o daemon quando nenhum responde.** O processo principal
procura um `dcode serve` no socket e anexa a ele; se nenhum responde, sobe um
como processo filho, no mesmo socket, e as TUIs abertas depois anexam a ele, como
já anexam a qualquer daemon que responda ali (`cmd/dcode/tui.go`). O filho morre
com o app, e as sessões dele junto: nada sobrevive a quem o criou, que é decisão
do núcleo. Fechar o app com sessão rodando ou esperando aprovação pergunta antes,
dizendo quantas param. Um daemon que o desktop não subiu não é encerrado por ele.

- **O binário** é o de `DCODE_BIN`, senão `~/.local/bin/dcode` (onde o `install.sh`
  e o `make install` põem), senão o do `PATH` do app. Não achar é dito, com os
  lugares procurados.
- **O socket** é o de `DCODE_SOCKET`, senão o que `dcode socket` imprimir (N1) —
  nunca uma segunda cópia da regra do núcleo.
- **O filho herda o ambiente do app.** Aberto do terminal (`npm start`, o único
  jeito hoje), é o do shell. Um app empacotado, aberto pelo Dock, não teria o
  `PATH` das ferramentas que o agente roda; ler o ambiente do shell de login é
  decisão do empacotamento.

Descartados: um serviço sempre de pé (launchd), que reabriria "nada sobrevive a
quem o criou"; e um daemon privado por app, que não compartilharia nada com a TUI.

**D20. Um daemon atende todos os projetos,** com a configuração resolvida no
workspace de cada sessão (N2, feito). Antes, o `dcode serve` resolvia uma vez, no
workspace em que subia, e toda sessão usava essa — e o desktop sobe o daemon fora
de qualquer projeto. Descartados: um daemon por projeto (mais processos, e cada TUI teria de
achar o socket do seu); manter como está (a lateral com vários projetos mentiria
sobre a configuração de cada um).

**D21. A lista de sessões vem do protocolo** (N3): vivas e gravadas, com resumo, e
um fluxo só de mudanças da lista. Abrir uma gravada continua a conversa numa
sessão nova (`CreateSession{Resume}`), como a TUI faz. Com o N3 feito, a lateral
lê `GET /v1/conversations` e acompanha `/v1/conversations/events`. Descartados: ler o disco como a TUI (acopla o desktop
ao formato do registro e duplica a leitura); só as vivas (a lateral perde o
histórico).

**D22. `↵` nega a aprovação, como na TUI** (era A1). Permitir é sempre escolha
explícita: `1` uma vez, `2` nesta sessão; `3`, `esc` e `↵` negam. A opção
destacada é negar, e só isso muda no design. O campo de mensagem fica desabilitado
enquanto a aprovação espera, com o texto "Responda à aprovação acima". Medido
antes de decidir: a tela 03 fica a 1,21% das referências, dentro do limite.
O `↵` da janela só responde com o foco em lugar nenhum: num botão de resposta ou
numa linha da lateral, o `↵` é daquele controle — senão `↵` sobre "Permitir uma
vez" negaria. O campo, ao ser desabilitado, devolve o foco à janela, e com ele as
teclas da resposta. Descartados: `↵` permite uma vez, como no design (aprovar sem ler vira o gesto mais
barato, e a invariante da TUI teria de mudar junto); `↵` sem efeito (uma regra
diferente em cada cliente).

**D23. Mensagem durante o turno redireciona, como na TUI** (era A2). O texto vai ao
modelo na próxima rodada do mesmo turno (`POST …/steer`) e aparece no fluxo
quando o daemon o entrega (`turn.steered`); imagem e comando que custa turno,
quando existirem aqui, entram na fila. O campo diz "Escreva para redirecionar
este turno" — medido: a tela 02 fica a 1,20%. Descartado: a fila do design — com
ela, corrigir um turno que foi para o lado errado exigiria interrompê-lo, e a
mesma sessão teria uma regra em cada cliente.

**D24. O loop é uma tela sobre o mecanismo que existe** (era A4): os critérios do
`done.toml`, a reentrada por ciclo e um evento por ciclo com o resultado de cada
critério — passou ou não, sem contagem de testes, porque critério é código de
saída (`agent-loop`). A sequência do `/loop` sai da TUI para o daemon, para os dois
clientes verem o mesmo loop (N4). Vem depois da conexão; até lá, a sessão em loop
aparece como rodando. Descartados: o loop novo de `refs/design/desktop/LOOP.md`
(iteração como turno inteiro, orçamentos, contagem por teste, `dcode loop` —
várias specs, e contraria o `agent-loop`); e decidir depois.

**D25. Tokens durante o turno: só o tempo, por enquanto.** A linha de atividade
mostra o tempo desde o `turn.started`; tokens e contexto mudam quando o turno
termina, como na TUI desde o #401. Descartado por ora: um evento de uso por
rodada, que seria mudança de protocolo (MINOR). A lacuna L5 fica, por decisão.

**D26. Os dados do app ficam numa pasta só dele.** O Chromium guarda na pasta de
dados do app o `localStorage` das preferências (D7), caches e cookies. O padrão do
Electron é o nome do produto dentro da pasta de dados do sistema — no macOS,
`~/Library/Application Support/DCode` —, e o sistema de arquivos ali não distingue
`DCode` de `dcode`, a pasta em que o próprio dcode guarda configuração, registros e
estado, o `models.toml` incluso. O app usa `dcode-desktop` nessa pasta, e
`DCODE_DESKTOP_USER_DATA` escolhe outra, absoluta — a régua dá uma a cada cenário.
Apagar os dados do app nunca leva a configuração do dcode junto. Descartado: trocar
o nome do produto, que é o nome da janela e do menu — decisão de design, não de
onde os arquivos ficam.

**D27. O ⌘K procura na lista de conversas, pelo título e pelo projeto.** Cada
palavra digitada precisa estar no título ou no projeto (o rótulo ou o caminho),
sem distinguir maiúsculas nem acentos. Os grupos são os do design: "Ativas"
primeiro — rodando ou esperando você —, depois um por projeto, na ordem da
lateral; uma conversa ativa aparece só em "Ativas". `↵` abre a do cursor pelas
regras de um clique na linha — uma terminada é continuada numa sessão nova (D21).
O vazio diz que nada foi achado e não promete o que não existe: o "↵ cria uma nova
com esse pedido" do design fica de fora até haver onde criá-la. O escopo do
projeto é o da sessão aberta, e some sem sessão aberta.

## Decisões em aberto

A1, A2 e A4 foram decididas e viraram D22, D23 e D24.

**A3. Três respostas ou cinco.** O protocolo tem cinco decisões (`allow`,
`allow_session`, `allow_project`, `allow_always`, `deny`), e a TUI oferece
`allow_project`/`allow_always` para a rede. O design mostra três; esta versão
mostra as três.

**A5. Voltar/avançar e recolher a lateral.** ←/→ andam no histórico de sessões
abertas nesta janela (← fica esmaecido até haver para onde voltar). Nova sessão,
pelo botão e pelo ⌘N, pede a pasta e abre a sessão nela; procurar, pelo botão e
pelo ⌘K, abre a busca (D27). Recolher a lateral, rotinas, anexar, trocar modelo, o
menu da sessão e configurações avisam que ainda não existem.

## Lacunas — o que o design pede e o protocolo não traz

Cada uma é desenhada com o que existe, e nada é inventado. Fechar qualquer uma é
mudança de protocolo, com changelog de spec no núcleo antes do código.

- **L1. Progresso por filho.** Os passos de um filho não são eventos da sessão
  (`internal/loop/delegate.go`: o filho roda sem emissor). A barra de um filho
  rodando fica vazia; a de um que voltou fica cheia, na cor do glifo.
- **L2. O que um filho está fazendo** ("charlie lendo internal/store/sqlite.go").
  A linha de atividade diz quais filhos ainda rodam.
- **L3. Contagens do filho.** Rodando: nenhuma (aparece "…"). Concluído: "leu n"
  vem de `files`, que o `explore` preenche com o que o filho leu; "escreveu n" só
  existe no texto da saída e não se lê texto.
- **L4. Nome curto do filho.** O `explore` recebe `task`, `path` e `owns`; alpha,
  bravo, charlie e delta do mock não viajam. Ver D9.
- **L5. Tokens durante o turno.** `usage` só chega no `turn.completed`; a linha de
  atividade mostra só o tempo. Fica assim por decisão (D25).
- **L6. Contexto antes do fim do primeiro turno.** Ver D15. `context.band` mede
  outra coisa (fração até o resumo) e vira nota, não medidor.
- **L7. Diff das escritas de filhos.** O `explore` não reporta `added`/`removed`,
  então o chip da sessão 02 mostra só a branch.
- **L8. Aprovação: recursos pedidos e política.** O pedido traz `boundary_crossed`,
  `reason` e `rule`; os caminhos pedidos (`package.json`, `node_modules/`) e o
  nome da política ("on-request") não viajam. O chip do composer mostra
  `sandbox_mode · mode` — o modo é o nome que a pessoa escolhe, e derivar a
  política dele seria uma segunda cópia da tabela do daemon.
- **L9. `expires_at` sempre zero.** Resolvida no núcleo (#403): o pedido chega
  com o prazo. Uma aprovação pendente continua negada sozinha depois de 2
  minutos, e o card não desenha o prazo (D11).
- **L10. Estado agregado por sessão.** `GET /v1/sessions` lista só sessões vivas,
  sem título, turnos, selo, diff ou último evento; a lateral precisa do log de
  cada sessão (SPEC_GAPS C3). As sessões gravadas vêm do disco. Decidido: uma rota
  no protocolo (D21, N3).
- **L11. Loop.** Iteração, limites, checks por iteração: nada disso existe no fio
  (SPEC_GAPS B e C1–C2). A sessão em loop do mock aparece como rodando, sem
  `↻ 8/100`, e conta em "n rodando" (por isso a barra diz "2 rodando" onde o mock
  diz 1). Decidido: uma tela sobre o mecanismo que existe (D24, N4).
- **L12. Quem usa.** Nome e iniciais do rodapé vêm do sistema, pelo processo
  principal (`id -F` no macOS, GECOS no Linux, senão o login) — não do protocolo.
- **L13. Status do daemon.** Fechada: a barra diz conectando, conectado — com a
  versão que o `GET /version` deu —, caiu ou não subiu, e os dois últimos dizem o
  porquê (D19).
- **L14. Estados.** O handoff lista `running | loop | awaiting_approval | idle |
  interrupted`; o fio tem `idle | running | blocked | closed`. "Esperando
  aprovação" é `blocked`; "interrompida" é um `reason` do `turn.completed`
  (aparece como nota no fluxo), não um estado; "loop" não existe (L11).
- **L15. Selo.** O handoff fala em `verified | unverified | not_verified`; o fio
  traz `Completion.verification = clean | passed | failed | stale | unavailable`.
  O mapeamento está em D13.
- **L16. Verbo da atividade.** O handoff pede um rodízio fixo de delegando,
  lendo, conferindo e pensando, "e depois vem de `tui.activity_verbs`" — essa
  chave de configuração não existe; o que existe é o catálogo por fase da TUI e o
  liga-desliga `DCODE_ACTIVITY_VERBS`. Ver D14. Por isso a tela 02 diz
  "Delegando…" onde o mock diz "Conferindo…".
- **L17. A prévia do ⌘K.** O design mostra a ferramenta que espera aprovação
  ("Esperando aprovação · bash") e quantos checks o selo conferiu ("✓ verified ·
  2 checks"). A lista de conversas traz o estado, o selo e o último evento, sem a
  ferramenta nem a contagem: a prévia diz "Esperando aprovação" e "✓ verified".

## Pedidos ao núcleo

O que as decisões acima pedem ao núcleo. Cada um é um PR do núcleo, com a spec
antes do código (`docs/conventions/SDD-HARNESS.md`) e changelog na família; o
desktop não os implementa, espera por eles.

- **N1. O socket fixo por usuário** — `client-server-protocol`. **Feito.** O
  caminho padrão lia `XDG_RUNTIME_DIR` e `TMPDIR`, que um app aberto pelo Dock,
  uma sessão SSH e um terminal podem ter diferentes. Agora é
  `/tmp/dcode-<uid>/dcode.sock`, sem ler o ambiente — `DCODE_SOCKET` continua
  sendo a escolha explícita —, numa pasta que só vale se for do usuário e só dele.
  `dcode socket` imprime o caminho, ou falha dizendo por quê, e é a ele que o
  desktop pergunta. Pedido por D19.
- **N2. Configuração por sessão** — `configuration`. **Feito.** O daemon resolvia
  a cadeia uma vez, onde subia, e toda sessão usava essa. Agora resolve no
  workspace de cada sessão, o `.dcode/config.toml` do projeto incluso, e
  configuração de projeto ilegível recusa a sessão dizendo por quê. Conserta também
  uma TUI anexada a um `dcode serve` que subiu em outro projeto. Pedido por D20.
- **N3. A lista de sessões** — `client-server-protocol`, MINOR. **Feito.**
  `GET /v1/conversations` lista as vivas e as gravadas, cada uma com título,
  projeto, estado, branch, modelo, turnos, selo, diff, última atividade e último
  evento; `GET /v1/conversations/events` abre com a lista inteira e manda só o
  que muda. A TUI passar a usá-la no seletor (`-r`), em vez de ler o disco, fica
  para um PR dela. Pedido por D21; fecha L10.
- **N4. O `/loop` no daemon** — `loop-command` e `client-server-protocol`, MINOR. A
  sequência sai do processo da TUI (`internal/tui/program.go`) para o daemon, e um
  evento por ciclo traz o resultado de cada critério. Pedido por D24; fecha L11.

A ordem: o N1 antes da conexão (o loop de `docs/loop/`); o N2 junto com ela —
sem ele, uma sessão que o desktop abre num projeto ignoraria o `.dcode/config.toml`
desse projeto —; o N3 antes da lateral com as gravadas, do ⌘K e da troca de
modelo; o N4 antes da tela do loop. O N1, o N2 e o N3 estão feitos.
