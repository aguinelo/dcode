# Resposta vazia não é fim de turno

**Data:** 2026-10-13
**Specs afetadas:** `202608072335-agent-loop` (`.r` — nova RN-12 e o passo 3 da RN-1; `.p` — §2.1, `StopDone` e invariantes), `202608072334-provider-adapter` (`.p` — nota sob o mapa de decisão)
**Fonte:** auditoria de "harness > modelo" — as mesmas premissas valem qualquer que seja o modelo

## O que mudou

Stream que termina limpo sem texto — espaço em branco não conta — e sem tool
call passa a ser falha do provedor: `provider.EmptyCompletion`, classe
`provider`, `Retryable`. O loop repete a mesma requisição dentro do orçamento
de repetição do turno e, persistindo, encerra em `StopError` emitindo
`session.error` com a mensagem `empty completion: …`.

## O defeito

Em `internal/loop/turn.go`, uma resposta sem chamada ia para o caminho
`len(calls) == 0`: verificação de pronto e, sem definição de pronto, `StopDone`.
Uma resposta **vazia** fazia o mesmo caminho. O turno terminava `done`, a tela
ficava vazia e nada dizia por quê.

A suíte de medição (`internal/evals/harness_eval_test.go`) já tinha visto o
caso e decidido o contrário: resposta vazia é falha de medir, repetida como
qualquer tropeço de transporte. Foi medido ali, à mão, com o corpo exato de um
cenário: a Gemini devolveu chamada duas vezes e nada três vezes em cinco; a
nota em `internal/evals/measured.go` registra dezessete repetições em vinte
rodadas. O mesmo modelo, no mesmo harness, era falha na medição e resposta no
produto.

## Por que falha do provedor e não resposta

Recusar agir custa tokens e deixa uma frase. Resposta vazia não deixa nada —
às vezes nem tokens, às vezes tokens gastos em frames sem conteúdo. Não há o
que o modelo tenha dito para o turno honrar.

E `done` é a única forma que uma falha não pode ter: um turno que parece
concluído não convida ninguém a olhar de novo. Erro com motivo convida.

## Por que dentro do orçamento existente

A classe `provider` já tem uma decisão no mapa (§4 do adaptador): repetir com
recuo exponencial, até o teto. Inventar um orçamento novo para este caso seria
uma segunda política de repetição para a mesma classe. O contador de tentativas
é o do turno, o mesmo que limites de taxa e quedas de conexão consomem.

## Uma linha, não duas

`EmptyCompletion` vive no adaptador e é chamada pelo loop e pela suíte de
medição; `evals.unreadable` passou a delegar a ela. A linha — "nada que um juiz
leia", não "zero tokens de saída" e muito menos "sem tool call" — tinha sido
traçada e justificada na suíte. Copiá-la no loop seria o começo de duas
definições que divergem.

## O que fica de fora

- Stream que só trouxe uma chamada recusada por schema não é vazio: produziu
  algo, já respondido no histórico. O loop marca esse caso e não o reclassifica.
- Stream cortado pela interrupção da pessoa continua sendo interrupção: com o
  `ctx` cancelado, a checagem não roda, e o caminho seguinte encerra em
  `StopInterrupted` como antes.
- Texto só de espaço em branco já tinha sido emitido ao cliente como
  `message.delta` quando a falha é declarada. Não volta atrás; também não entra
  no histórico.

## Testes que mudaram de roteiro

Dois testes do loop rodavam além do roteiro do provedor dublê e recebiam a
resposta vazia de reserva como se fosse uma resposta:
`TestChildTokensAreDebitedFromTheParent` e
`TestAfterATurnTheCallerIsToldWhatWasWritten`. O que eles conferem não muda;
ganharam a resposta que o turno deles de fato pressupõe.

## Como fica preso

`TestAnEmptyCompletionIsRetriedRatherThanDone` (nada, tokens gastos em nada, só
espaço em branco) e `TestAnEmptyCompletionThatPersistsStopsSayingWhy`.
Vermelhos antes do conserto:

```
the provider was asked 1 times, want 2: an empty completion ended the turn
an empty completion that never clears ended the turn without an error
```
