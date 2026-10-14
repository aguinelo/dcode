# O loop é um recurso do daemon

**Data:** 2026-10-14
**Specs afetadas:** `202608072240-client-server-protocol` (`.p` §4, §5, §8,
§9). Passo 4 da §8 de `desktop/docs/proposals/loop.md` (N4), decisões Q2 A e
Q7 A, aprovadas pelo dono em 2026-10-13. A sequência que estas rotas conduzem
está no changelog de mesmo instante da `loop-command`.

## O que mudou

Cinco rotas novas:

| Método | Rota | Corpo → resposta |
|---|---|---|
| `POST` | `/v1/loops` | `StartLoopRequest` → `LoopRun` |
| `GET` | `/v1/loops?workspace=` | → `ListLoopsResponse` |
| `GET` | `/v1/loops/events` | **SSE**: `snapshot`, `changed` |
| `POST` | `/v1/loops/{id}/continue` | → `LoopRun` |
| `POST` | `/v1/loops/{id}/stop` | `StopLoopRequest` → `LoopRun` |

`Session` e `Conversation` ganham `loop_id,omitempty`. Três códigos novos:
`loop_already_running` (409), `loop_not_awaiting_review` (409) e
`loop_not_found` (404). O cliente Go ganha `StartLoop`, `ListLoops`,
`WatchLoops`, `ContinueLoop` e `StopLoop`.

## Por quê

O `/loop` morava na TUI, e o desktop não tinha como começar nem acompanhar
um. Com a sequência no daemon, os dois clientes veem o mesmo loop, e ele não
morre com o cliente que o pediu.

- **`args` como foi digitado.** A gramática tem uma casa só, a do daemon; o
  cliente reconhece `/loop` (RN-3 da `loop-command`) e nunca interpreta o
  argumento.
- **Um loop vivo por workspace (Q7 A).** Dois loops na mesma árvore medem o
  trabalho um do outro. O protocolo não tem um `conflict` genérico, e um código
  que diz o que houve é o que o cliente consegue dizer à pessoa.
- **O primeiro `continue` vence.** O estado sai de `awaiting_review` antes de
  qualquer outra coisa; o segundo cliente recebe `loop_not_awaiting_review` e o
  estado novo pelo fluxo.

Divergências da proposta — sem `mode` no pedido, sem frame `removed`, um
terceiro código — estão explicadas no changelog da `loop-command`.

`experimental`, abaixo de 1.0: rotas e campos novos são **MINOR** como
feature.

## O que ainda não está ligado

- A TUI continua conduzindo o próprio `/loop` (passo 5 da proposta).
- O desktop só aprende o campo `loop_id`; começar, seguir e parar um loop pela
  janela são os passos 7 e 8.
- A pergunta de fechar do desktop (D19) ainda conta só sessões, não loops.

## O que a revisão do dono mudou antes do merge

- `LoopRun` ganha `warnings` e `LoopSpecRun` ganha `reason` (como o último
  turno da pasta terminou).
- Workspace de loop é conferido como o de uma sessão: existe, é pasta, e é lido
  através dos links; e a recusa de um segundo loop vale para quem está dentro
  ou em volta da árvore de um loop vivo, não só para o mesmo caminho escrito.
- O fluxo de loops tira o retrato e inscreve o cliente sob a mesma trava, para
  nenhuma mudança cair entre os dois; inscrito lento é largado e isso é dito.
- Um turno de loop num contexto já cancelado não é submetido, e um cancelado
  depois de submetido interrompe a sessão.
- Daemon encerrando não começa loop; id que não pôde ser sorteado é erro.
- O cliente Go encerra a observação, dizendo, num quadro que não consegue ler.
