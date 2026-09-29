# DCode Desktop — conferido contra o repositório

## Conferido contra o repositório — 2026-09-29

O handoff (`README.md`, `SPEC_GAPS.md`, `LOOP.md`) foi conferido contra a `main` em `f08d742` (0.21.1). Seguindo o precedente da TUI, os arquivos ficam verbatim. O que vem abaixo é o que o código diz, com arquivo e linha.

**Confere:**

- O daemon é dono da sessão, e o turno segue sem nenhum cliente conectado. Trocar de sessão na janela não para a anterior (`internal/session/session.go:16-18`, `internal/server/server.go:527-530`). Um daemon segura várias sessões vivas, até 64 por padrão (`session.go:659-678`, `cmd/dcode/serve.go:30`).
- Cada sessão nasce ancorada no próprio workspace. `CreateSessionRequest.Workspace` é absoluto e vem em cada pedido (`internal/protocol/protocol.go:216-218`, `server.go:205-209`). Sandbox, resolver, concessões, branch, cadeia de instruções, memória e skills saem dele (`internal/app/daemon.go:132-144`, `internal/app/app.go:483,499,536,607,626,652`). O daemon não conhece "projeto", então ordem, rótulo e recolhido como preferência local não brigam com nada (`protocol.go:167-168`).
- `running`, `idle` e `awaiting_approval` correspondem a `running`, `idle` e `blocked` (`protocol.go:151-156`; `blocked` é posto em `session.go:550`).
- Prévia e cabeçalho: modelo, workspace e branch estão em `protocol.Session` (`protocol.go:169-214`), com a branch congelada na criação (`protocol.go:184-189`). O `+38 −0` é a soma de `ToolCompleted.Added/Removed` feita no cliente, como a TUI já faz (`protocol.go:597-608`, `internal/tui/model.go:464`).
- Linha de ferramenta com resultado resumido: `ToolCompleted` traz `Lines`, `Files`, `Added`, `Removed`, `ExitCode`, `DurationMS` e `Diff` (`protocol.go:590-627`).
- O selo vem dos critérios, nunca da prosa (`protocol.go:666-672`). "Uma linha por check do `done.toml`, sem nomes fixos" é o formato real: cada `[seção]` é um critério com nome livre (`internal/app/done.go:29-78`).
- Delegação: o filho que não respondeu é nomeado no lugar (`internal/tools/explore.go:143-151`, `internal/tui/delegation.go:60-65,104-106`). O diretório dono vem do `owns` da chamada (`explore.go:55-66`). Os tokens dos filhos contam para o pai (`internal/loop/delegate.go:165-169`, `internal/loop/turn.go:380`).
- Linha de atividade: a cadência de 2,4 s é a da TUI (`internal/tui/activity.go:17-23`), e lá o tempo decorrido também é medido pelo cliente (`model.go:209-213,352`).
- Aprovação: comando, motivo e regra estão no `ApprovalRequest` (`protocol.go:358-374`). "Permitir nesta sessão" existe como `allow_session` (`protocol.go:297-299`) e é lembrado pela regra quando há uma, senão pela ferramenta e comando exatos (`session.go:506-521,570-575`). `3 Negar esc` bate com a TUI (`internal/tui/program.go:1509`).
- `⌘K`: `↑↓` sem dar a volta e `esc` que limpa o filtro antes de fechar são as regras da coluna da TUI (`internal/tui/railnav.go:61-74,153-165`).
- Medidor de contexto: `Usage.ContextTokens` sobre `ContextWindow`, medido pelo daemon (`protocol.go:202-205,697-713`).
- Barra inferior: `GET /health` e `GET /version`, com versão e protocolo, existem (`server.go:168-169,188-197`).
- Parar é `POST …/interrupt`, e anexar é `SubmitTurnRequest.Images` (`server.go:180`, `protocol.go:268-280`).
- `⌘Z` sobre um turno inteiro, filhos incluídos: `POST …/undo` desfaz o último turno, e o `Adopt` traz as escritas dos filhos para ele (`server.go:182,408-420`, `internal/tools/undo.go:200-216`).

**Diverge:**

1. **"O mesmo daemon da TUI e da CLI" só vale com `dcode serve` de pé.**
   - A TUI anexa ao socket padrão quando alguém responde. Senão, sobe um daemon embutido num socket privado (`cmd/dcode/tui.go:84-107,309-314`) que fecha todas as sessões quando ela sai (`server.go:147-154`).
   - O CLI não fala com daemon nenhum. `dcode "<tarefa>"` embute o motor (`cmd/dcode/main.go:1-7`, `cmd/dcode/once.go:55-61`), e `dcode sessions` lê o disco (`cmd/dcode/sessions.go:14-19`).
   - O caminho padrão depende do ambiente (`DCODE_SOCKET`, `XDG_RUNTIME_DIR`, `TMPDIR`: `daemon.go:50-62`). Se essas variáveis estiverem só no shell, um app aberto pelo Finder calcula outro caminho.
   - O socket é Unix, 0700 e sem autenticação. Expor em TCP seria mudança de contrato (`server.go:7-9,102-137`). No Electron, quem fala com ele é o processo main, não o renderer.
   - *Consequência:* um desktop que suba `dcode serve` no socket padrão é achado pelas TUIs abertas **depois** dele. As abertas antes ficam invisíveis no socket privado.

2. **Um daemon para vários projetos funciona pela metade.**
   - Sandbox e instruções são por sessão (ver Confere). A cadeia de configuração, porém, é resolvida **uma vez**, no `--workspace` de boot (`serve.go:42-49`), e vira o `Base` de toda sessão (`daemon.go:143`). Isso inclui a camada de projeto `<ws>/.dcode/config.toml` (`internal/config/toml.go:287-295`).
   - Resultado: uma sessão do projeto B num daemon aberto no A herda o que o `.dcode/config.toml` do A define (modelo, sandbox, regras, limites, `done.file`, `verify.command`) e ignora o do B.
   - Os comandos `/` também são lidos por workspace, no cliente (`tui.go:75`).
   - A regra "um workspace só" não está em `docs/specs/`. Ela está no handoff da TUI (`refs/design/HANDOFF.md:117-119`) e no código do cliente: a coluna filtra por workspace (`tui.go:160,249-256` → `internal/session/browse.go:80-82`), e toda sessão da TUI nasce no workspace dela (`tui.go:128`, `program.go:1628-1642`). A spec só diz "conversas gravadas deste workspace" (`client-tui.p.spec.md:539`).

3. **Os estados `loop` e `interrupted` não existem.**
   - O protocolo tem `idle | running | blocked | closed` (`protocol.go:151-156`).
   - `interrupted` é o motivo do último turno (`TurnCompleted.Reason`, `protocol.go:481-496`), com a sessão de volta a `idle` (`session.go:321-324`).
   - `loop` não tem representação: `Describe` não carrega `LoopSpec` (`session.go:141-160`). O único sinal é `done_criteria > 0` (`protocol.go:206-213`), que também vale para qualquer sessão comum num workspace com `.dcode/done.toml` (`done.go:167-168`).
   - `closed` fica de fora do design, e sessão fechada sai da lista (`session.go:720-740`).

4. **No fio, o selo não é `verified | unverified | not_verified`.**
   - `Completion.Verification` carrega `clean | passed | failed | stale | unavailable` (`protocol.go:675-689`, `internal/loop/done.go:58-64`).
   - A TUI traduz: `passed` → verified, `failed` → NOT VERIFIED, `stale`/`unavailable` → unverified, `clean` ou sem `Completion` → nada (`internal/tui/render.go:1731-1747`).
   - A TUI pinta NOT VERIFIED com `StyleDanger` (`render.go:1737`). Isso contraria o próprio comentário de `style.go:57` ("full-access, and nothing else") e a regra do handoff de que `danger` é só do full-access (`README.md:337-339`). O `!! NOT verified !!` do `LOOP.md` herda essa contradição.

5. **`tui.activity_verbs` não existe, e escrever essa chave impede a inicialização.**
   - Não há nenhuma chave `tui.*` em `KnownKeys` (`toml.go:27-91`), e chave desconhecida é erro (`toml.go:160-163`).
   - O que existe é `DCODE_ACTIVITY_VERBS`, só liga/desliga e só pelo ambiente (`activity.go:76-92`, `tui.go:203`).
   - O verbo não gira entre `delegando/lendo/conferindo/pensando`. Gira entre sinônimos da fase da ferramenta que está rodando (`activity.go:35-52,101-116`) e nunca aparece sozinho.
   - Sem ferramenta rodando, a linha diz a palavra fixa "trabalhando" (`activity.go:8-12`, `render.go:1162-1176`, `lang.go:574`). "Pensando" não existe.

6. **Aprovação: `↵` quer dizer o contrário na TUI.**
   - No design, `1 Permitir uma vez ↵` vem pré-selecionado. Na TUI, `enter` **nega** (`program.go:1509-1511`). Isso é invariante da spec: "aprovação pendente bloqueia a entrada e tem negar como default" (`client-tui.p.spec.md:624`).
   - As teclas da TUI são `a`, `A`, `P`, `G` e `d` (`program.go:1491-1516`).
   - `1/2/3` com o composer piscando são teclas sem modificador numa linha em que se digita, o que a RN-16 veta (`client-tui.p.spec.md:341-343`). A TUI resolve isso deixando o modal com o teclado (`program.go:765-766`).
   - O protocolo tem cinco respostas, não três: também `allow_project` e `allow_always` (`protocol.go:301-312`). A TUI oferece essas duas só para rede (`render.go:1465-1471`).

7. **"Ou diga o que fazer em vez disso" não tem caminho.**
   - Durante a aprovação a sessão está `blocked`. `steer` é recusado (`internal/session/steer.go:30-33`), `turns` também (`session.go:286-290`), e a resposta à aprovação só carrega a decisão (`protocol.go:345-348`).
   - O cliente conseguiria compor negar e depois dirigir quando a sessão voltar a `running`, mas não é o que o placeholder promete.

8. **Mensagem durante o turno dirige o turno, não entra na fila.**
   - A fila é do cliente por decisão do protocolo (`session.go:275-279`, `client-tui.p.spec.md:483-494`).
   - A regra da TUI, porém, é outra: "palavra digitada durante turno ativo **dirige** o turno" (`client-tui.p.spec.md:629`; `program.go:1249-1262` → `POST …/steer`, `server.go:181`).
   - Só vai para a fila a mensagem com imagem anexada ou o comando embutido que custa um turno (`program.go:1263-1273`, `client-tui.p.spec.md:630-631`).
   - *Mensagem entra na fila depois deste turno* descreve o contrário do que a TUI faz na mesma sessão.

9. **O relógio de aprovação não sabe quando a pergunta expira.**
   - "Desde" existe: é o `At` do envelope do evento (`protocol.go:136-146`, `internal/session/eventlog.go:104-110`).
   - O `expires_at`, porém, sai **zerado**. O evento é emitido antes de o prazo ser posto (`internal/loop/turn.go:937-947`), e o prazo só existe na cópia guardada pela sessão (`session.go:545-547`). Nenhuma rota expõe essa cópia: `Pending()` (`session.go:634-644`) não é usado fora de testes.
   - Passados 2 min, a pergunta é negada sozinha (`serve.go:32`, `daemon.go:82-84`, `session.go:588-598`). Um relógio que só sobe esconde isso, inclusive nas sessões de fundo marcadas `aprovar` na lateral.
   - "Recursos pedidos" não é uma lista. Vêm uma fronteira (`network`, `rule:write`, `filesystem_write`…: `internal/policy/policy.go:93-106`), um motivo genérico (`policy.go:202-248`), a regra e o comando. Os caminhos só estão no `input` do `tool.requested` com o mesmo `tool_call_id` (`protocol.go:566-584`).

10. **O card de delegação pede três coisas que não trafegam.**
    - **Nome do filho:** a entrada do `explore` é só `task`, `path` e `owns` (`explore.go:52-67`). A TUI usa o `path` como nome (`model.go:859-874`, `delegation.go:84-87`).
    - **Progresso e "charlie lendo X":** os passos do filho não são eventos da sessão (`delegate.go:225`, `Emitter: nil`).
    - **Leituras e escritas:** só chega `files = len(read)`, no fim (`explore.go:175-179`). As listas de lidos e escritos vão como prosa no `Output` (`explore.go:158-168`), que o protocolo proíbe parsear (`protocol.go:597-601`).
    - A TUI detecta a delegação por duas ou mais chamadas `explore` adjacentes (`delegation.go:21-41`).

11. **Não há contagem de tokens ao vivo.**
    - O uso só chega no `turn.completed` (`protocol.go:659-665`).
    - Durante o turno há `progress` (rodadas, chamadas em voo, bytes de argumento, arquivos: `protocol.go:498-520`) e `context.band`, sem tokens.
    - O `1m42s · 18.4k` copia a linha da TUI (`render.go:1177-1184`). Lá, o número mostrado durante o turno é o `OutputTokens` do turno **anterior**: só é escrito no fim (`model.go:638`), e o `turn.started` não o zera (`model.go:339-356`).

12. **Sessões antigas e a prévia do `⌘K` vêm do disco, não do log.**
    - `GET /v1/sessions` lista só as sessões vivas (`server.go:268-270`, `session.go:706-718`), sem título, nome, turnos, selo, diff ou último evento (`protocol.go:169-214`). Os eventos de sessão não viva não são servidos (`server.go:470-475`).
    - O que `2h`, `ontem` e a contagem de turnos precisam existe em `session.Summary`, lido do disco (`browse.go:25-40,59-92`). Isso contradiz "nunca de polling do disco" (`README.md:289-290`).
    - Renomear sessão viva escreve direto no arquivo (`server.go:239-266`, `internal/session/rename.go:84-108`), fora do log em memória (`eventlog.go:102-110`). Os clientes conectados não recebem o `session.renamed`, e o próximo evento vivo pode repetir o `seq` no registro. `internal/server/rename_test.go` só cobre sessão não carregada.
    - Um defeito da TUI que o desktop não deve repetir: a coluna lista sessões **gravadas** (`tui.go:160,249-270`), mas `enter` nela chama `GetSession` (`program.go:873,1710-1717`), que só responde por sessões **vivas** (`server.go:272-279`, `session.go:680-689`). O teste usa um transporte falso que sempre acha a sessão (`internal/tui/program_test.go:104-109`). Abrir uma conversa antiga é `CreateSession{Resume}`: sessão nova, com outro id (`daemon.go:155-202`).

13. **Trocar de modelo não continua a sessão.**
    - `/model` abre sessão nova **vazia**, com `CreateSessionRequest` sem `Resume` (`program.go:1628-1642`; `client-tui.p.spec.md:441`).
    - O protocolo permitiria continuar: `Resume` junto com `Model`, com o modelo pedido vencendo (`daemon.go:152-202`), desde que haja registro gravado (`serve.go:89-101`). Nenhum cliente faz isso.
    - O `LOOP.md:119-120` afirma o contrário. A nota dos contratos de cada modelo também não trafega.

14. **Chip `workspace-write · on-request`: a política de aprovação não está no fio.**
    - A sessão traz só `sandbox_mode` e `mode` (`protocol.go:190-194`). A política só se deduz pelo mapa de modos (`session.go:253-266`), e o protocolo diz que copiar esse mapa no cliente é justamente o que deriva (`protocol.go:770-775`).
    - A criação aceita `sandbox_mode`, não o modo (`protocol.go:217-220`, `policy.go:33-40`).

15. **O `LOOP.md` descreve como novo um loop que já existe dentro do turno.**
    - "Uma sessão comum para quando o modelo responde" (`LOOP.md:17-18`) só é verdade quando não há definição de pronto.
    - Com `.dcode/done.toml` ou `verify.command` (`done.go:29-85,160-169`), **toda** sessão já reentra quando o modelo para. Ela roda os critérios, devolve ao modelo a saída do que falhou e segue enquanto o conjunto de não cumpridos encolhe (`turn.go:424-472,1182-1276`; RN-10 em `agent-loop.r.spec.md:72-81`).
    - O que o `LOOP.md` muda é a escala: a iteração vira um turno inteiro, há orçamento de tempo e tokens, e a falta de progresso pausa em vez de encerrar. O princípio já está lá.

16. **O `/loop` de hoje é outro comando.** A tela do loop vai sentar nisto:
    - **`/loop <pasta>`** abre **sessão nova, sempre** (`loop-command.r.spec.md:87`; `program.go:1401-1408`), com `LoopSpec` (`program.go:1867-1881`), e submete o turno logo em seguida (`program.go:523-545`).
      - O daemon lê `<pasta>/done.toml`, que vence o `tasks.md` (`loopspec.go:87-106`, `loopcommand/dispatch.go:196-240`). O formato é o mesmo do `.dcode/done.toml`: um critério por `[seção]` com `command` e `exit_code` opcional, e `protected` no topo.
      - Sem `done.toml`, ele lê as linhas `- [ ] N. … verify: \`cmd\`` do `tasks.md` (`loopspec.go:62-69,107-124`).
      - `done.toml` vazio é erro, não "pronto" (`dispatch.go:209-215`).
    - **`/loop <frase>`** mede as pastas de spec (`GET /v1/specs`) e trabalha as pendentes uma por vez, uma sessão para cada (`program.go:474-509,1917-1936`).
      - Sem pasta nenhuma, ele **qualifica**: uma sessão em plan mode propõe critérios, o daemon mede e escreve `.dcode/done.toml`, e o laço para para a pessoa ler (`commands.go:453-481`, `daemon.go:483-514`, `done.go:251-283`, `program.go:510-521`).
      - A partir daí, toda sessão comum do workspace é medida por esse arquivo.
    - **A sequência vive no processo da TUI** (`program.go:161-166`). Outro cliente não vê que há um loop nem em que spec ele está.
    - **Única flag: `--protect`.** Qualquer outra é recusada (`commands.go:349-383`). `--max`, `--tokens`, `--tempo`, `--parado` e `--checks` não existem.
    - **`B4` inverte uma decisão escrita:** sem critério, o `/loop` qualifica em vez de recusar (`loop-command.r.spec.md:179,196`).
    - **`dcode loop` não existe** (`main.go:35-57`). Hoje, `dcode loop "x" --max 50` cai no one-shot e roda um turno cuja tarefa é literalmente `loop x --max 50` (`main.go:59-72`, `once.go:19-31,81-91`), porque o pacote `flag` para no primeiro argumento que não é flag.

17. **Regras e controles do loop contra o motor.**
    - Sem progresso, o turno **encerra** como `incomplete`, não pausa (`turn.go:1257-1262`; `agent-loop.r.spec.md:81`). Isso acontece depois de 2 ciclos por padrão, não 3 (`app.go:322`, `turn.go:1318-1323`).
    - Um ciclo que regride é desfeito sozinho (`turn.go:1250-1253`).
    - Os tetos são dois:
      - `max_iterations` conta rodadas do turno, com o padrão da família ou 50 (`limits.go:26-29,47-53`).
      - `max_turn_tokens` conta tokens de **saída**, e 0 é sem teto (`limits.go:33-37`, `turn.go:518-520`).
    - Não há teto de tempo.
    - Um critério só roda se a sessão escreveu algo (`turn.go:1189-1193`).
    - "Continuar em loop" a partir de um selo não pode converter a mesma sessão, porque a definição de pronto é fixada no nascimento (`done.go:154-159`). Seria `CreateSession{Resume, LoopSpec}`, com id novo.
    - "Parar depois desta iteração" não tem rota (`server.go:166-186`).
    - `undo` é recusado com turno rodando (`session.go:426-429`) e desfaz o turno inteiro, não um ciclo. Desfazer ciclo é só interno (`tools/undo.go:64-87`).
    - Aprovação dentro do loop não espera por você: nega em 2 min (item 9).

18. **`↻ 8/100` e "iteração N" não têm fonte.**
    - `progress{rounds}` conta **toda** rodada do turno, inclusive as de ferramenta, e só aparece depois da primeira volta (`turn.go:467,514,1082-1100`).
    - Não há evento por ciclo. O relatório de cada ciclo fica na memória do motor, e só o `Completion` final trafega (`loop/done.go:86-105`, `protocol.go:675-689`).
    - A lista de critérios da sessão também não trafega: `Session` traz só a contagem (`protocol.go:206-213`), e os nomes só aparecem no `Completion` no fim do turno.
    - O painel "Pronto quando" só teria de onde ler antes disso se o cliente abrisse o arquivo, o que o protocolo evita de propósito (`protocol.go:229-235`).

19. **Teclas contra as convenções da TUI.**
    - `esc` interrompe no desenho. Na TUI, quem interrompe é `^C` (`program.go:916-921`, `lang.go:575`). Lá, `esc` com a linha vazia entra no fluxo (`program.go:1127-1143`), e na aprovação `esc` nega.
    - No Linux, com `⌘` virando `Ctrl`, três atalhos caem em teclas já usadas:
      - `⌘K` vira `^K`, que apaga até o fim da linha (`program.go:1091`).
      - `⌘N` vira `^N`, que desce na lista (`picker.go:184`).
      - `⌘Z` vira `^Z`, que suspende o processo, já apontado no precedente.
    - `⌘1…⌘9` e `⌘.` não colidem com nada.

20. **Outros fatos do handoff.**
    - O `README.md:23-27` manda escolher Wails ou Tauri; o `BRIEF.md:30-32` já decidiu Electron.
    - "Rotinas" (`◷`) e usuário com avatar e nome não existem no produto. O daemon não tem conta nem autenticação (`server.go:7-9`). Nenhum dos dois está no `SPEC_GAPS.md`.

**O `BRIEF.md` também afirmou ao designer três coisas que o código não sustenta:**

- `done.proposed`, `done.signed` e `SignDoneRequest` estão declarados (`protocol.go:42-51,376-420`), mas nada emite esses eventos e não há rota de assinatura (`server.go:166-186`).
- "Criar sessão (… modo …)": a criação aceita `sandbox_mode`, não o modo (item 14).
- "Trocar de modelo … carregando o histórico" (item 13) e "`tool.approval_required` … quando expira" (item 9).

**Lacunas do `SPEC_GAPS.md`:**

- **A1. Existe em parte.**
  - Existe: protocolo e transporte, que servem a qualquer cliente (HTTP+SSE em socket Unix: `server.go:1-12,166-186`; cliente em `pkg/client/client.go`).
  - Falta: a família `client-desktop` (não existe em `docs/specs/architecture/`), a decisão de framework (item 20) e, sobretudo, quem sobe o daemon (item 1).
- **A2. Existe em parte.**
  - Existe: sessão por workspace.
  - Falta: configuração por workspace (item 2).
  - "Um workspace só" é regra do cliente TUI, não do daemon.
- **A3. Já existe no daemon** (`session.go:16-18,651-678`; `server.go:527-530`).
  - Ressalvas: as sessões morrem com o daemon que as criou (`server.go:147-154`), e a aprovação de sessão de fundo nega sozinha em 2 min (`session.go:588-598`).
- **A4. Não depende do daemon**, que não conhece projeto (`protocol.go:169-214`).
  - Renomear **sessão** existe (`server.go:176`), com o defeito do item 12 quando a sessão está viva.
- **A5. Existe em parte.**
  - Existe: o teclado confere com a TUI (`railnav.go:61-74,153-165`), e `⌘1…⌘9` não colide.
  - Falta: a prévia de sessão viva (título, turnos, selo, último evento) não tem fonte. A de sessão gravada só existe no disco (`browse.go:25-40`).
- **A6. Existe em parte.**
  - Existe: as contagens saem do `state` de `GET /v1/sessions` (`server.go:268-270`).
  - Falta: só cobre sessões vivas e só por consulta. O SSE é por sessão (`server.go:177,470-533`), e não há fluxo de mudanças da lista.
- **A7. Já existe, com mais opções:** são cinco decisões (`protocol.go:296-312`).
  - A decidir: quais mostrar e o default do `↵` (item 6).
- **A8. Já existe como fila do cliente** (`session.go:275-279`; `model.go:228`; drenagem em `program.go:627`).
  - Conflita com a regra de dirigir o turno (`client-tui.p.spec.md:629`; item 8).
- **A9. Confirmado ausente.** A TUI tem uma paleta só, as 16 cores ANSI do terminal, sem fundo próprio (`internal/tui/theme.go:8-24`). Não há tokens a herdar.
- **B1. Existe em parte, dentro do turno.**
  - Existe: reentrada por critério, com saída por progresso (`turn.go:424-472,1182-1276`).
  - Falta: a iteração como unidade (turno inteiro), com eventos e orçamento próprios.
- **B2. Existe em parte.**
  - Existe: `/loop`, só na TUI e com outra semântica (item 16).
  - Não existem: `⌘⇧↵`, "Continuar em loop" e `dcode loop`.
- **B3. Confirmado ausente como flags:** só existe `--protect` (`commands.go:356-371`).
  - Análogos existem só na configuração e valem por turno: `limits.max_iterations`, `limits.max_turn_tokens`, `limits.max_stall_cycles` (`toml.go:37-38,70`).
  - Tokens dos filhos contando para o pai já existe (`delegate.go:165-169`).
- **B4. Conflita com decisão escrita.**
  - Sem critério, o `/loop` qualifica em vez de recusar (`loop-command.r.spec.md:179,196`; `program.go:2035-2070`).
  - Sessão comum sem critério termina `done` (`turn.go:1183-1184`) e diz `done_criteria: 0` (`protocol.go:206-213`).
- **B5. Existe em parte.**
  - Regra 1 existe: `done` com `passed` (`turn.go:1203-1227`, `loop/done.go:243-270`).
  - Regra 2 diverge: encerra `incomplete` em 2 ciclos, não pausa (`turn.go:1257-1262`).
  - Regra 3 existe em parte: há `max_iterations` e `max_tokens`, não há tempo (`turn.go:513-520`).
- **B6. Já existe, na granularidade de critério.** Progresso é o conjunto de não cumpridos encolher sem nada novo falhar (`loop/done.go:171-234`; `agent-loop.r.spec.md:77`). "Itens de um check" não existe (ver C1).
- **B7. Existe em parte.**
  - Existe: `steer` entrega a mensagem no topo da **próxima rodada** (`turn.go:350-353`), não da próxima iteração.
  - O que sobra quando o turno acaba é descartado com aviso (`session.go:310-320`).
- **B8. Existe em parte.**
  - Existem: interromper (`server.go:180`) e `undo` do último turno com filhos (`undo.go:200-216`).
  - Falta: parar depois da iteração.
  - O `undo` é recusado com turno rodando (`session.go:426-429`).
- **B9. Existe em parte.**
  - Existe: o turno bloqueia na aprovação (`session.go:550`).
  - Não há relógio de loop para parar, e a espera tem prazo que nega (`session.go:588-598`).
- **C1. Confirmado ausente, e contra decisão escrita.**
  - Critério é código de saída (`loop/done.go:20-24,305-308`; `agent-loop.r.spec.md:75`). O protocolo registra por que não há contagem de testes (`protocol.go:522-528`).
  - Isso é mudança de spec do `agent-loop`, não campo novo.
- **C2. Confirmado ausente.** Não há nada entre `progress{rounds}` e `turn.completed` (`turn.go:1082-1100`, `protocol.go:658-673`).
- **C3. Existe em parte.**
  - Existe: `protocol.Session` tem estado, workspace, branch, modelo, modo, criação e `last_seq` (`protocol.go:169-214`).
  - Falta: título/nome, diff, selo, turnos e última atividade.
  - O último evento dá para ler abrindo o SSE em `from=last_seq` (`server.go:477-485`), mas é um stream por sessão.
- **C4. Existe em parte.**
  - Existe: "desde" é o `Event.At` (`protocol.go:136-146`).
  - O prazo está declarado mas sai zerado (`turn.go:937-947` contra `session.go:545-547`).
  - Os recursos vêm como uma fronteira, uma regra e um comando. Os caminhos só estão no `tool.requested` (`protocol.go:358-374,566-584`).

**Decisões antes de o desktop falar com o daemon** (não são polimento de design):

1. **Quem sobe o daemon e onde.** O desktop adota `dcode serve` no socket padrão, e a TUI passa a anexar a ele, ou embute o próprio, e aí nada é compartilhado. Isso inclui o que acontece com sessões rodando quando o app fecha (`server.go:147-154`) e como o app acha o socket sem o ambiente do shell (`daemon.go:50-62`).
2. **Um daemon para todos os projetos exige configuração por sessão.** Seria resolver a cadeia de configuração no workspace de cada `CreateSession`, em vez de usar o `Base` de boot (`serve.go:42-49`, `daemon.go:143`). A alternativa é um daemon por projeto.
3. **De onde vem a lista de sessões.**
   - Opções: uma rota para sessões gravadas com resumo agregado (C3), ou o desktop lê o disco como a TUI.
   - Também é preciso decidir como observar o estado de N sessões sem abrir N streams.
   - E a semântica de abrir conversa antiga: `GetSession`, só para sessão viva, contra `CreateSession{Resume}`, que cria uma nova.
4. **Aprovação.** Decidir o default do `↵` (a spec da TUI diz negar), se a entrada fica bloqueada, quantas das cinco decisões aparecem, e preencher `expires_at` no evento.
5. **Mensagem durante o turno.** Dirigir (regra da TUI) ou enfileirar: a mesma sessão não pode ter uma regra diferente em cada cliente. Decidir também se "diga o que fazer em vez disso" vira negar e depois dirigir, no cliente, ou uma rota nova.
6. **O que é o loop.**
   - Uma tela sobre o que já existe (reentrada no turno mais `/loop` de pasta ou frase), ou um modo novo, com a iteração como unidade e eventos próprios (C2).
   - Nos dois casos, a orquestração do `/loop` precisa sair do processo da TUI (`program.go:161-166`) para os dois clientes verem o mesmo loop.
   - A contagem por check (C1) é decisão de spec do `agent-loop`, não campo novo no protocolo.
7. **Tokens ao vivo.** Um evento de uso por rodada, ou a linha de atividade mostra só o tempo. A TUI também precisa parar de mostrar o número do turno anterior.
