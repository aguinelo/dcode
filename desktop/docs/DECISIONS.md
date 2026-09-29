# Decisões do desktop

O porquê do que o cliente desktop faz. Decisões de interface moram aqui, e não em
specs SDD: o comportamento do agente e o protocolo continuam com a disciplina de
spec do núcleo (`docs/specs/architecture/`), e esta versão **não muda o
protocolo**. O design de referência é o handoff v2 (`refs/design/desktop/`).

Três partes: o que foi decidido, o que ficou em aberto, e o que o design pede e o
protocolo não traz.

## Decisões

**D1. Electron, com Electron Forge (Vite + TypeScript), React e npm.** O Chromium
embutido faz o app ter a mesma cara no macOS, no Linux e no Windows. Mac primeiro.

**D2. Segundo cliente do daemon, como a TUI.** O processo principal é quem vai
falar com o socket Unix (próxima versão). O renderer não vê Node nem o socket
(`contextIsolation` ligado, `nodeIntegration` desligado, `sandbox` ligado); o
preload expõe só `platform` e `user()`. Nenhuma porta local é aberta. O renderer
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

**D5. Nesta versão os dados vêm de uma gravação.** Eventos no formato exato do
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
`tool.approval_required`.** O `expires_at` do pedido não é lido (ver L9).

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

## Decisões em aberto

**A1. `↵` na aprovação.** O design liga `↵` a "Permitir uma vez", pré-selecionada.
Na TUI, `↵` **nega**, e isso é invariante da spec (`client-tui.p.spec.md`: "aprovação
pendente bloqueia a entrada e tem negar como default"). Aqui `↵` não responde:
`1`, `2`, `3` respondem, `esc` nega, e `↵` avisa que não responde. Nenhuma opção
aparece pré-selecionada. Decidir entre o default do design e o da spec.

**A2. Mensagem durante o turno.** O placeholder do design diz "Mensagem entra na
fila depois deste turno". Na TUI, o que se digita durante o turno **dirige** o
turno (`POST …/steer`, `client-tui.p.spec.md`); só entra na fila mensagem com
imagem ou comando que custa um turno. O texto do design ficou; nenhuma das duas
semânticas foi construída (nada é enviado nesta versão). Decidir.

**A3. Três respostas ou cinco.** O protocolo tem cinco decisões (`allow`,
`allow_session`, `allow_project`, `allow_always`, `deny`), e a TUI oferece
`allow_project`/`allow_always` para a rede. O design mostra três; esta versão
mostra as três.

**A4. O estado "loop".** O fio só tem `idle | running | blocked | closed`; loop
não tem representação. A sessão em loop do mock aparece como rodando, sem
`↻ 8/100`, e conta em "n rodando" (por isso a barra diz "2 rodando" onde o mock
diz 1).

**A5. Voltar/avançar e recolher a lateral.** ←/→ andam no histórico de sessões
abertas nesta janela (← fica esmaecido até haver para onde voltar). Recolher a
lateral, nova sessão, procurar (⌘K), rotinas, anexar, trocar modelo, o menu da
sessão e configurações avisam que ainda não existem.

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
  atividade mostra só o tempo.
- **L6. Contexto antes do fim do primeiro turno.** Ver D15. `context.band` mede
  outra coisa (fração até o resumo) e vira nota, não medidor.
- **L7. Diff das escritas de filhos.** O `explore` não reporta `added`/`removed`,
  então o chip da sessão 02 mostra só a branch.
- **L8. Aprovação: recursos pedidos e política.** O pedido traz `boundary_crossed`,
  `reason` e `rule`; os caminhos pedidos (`package.json`, `node_modules/`) e o
  nome da política ("on-request") não viajam. O chip do composer mostra
  `sandbox_mode · mode` — o modo é o nome que a pessoa escolhe, e derivar a
  política dele seria uma segunda cópia da tabela do daemon.
- **L9. `expires_at` sempre zero.** O núcleo emite o pedido antes de fixar o
  prazo (`internal/loop/turn.go`), e uma aprovação pendente é negada sozinha
  depois de 2 minutos; o cliente só sabe quando chega o `tool.approval_resolved`.
- **L10. Estado agregado por sessão.** `GET /v1/sessions` lista só sessões vivas,
  sem título, turnos, selo, diff ou último evento; a lateral precisa do log de
  cada sessão (SPEC_GAPS C3). As sessões gravadas vêm do disco.
- **L11. Loop.** Iteração, limites, checks por iteração: nada disso existe no fio
  (SPEC_GAPS B e C1–C2). Ver A4.
- **L12. Quem usa.** Nome e iniciais do rodapé vêm do sistema, pelo processo
  principal (`id -F` no macOS, GECOS no Linux, senão o login) — não do protocolo.
- **L13. Status do daemon.** Esta versão não conecta; a conexão é a próxima.
- **L14. Estados.** O handoff lista `running | loop | awaiting_approval | idle |
  interrupted`; o fio tem `idle | running | blocked | closed`. "Esperando
  aprovação" é `blocked`; "interrompida" é um `reason` do `turn.completed`
  (aparece como nota no fluxo), não um estado; "loop" não existe (A4).
- **L15. Selo.** O handoff fala em `verified | unverified | not_verified`; o fio
  traz `Completion.verification = clean | passed | failed | stale | unavailable`.
  O mapeamento está em D13.
- **L16. Verbo da atividade.** O handoff pede um rodízio fixo de delegando,
  lendo, conferindo e pensando, "e depois vem de `tui.activity_verbs`" — essa
  chave de configuração não existe; o que existe é o catálogo por fase da TUI e o
  liga-desliga `DCODE_ACTIVITY_VERBS`. Ver D14. Por isso a tela 02 diz
  "Delegando…" onde o mock diz "Conferindo…".
