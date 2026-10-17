# A sessão gravada reabre sob o próprio id

Passo 3 da `.i`. Continuar uma conversa gravada (`CreateSessionRequest.Resume`) abre uma sessão
**nova** que a reproduz — certo para quem digita `dcode -c`, errado para o que outros guardam pelo
id. A orquestradora é nomeada no `state.json` do projeto e no `parent_id` de cada filho; um
reinício do daemon que a trouxesse sob outro id órfã todos eles. Agora `session.Manager.Reopen`
traz a sessão de volta com o mesmo id, o histórico, a sequência e o arquivo.

## O que entrou

- **`Manager.Reopen(dir, id, retention, clock, build)`.** Lê o registro (`ReadReopening`), abre o
  log com o `Seq` continuando do último evento gravado e o arquivo em acréscimo
  (`Reopening.Log`, pelo `NewRecord` de hoje), e chama o `ReopenBuilder` do daemon, que monta o
  motor com o histórico de `Rebuild` e a sessão com `NewReopened`. O id e o log devolvidos são
  conferidos: um construtor que monta outra sessão é recusado.
- **O que volta é o que o registro diz que a sessão era.** `Reopening.Origin` é o
  `session.created` — `protocol.Session`, o tipo do fio —, com o modo do último
  `session.mode_changed`; `Title` é o último `session.renamed`. Por ser o tipo do fio, o
  `parent_id`, `project_id`, `role` e `owns` que o passo 2 (#518) pôs na `Session` voltam junto,
  sem segunda lista aqui; a `session.Session` ganha os quatro para os devolver no `Describe`, e um
  teste confere que a sessão reaberta continua sob a mesma mãe.
- **Turno aberto ganha fim.** O turno que o registro mostra começado e não terminado recebe
  `turn.completed{reason: daemon_restart}` antes de a sessão entrar no `Manager`, como primeiro
  evento novo; ninguém começa um turno à frente dele. O motor reaberto numera o próximo turno
  depois do último gravado (`loop.Engine.ContinueTurns`): o mesmo id com `t1` de novo daria dois
  turnos de um log com o mesmo nome, e o id de aprovação é feito dele.
- **`SubmitOrigin(text, origin)`.** O turno nasce com `TurnStarted.Origin`; vazio é a pessoa,
  `children` é o harness. Origem desconhecida é `invalid_input` antes do turno.
- **Aprovador sem prazo.** `session.Approver` substitui o `deadlineApprover` do `app` — o mesmo
  comportamento com o tempo do daemon, que nunca é zero ali — e, com tempo zero, anuncia
  `ExpiresAt` zero: a pergunta espera a pessoa o quanto for, para o filho da orquestradora.
- **No fio**, só os três itens do passo 2 de que este depende, combinados com quem fez o passo 2:
  `TurnStarted.Origin` (`omitempty`), `OriginChildren` e `StopDaemonRestart`. `generated.ts`
  regenerado, e o texto deles na `client-server-protocol.p` §5.1.

## O que é recusado, e como é dito

| Caso | Resposta |
|---|---|
| sessão viva, ou já sendo reaberta | `invalid_input`: duas sessões escrevendo um registro são um registro com duas sequências |
| registro ausente (nunca gravado, ou podado) | `session_not_found`, e **nada abre** — nunca uma sessão vazia sob o id antigo |
| gravação desligada | `session_not_found`, dizendo que não há gravação |
| linha ilegível no meio, sem `session.created`, linha de outra sessão | `internal`, com o número da linha |
| id que não é id | `invalid_input`, antes de virar caminho |

A **última** linha cortada é tolerada: é o que um crash no meio de uma escrita deixa. Ela ganha um
`\n` antes do primeiro evento novo, que fica numa linha própria em vez de colado ao lixo.

## Fora deste passo

- **Ninguém chama `Reopen` ainda.** O `app.Daemon` implementa a `Spawner` (e o `Reopen` dela) no
  passo 5, com o `Registry.Reload` no boot. Rota de reabrir não há: a `.p` não nomeia uma.
- **A poda.** `session.Prune` roda a cada sessão aberta e poupa só as vivas; o registro da
  orquestradora de um projeto não pode sumir sob ela. Poupar os registros de papel `orchestrator`
  (e dos filhos não terminais) é do passo 5, que é quem sabe quais são; aqui, registro podado é
  `session_not_found`, dito, e o teste disso existe.
- Aprovação pendente no registro de um turno que caiu não é respondida: o turno terminou
  (`daemon_restart`), e a pergunta morreu com ele.
