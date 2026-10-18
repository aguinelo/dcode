# Os cenários roteirizados

**Data:** 2026-10-18
**Specs afetadas:** `202610132000-orchestration` (`.p` §13, §14; `.i`, passos 9 e 12).

## O que mudou

O passo 12: os cenários da RN-11, de graça e determinísticos, em todo PR. Cada um abre um daemon
de verdade, em processo, com `orchestrator.enabled`, configurado pela cadeia real (ambiente e
projeto). Um modelo roteirizado, por HTTP e no formato de streaming de verdade, responde à
orquestradora, a cada filho e a cada revisor pelo roteiro de cada um. Cada cenário tem o próprio
repositório de brinquedo, com um `gate.sh` que fica vermelho quando algum arquivo diz `BROKEN`.
Quando há forge, entram um `origin` bare e um `gh` falso no `PATH`. O cenário só confere o log de
decisões, lido pelo `LogReader` do `dcode decisions`, as rotas que o cliente da pessoa usa e o git.
O que o modelo diz nunca é conferido.

Os doze da `.i` passam com `-race -count=3`, cada um em 1 a 6 s; os treze juntos, em ~35 s
com `-race`. Veio mais um, `TestScenarioEveryEndOfAChildsTurnIsNamed`. As duas invariantes que não
são cenário ganharam teste em `internal/orchestrate/`: a ordem log → fio, e a transição que não
acontece quando a linha não grava (`transition_test.go`); e a guarda de fonte contra `rebase`,
`push --force` e `--amend` em `internal/orchestrate/` e `internal/vcs/` (`history_guard_test.go`),
que o passo 9 pedia. A §14 da `.p` fica vazia: nenhuma invariante segue prevista.

## O que o código decide onde a `.i` deixava a forma em aberto

- **Em `internal/app/`, não num `scenario_test.go` do `orchestrate`.** O daemon que liga a
  esteira às sessões, às rotas e à configuração mora no `app`. Os ajudantes de teste que o abrem
  em processo também estão lá. Um cenário em `orchestrate` testaria um daemon de mentira, que é
  exatamente o que estes não fazem.
- **Um modelo próprio, não o `scriptedModel`.** A orquestradora, os filhos e os revisores falam ao
  mesmo tempo. Cada pedido vai ao roteiro de quem o fez: o papel sai das ferramentas oferecidas
  (`delegate` é da orquestradora, `verdict` é do revisor), e o filho sai de uma marca `[[chave]]`
  na tarefa. Um pedido além do roteiro reprova o cenário: um filho que pediu mais do que o cenário
  escreveu foi parar onde ninguém quis.
- **Um `gh` falso próprio.** O de `internal/orchestrate/testdata/fakegh` responde de arquivos que
  o teste escreve antes, e um cenário não sabe de antemão qual commit o squash vai gerar. O de
  `internal/app/testdata/scenariogh/gh` faz o squash de verdade no `origin` bare. Ele recusa
  `--match-head-commit` quando a branch no `origin` não está nele, recusa um segundo PR aberto para
  a mesma branch e responde ao `pr edit`. Com `red-after-merge`, deixa o CI vermelho nos commits
  que ele mesmo fez.
- **Cenário que não pode rodar reprova.** Sem `git`, `sh`, `grep` ou sandbox, o cenário falha com
  o motivo: pular contaria como passar, sem ter provado nada.
- **A pessoa é a das rotas.** O merge sai de `POST /v1/sessions/{id}/questions/{qid}`, a aprovação
  de `POST /v1/sessions/{id}/approvals/{aid}` e o fechar de `DELETE /v1/sessions/{id}`.

## Defeitos que os cenários acharam, consertados no passo 7 (#534) antes do merge

- Com forge `github`, a esteira media contra a `main` local (`mainTip` em `pipeline_gate.go`), que
  um merge no GitHub nunca move. Depois do primeiro merge, todo filho seguinte ficava em
  `gate_stale` para sempre, e um conflito com o que entrou pelo `origin` nunca era visto.
- Cada nova medição abria um PR novo para a mesma branch (`Merges.Propose`), coisa que o GitHub
  recusa. Agora o PR existente é atualizado (`pr.updated`).
- `child.returned` saía em dois formatos, conforme quem escrevia: o `Return` solto no conflito
  (`integrate.go`), e `{return, returned}` na esteira. Ficou um formato só. O leitor dos cenários
  reprova uma linha sem `return`.
