# Decisões do desktop

O porquê do que o cliente desktop faz. Decisões de interface moram aqui, e não em
specs SDD: o comportamento do agente e o protocolo continuam com a disciplina de
spec do núcleo (`docs/specs/architecture/`), e o desktop **não muda o
protocolo** — o que ele precisa do núcleo vira pedido, com spec lá. O design de
referência foi o handoff v2 (`refs/design/desktop/`) até a grade de atenção (D29,
D30); desde 2026-10-13 a régua visual mede contra capturas da janela aprovada (D18).

Quatro partes: o que foi decidido, o que ficou em aberto, o que o design pede e o
protocolo não traz, e o que o desktop pede ao núcleo. As decisões de D21 em
diante moram em outros arquivos, para que cada um fique abaixo de 500 linhas:
de D21 a D40, sem D39, em [`DECISIONS-2.md`](DECISIONS-2.md); D39 e de D41 a
D53, sem D49 e D52, em [`DECISIONS-3.md`](DECISIONS-3.md); D49, D52 e de D54 a
D61 em [`DECISIONS-4.md`](DECISIONS-4.md); e de D62 em diante em
[`DECISIONS-5.md`](DECISIONS-5.md). A numeração é uma só, e uma referência a
"D29" vale em todos; a lista abaixo diz onde cada uma está.

As que estão nos outros arquivos:

- [D21](DECISIONS-2.md) — A lista de sessões vem do protocolo.
- [D22](DECISIONS-2.md) — `↵` nega a aprovação, como na TUI.
- [D23](DECISIONS-2.md) — Mensagem durante o turno redireciona, como na TUI.
- [D24](DECISIONS-2.md) — O loop é uma tela sobre o mecanismo que existe.
- [D25](DECISIONS-2.md) — Tokens durante o turno: só o tempo, por enquanto.
- [D26](DECISIONS-2.md) — Os dados do app ficam numa pasta só dele (o nome do produto, decidido depois em D65).
- [D27](DECISIONS-2.md) — O ⌘K procura na lista de conversas, pelo título e pelo projeto.
- [D28](DECISIONS-2.md) — Trocar de modelo é continuar a conversa numa sessão nova.
- [D29](DECISIONS-2.md) — Grafite: escuro, um acento, a fonte do sistema (a cor, substituída por D52).
- [D30](DECISIONS-2.md) — A janela abre numa grade de atenção.
- [D31](DECISIONS-2.md) — Cada janela da grade é uma conversa em que se trabalha.
- [D32](DECISIONS-2.md) — Uma segunda versão da tela, Crew, ao lado da grade.
- [D33](DECISIONS-2.md) — A Crew é um painel com trilho de seções, só com o que o daemon tem.
- [D34](DECISIONS-2.md) — O texto do modelo é desenhado como o markdown que ele é.
- [D35](DECISIONS-2.md) — Um tema claro, sóbrio, à escolha; o escuro continua o padrão.
- [D36](DECISIONS-2.md) — O modo da sessão muda pela janela, como na TUI.
- [D37](DECISIONS-2.md) — Um comando que o agente propôs pode ser rodado pela pessoa, num terminal logo abaixo da conversa, e o resultado volta ao agente.
- [D38](DECISIONS-2.md) — Modelo e modo escolhidos antes de a sessão abrir.
- [D39](DECISIONS-3.md) — A aprovação oferece as respostas que a TUI oferece.
- [D40](DECISIONS-2.md) — Desfazer, resumir e rodar um comando, pela janela.
- [D41](DECISIONS-3.md) — Onde a conversa trabalha: sessão, pasta, branch de agora, worktree e o diff da pasta.
- [D42](DECISIONS-3.md) — Uma conversa terminada abre no histórico dela.
- [D43](DECISIONS-3.md) — Renomear uma conversa pela janela.
- [D44](DECISIONS-3.md) — O turno ao vivo: rodadas, chamadas em voo e o que o turno custou.
- [D45](DECISIONS-3.md) — Uma chamada de ferramenta do modelo abre no que ela fez.
- [D46](DECISIONS-3.md) — Imagens vão com o turno.
- [D47](DECISIONS-3.md) — O campo lembra o que se pediu e guarda o que vem depois, como na TUI.
- [D48](DECISIONS-3.md) — Os comandos do campo num menu, e uma ajuda com eles e com as teclas.
- [D50](DECISIONS-3.md) — Os comandos que a pessoa escreveu, no menu e no campo, como na TUI.
- [D51](DECISIONS-3.md) — Encerrar uma sessão pelo menu ⋯, perguntando antes quando um turno roda.
- [D53](DECISIONS-3.md) — A memória e as skills de um projeto, alcançadas da grade.
- [D49](DECISIONS-4.md) — O mascote entra no app: o ícone e o estado de cada conversa.
- [D52](DECISIONS-4.md) — Laranja: a cor do dcode é a do mascote.
- [D54](DECISIONS-4.md) — As luzes dos critérios andam a cada conferência.
- [D55](DECISIONS-4.md) — Um loop começa e para pela janela, e ocupa uma janela que segue a sessão corrente dele.
- [D56](DECISIONS-4.md) — A definição de pronto proposta é revisada antes do trabalho, num card.
- [D57](DECISIONS-4.md) — O plano do turno fica fixo no alto, como checklist, enquanto o turno roda.
- [D58](DECISIONS-4.md) — A linha de atividade mostra o que o turno tocou.
- [D59](DECISIONS-4.md) — PDFs e arquivos de texto vão com o turno, como as imagens.
- [D60](DECISIONS-4.md) — O painel de mudanças cria o PR e mostra o CI dele, sempre confirmado.
- [D61](DECISIONS-4.md) — As cores do PR e do CI seguem a gramática de D52.
- [D62](DECISIONS-5.md) — A revisão mostra as propostas anteriores do turno, recolhidas embaixo da escrita.
- [D63](DECISIONS-5.md) — A Crew é a tela principal e leva todo elemento novo; a grade fica como alternativa.
- [D64](DECISIONS-5.md) — A conversa orquestradora mostra a lista viva dos agentes na aba Agentes da Crew.
- [D65](DECISIONS-5.md) — O produto se chama DCode Box; a marca na barra do topo continua "dcode".

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
barra do daemon diz **gravação**, nunca "daemon" — um ponto verde sem conexão
seria um estado que parece sucesso. Enviar, parar e responder aprovação avisam
que nada foi enviado. Sem estado no endereço, a gravação abre na grade (D30).

**D6. O tema é o de D29, na cor de D52** (era: os tokens do handoff nos dois
temas, seguindo o sistema), escuro por padrão e claro à escolha (D35). Os tokens são variáveis CSS,
num bloco só por tema (`styles/tokens.css`, e os da Crew em `crew.css`). A Geist
Mono vem do `@fontsource-variable` (OFL), empacotada, e é a reserva da mono onde o
sistema não tem uma — e também do texto no Linux, que não tem uma fonte de
sistema só (`--dc-font` em `tokens.css`); a Geist e a Fragment Mono da Crew
(D32) vêm empacotadas do mesmo jeito. Nada vem da rede em tempo de execução. Com
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

**D13. Selo do turno:** `passed` → "✓ verificado · n critérios" (ok); `failed` →
"✗ não verificado · …", com os critérios que falharam (erro, nunca a cor de perigo,
que é só do acesso total); `stale` → "⚠ não conferido · mudou depois do check" e
`unavailable` → "⚠ não conferido · nada pôde ser conferido" (esmaecido); `clean` ou
ausente → nada. Era em inglês, como o handoff; em português desde a grade (D30),
como o resto da janela (D17). Num painel, o selo vem antes de cada critério, com
✓, ✗ ou ? por critério quando o turno os nomeia. Um turno que escreveu onde o
trabalho é medido (`touched_protected`) nunca omite isso: o selo ganha "· tocou a
régua: …" e deixa de ser verde — um "verificado" medido com a régua que o próprio
turno mudou não é um verificado comum.

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

**D17. Os textos da interface são em português**, os do design, literais — os da
grade (D30) são deste app, não do handoff; um
segundo idioma é um arquivo (`src/renderer/text.ts`), não uma caça.

**D18. Verificação visual.** `npm run check:visual` compara, com pixelmatch, a
janela em modo fixture com os PNGs de `refs/design/desktop/screenshots/` dos
estados de `tests/visual/states.json`, em Chromium, escuro, movimento reduzido,
relógio fixo. Os controles de janela do macOS são desenhados pelo renderer no modo
fixture (o navegador não tem os nativos), então a faixa do título é comparada, não
mascarada. As referências têm 1602×962 — a borda de 1px da janela do mock em volta
— e a comparação usa o miolo de 1600×960. O limite e o raciocínio estão em
`tests/visual/states.json`.

Até a grade (D29, D30), as referências eram os PNGs do handoff v2; o mock do
próprio design, desenhado por este Chromium, ficava a 1,15% e 1,19% delas, e esse
era o piso. Desde 2026-10-13, com o visual novo aprovado pelo dono, as referências
dos estados medidos (02 e 03) são capturas da janela aprovada — o topo da pilha da
Crew, `9656919` na `main` (#433) —, feitas pela própria verificação: o miolo é
o screenshot dela, a borda é a do PNG antigo. Foram recapturadas do mesmo jeito
duas vezes, cada uma com autorização do dono no PR: a 03 com as respostas da
aprovação (D39), e as duas com o mascote e o Laranja (D49, D52). O piso medido
é 0 px nos dois estados, em execuções repetidas; os outros PNGs da pasta continuam os do handoff. Em 2026-10-17, com
a autorização do dono para estender a régua, entraram de 05 a 12 — as telas
feitas desde D41, que não tinham referência, na Crew que D63 fez tela
principal, cada uma no claro e no escuro —, capturados do mesmo jeito sobre o
PR da D63, à espera do visto do dono em cada uma. O `states.json`
passou a dizer, por estado, o esquema de cores, o que a janela acha no
armazenamento dela e as teclas e cliques antes da captura; o que a primeira
gravação não mostra sem mexer na 02 e na 03 vem de uma segunda
(`src/fixtures/loop-recording.ts`). Uma mudança visual
futura troca as referências num PR próprio, depois do visto de quem aprova o
visual: a régua é caminho protegido do loop (`tests/visual/**`,
`refs/design/**`), e trocá-la não é progresso do código que a mede.

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

## Decisões em aberto

A1, A2 e A4 foram decididas e viraram D22, D23 e D24; A3 foi decidida por D39.

**A3. Três respostas ou cinco.** Decidida em D39: a janela oferece as
respostas que a TUI oferece — quatro para a rede (uma vez, neste projeto,
sempre, negar), três para as outras travessias (uma vez, nesta sessão, negar).
Era: o protocolo tem cinco decisões (`allow`, `allow_session`,
`allow_project`, `allow_always`, `deny`), a TUI oferece
`allow_project`/`allow_always` para a rede, o design mostrava três e a janela
mostrava as três.

**A5. Voltar/avançar e recolher a lateral.** ←/→ andam no histórico do palco —
as conversas maximizadas nesta janela e a grade — (← fica esmaecido até haver para
onde voltar). Grade, pelo botão e pelo ⌘0, volta à grade (D30); nova sessão, pelo
botão e pelo ⌘N, pergunta onde numa janela da grade (D31); procurar, pelo botão e
pelo ⌘K, abre a busca (D27); o chip do modelo, numa janela ou na conversa
maximizada, o menu da troca (D28). Recolher a lateral e o menu da sessão saíram com a grade, e
rotinas saiu da lateral até existir. Adicionar projeto avisa que ainda não
existe; o ⚙ do pé da lateral abre o tema desde D35 (na Crew, a seção
Configurações, D33); anexar existe desde D46.

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
  O evento está **feito** (#471, `done.checked`) e a janela o lê (D54); a
  sequência no daemon também (#474, `/v1/loops`), e a janela começa, segue e
  para um loop (D55).
- **N5. Os modelos que uma sessão pode pedir** — `client-server-protocol`, MINOR.
  **Feito** (#422). `GET /v1/models?workspace=` responde o padrão do projeto e
  cada perfil do `models.toml`, o do usuário com o do projeto por cima, cada um
  resolvido pelo caminho que monta a sessão e dizendo se a família tem medição;
  nunca a chave, nem a máscara, nem a impressão digital dela. Pedido por D28.
- **N6. As skills e a memória de um projeto** — `client-server-protocol`, MINOR.
  **Feito** (#432). `GET /v1/skills?workspace=` e `GET /v1/memory?workspace=`
  respondem o que uma sessão no projeto tem: as skills, com a origem e se pedem
  permissão, nunca o corpo; e as memórias do arquivo do projeto, com o que está
  velho e o que fica fora do que a sessão lê. Os tipos vêm do `generated.ts`, e
  `src/protocol/crew.ts` só os confere na fronteira. Pedido por D33.
- **N7. O histórico de uma conversa gravada** — `client-server-protocol`, MINOR.
  **Feito** (#448). `GET /v1/conversations/{id}/events?from=` responde uma
  página da conversa lida do registro, sem carregá-la nem continuá-la — a
  cadeia que ela continua primeiro, numerada de 1 sob o id dela —, com teto de
  5000 eventos por página e `next` dizendo onde continuar. Pedido por D42.

A ordem: o N1 antes da conexão (o loop de `docs/loop/`); o N2 junto com ela —
sem ele, uma sessão que o desktop abre num projeto ignoraria o `.dcode/config.toml`
desse projeto —; o N3 antes da lateral com as gravadas, do ⌘K e da troca de
modelo; o N5 antes do menu de modelo; o N4 antes da tela do loop. O N1, o N2, o
N3 e o N5 estão feitos.
