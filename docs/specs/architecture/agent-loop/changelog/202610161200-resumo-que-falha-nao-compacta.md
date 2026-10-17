# Resumo que falha não compacta

**Data:** 2026-10-16
**Specs afetadas:** `202608072335-agent-loop` (`.p`), `202608072334-provider-adapter` (`.p`)

> **Regra:** só um resumo inteiro aplica o plano de compactação. Resumo que
> falha, volta vazio ou vem cortado deixa o histórico como estava e diz por quê.

## O que acontecia

A compactação chama o modelo para resumir o trecho que vai sair do contexto, e
dois pontos aceitavam qualquer resposta como resumo:

- o resumidor (`internal/app`) lia só `EventTextDelta` e devolvia `nil` fosse
  qual fosse o fim do stream. Um 429 no meio, ou uma resposta cortada no teto
  de saída, voltava como resumo — metade de um, aceita como inteira;
- o loop (`applyCompaction`), quando o resumidor devolvia erro ou texto vazio,
  trocava o resumo por uma frase fixa ("Earlier work in this session was
  summarised to save room."), aplicava o plano e emitia `session.compacted`.

Um rate limit, uma credencial revogada ou um stream cortado no instante errado
apagavam parte da sessão do contexto, e o cliente mostrava a compactação como
feita. É exatamente o estado que parece sucesso que o `AGENTS.md` proíbe, num
dos caminhos onde o `TESTING.md` diz que os dados do usuário podem sumir.

O teste `TestCompactionFallsBackWhenSummarisingFails` afirmava esse
comportamento ("perder detalhe é melhor que perder a sessão"). Ele não sai em
silêncio: é substituído pelos testes que afirmam o contrário, porque o que ele
protegia era o defeito.

## O que mudou

- O resumidor só devolve texto de stream terminado em `done` sem `Truncated`.
  `EventError` devolve o `ProviderError` com a classe intacta; `Truncated` e
  stream que fecha sem evento terminal são erro.
- `applyCompaction` não aplica o plano sem resumo: erro e texto vazio voltam
  como erro, e a sessão fica intocada. A frase fixa só sobra para motor montado
  **sem** resumidor, que nenhum caminho do produto monta.
- Toda compactação que não aconteceu emite `session.error` com código
  `compaction_failed`, dizendo que o histórico ficou inteiro e a causa.
- A recusa `context_size` que termina o turno passou a ser dita em
  `session.error` também. A sessão descarta o que `Run` devolve, então um turno
  encerrado por contexto recusado era, até aqui, um turno que só parava.

## Por que o gatilho automático segue e não encerra o turno

Foram consideradas as duas saídas: manter o histórico e avisar, ou falhar o
turno com o erro classificado. Cada caminho fica com a que cabe nele:

| Caminho | Escolha | Motivo |
|---|---|---|
| gatilho automático | mantém e avisa; o turno segue | dispara numa fração do orçamento (`CompactAt`), então ainda há espaço; encerrar jogaria fora o trabalho do turno por um corte que ainda não era preciso |
| `/compact` | devolve o erro | é a pessoa pedindo; ela recebe a causa como resposta ao pedido, e não um aviso solto |
| `context_size` | encerra em `StopError` | o provider já disse que não cabe e a única saída falhou; repetir seria pedir de novo o que já foi recusado |

O gatilho automático e a recusa do provider se encontram: se o espaço acabar
depois de um resumo que falhou, a próxima chamada é recusada, a compactação
forçada tenta de novo — um 429 passageiro pode ter passado —, e se falhar outra
vez é esse caminho que encerra, com as duas causas ditas.

## O que não mudou

Nenhum limiar, nenhuma chave de configuração, nenhum evento novo no protocolo:
`session.error` já era o evento de falha não fatal (`memory_unreadable`,
`prompt_rebuild_failed`), e `compaction_failed` é mais um código dele.
