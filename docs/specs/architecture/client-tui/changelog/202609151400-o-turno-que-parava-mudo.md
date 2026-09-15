# O turno que parava mudo

**Data:** 2026-09-15
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 6)
**Fonte:** relato do usuário, com print — uma tarefa de código rodando contra
um modelo local (Qwen, via `--family generic`) "ficou assim": nenhum erro na
tela, só o contador `round 50/50` congelado. Perguntar "como está" parecia
retomar, mas a tarefa nunca concluía.

## O que mudou

`internal/tui/model.go`, tratamento de `EventTurnCompleted`: quando
`completionEntry(d.Completion, ...)` não produz nada (não havia critério de
conclusão pra relatar) **e** `d.Reason` é `max_iterations`, `repeat_loop` ou
`max_tokens`, uma entrada (`KindNote`) aparece dizendo qual foi o motivo —
com os números de rodada, quando o motivo é o teto de iterações.

## Por que ficava mudo

`protocol.TurnCompleted.Reason` existe desde a primeira versão deste evento.
Nunca tinha sido lido em `internal/tui` — busquei em todo o pacote e a única
leitura de qualquer `Stop*` era a definição das próprias constantes, em
`internal/protocol`. Um turno que termina normalmente não precisa dizer
nada: a última mensagem do modelo já É a resposta visível. Um turno cortado
no meio — teto de rodadas, repetição, teto de tokens — termina exatamente
tão abruptamente quanto um erro terminaria, e era reportado exatamente tão
alto quanto um: nada.

O print que originou isto mostrava a prova dupla: `round 50/50` congelado na
barra, e logo acima "19 earlier messages were summarised; 98 kept" — a
compactação automática tinha rodado certinho. Não era contexto. Era o teto
de 50 rodadas da família `generic`, batido no meio de uma tarefa real, e o
motivo nunca chegando à tela.

## O que não ganhou nota

`done` não ganha nada — é o caminho comum, e o texto do modelo já é a
resposta. `interrupted` não ganha nada — a pessoa acabou de apertar uma
tecla, e uma segunda nota pro que ela mesma fez é ruído. `error` não ganha
nada — os caminhos que produzem esse motivo já emitem `EventSessionError`
com a mensagem de verdade, e repetir aqui só repetiria pior. `unverified` e
`incomplete` normalmente já carregam um `Completion` (vêm do ciclo de
critérios de conclusão, que povoa `lastReport`), então já passam pelo
caminho existente — mas se algum dia não carregarem, caem no mesmo `else`
que os três novos, por segurança.

## O que este relato não fecha

O teto de 50 rodadas em si — pequeno demais pro que a família `generic`
serve na prática — é outra mudança, registrada em
`docs/specs/architecture/provider-adapter/changelog/202609151500-generic-erra-para-o-horizonte-longo.md`.
Esta entrada é só sobre o silêncio; aquela é sobre o número.
