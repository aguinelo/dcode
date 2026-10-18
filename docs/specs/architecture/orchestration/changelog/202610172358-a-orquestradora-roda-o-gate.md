# A orquestradora roda o gate ela mesma

**Data:** 2026-10-17
**Specs afetadas:** `202610132000-orchestration` (`.p` §7.4, §8, §13, §14; `.config` §3; `.i`, passo 9).

## O que mudou

O núcleo do passo 9: `internal/orchestrate/gate.go`. `Gate.Measure(ctx, dir, commit)` roda o gate
declarado num commit exato e devolve o `GateResult` (commit, árvore, comando, saída de fim, duração,
`green`, `red`, `timeout` ou `undeclared`). `GateFrom` lê `orchestrator.gate_command`, com
`verify.command` como queda, e `orchestrator.gate_timeout` (padrão `30m`, de `1m` a `6h`), que
entram em `KnownKeys`. `RecordGate` escreve `gate.measured` no log de decisões. `CheckFresh` é a
trava do merge: medição de outro commit ou sobre outra main é `gate_stale`.

É o gate da era humana (`.githooks/pre-push`) trazido para dentro: worktree descartável no commit,
`GIT_*` fora do ambiente, a árvore removida no fim. A seleção de gate pelo diff contra a main não
vem: a orquestradora roda o comando que o projeto declara.

## O que o código decide onde a `.p` deixava a forma em aberto

- **Worktree descartável, não o lugar de integração.** O gate roda numa worktree destacada criada
  em `H` só para aquela medição, num diretório temporário, e removida ao voltar — no fim, no
  `timeout` e num pânico (`defer`, com contexto próprio, porque o do chamador pode ser o motivo da
  parada). Nunca a árvore da pessoa, nunca a worktree do filho, cujos arquivos podem não ser os do
  commit. A integração continua sendo onde `H` nasce (o merge da main); medir em outra árvore do
  mesmo repositório é o que garante que o medido é o commit, e não o que sobrou no disco.
- **Todo `GIT_*` sai do ambiente do comando**, não só os quatro de `--local-env-vars`: um gate
  disparado por um hook encontraria o repositório do hook em vez da árvore em que está.
- **Só SHA completo.** Ref ou abreviação nomeia o que aponta quando o git lê, e a medição é de um
  commit; o resto é `invalid_input`.
- **`Base` é de quem chama.** `Measure` sabe o commit, não qual main está sob ele; a esteira, que
  fez `H` pelo merge da main, preenche. `CheckFresh` recusa medição sem `Base`.
- **A saída inteira vai para `gates/<commit>.log`**, não `<child-id>-<commit>.log`: o `Measure`
  conhece o commit, não o filho, e a saída de um commit é a mesma qualquer que seja o filho. A
  memória fica limitada aos últimos 4 KiB, cortados num caractere inteiro.
- **Tempo-limite mata o grupo de processos** (`Setpgid`), porque `make check` começa compiladores
  e binários de teste que um kill no `sh` deixaria vivos. Gate parado pelo chamador não é medição:
  é erro, e só gate terminado conta (§8).
- **A alegação do filho é dado, nunca prova.** `RecordGate` guarda o que o filho disse em `claim`,
  ao lado do `measured`; o `result` é o medido. O `why` é do harness e diz o resultado e o commit.
- **O sandbox do `bash` em volta do gate fica para a fiação.** A `.p` dizia que o gate roda pelo
  sandbox de sessão, gravável só a integração; a worktree descartável já mantém a pessoa e o filho
  intocados, e embrulhar o comando no sandbox é decisão do passo que liga o gate à esteira.

## Invariantes e o que fica

Sobem para a §13 cinco linhas com teste: a worktree descartável sem `GIT_*`, a limpeza até num
pânico, a alegação como `claim`, o `undeclared` e o `gate_stale`. Na §14 fica uma linha só: sem gate
verde não há PR nem merge, e o que andou mede de novo — isso é da esteira e do passo 10.

A chamada na esteira de cada filho (depois da revisão, antes da fila de merge), o lugar de
integração e o merge da main ficam para quando os passos 6 a 8 chegarem. Até lá `GateFrom`,
`RecordGate` e `CheckFresh` não têm chamador fora dos testes e ficam dispensados em
`internal/specguard/unused_orchestrate_test.go`, como os nomes dos passos 2 e 4.
