# Uma lista de conversas, vivas e gravadas

**Data:** 2026-09-30
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`: duas rotas em
§4, a §5.2, uma linha em §7, cinco linhas novas em §9)
**Fonte:** o pedido N3 do cliente desktop (`desktop/docs/DECISIONS.md`, D21),
decidido com a pessoa em 2026-09-29.

## O problema

A lateral do desktop e a busca dele precisam de todas as conversas: as vivas no
daemon e as que terminaram e ficaram no disco. O protocolo só tinha
`GET /sessions`, com as vivas e sem título, turnos, selo, diff ou última
atividade. As gravadas existiam só para um cliente que lesse a pasta de
registros por conta própria — que é o que a TUI faz no `-r`, acoplando o cliente
ao formato do disco. E acompanhar N sessões pedia N fluxos SSE.

## O que mudou

- **`GET /v1/conversations`** lista as conversas, vivas e gravadas, cada uma uma
  vez, da atividade mais recente para a mais antiga. A viva vem com o estado de
  agora; a gravada, como `recorded`. `?workspace=` guarda um projeto.
- **`GET /v1/conversations/events`** é um fluxo só para a lista inteira: abre com
  o retrato (`snapshot`) e depois manda cada conversa que mudou no que uma lista
  mostra (`changed`), ou que saiu dela (`removed`, a que terminou sem registro).
  Texto chegando em fragmentos, raciocínio e progresso não viram mudança.
- **O resumo** — título, nome, projeto, branch, modelo, estado, turnos, selo do
  último turno, diff somado como a barra da TUI soma, início, última atividade,
  último evento, de qual conversa continua — é dobrado dos eventos que vão para
  o registro, pelo mesmo código, venham eles ao vivo ou lidos do arquivo.
- **Continuar uma gravada** é `CreateSessionRequest.Resume`, como sempre: abrir
  uma antiga não ressuscita a sessão, abre uma nova carregando a conversa.

## Como, e por quê assim

- **O estado vem da sessão, não dos eventos.** O estado de uma sessão muda em
  momentos que nenhum evento marca: ela volta a `idle` depois de o
  `turn.completed` já ter saído. Uma lista que deduzisse o estado dos eventos
  mostraria um turno terminado como ainda rodando. A sessão avisa cada troca, sob
  a própria trava, então as trocas chegam à lista na ordem em que aconteceram.
- **Só o que vai para o registro.** A conversa carregada de uma continuação entra
  no log e não no registro; dobrá-la faria a linha viva discordar da que o
  registro dá quando a conversa termina.
- **Retrato no começo, sem `from`.** A lista não tem sequência, e o retrato é a
  posição: quem fica para trás é desligado e reconecta para outro retrato, que se
  aplica como o primeiro. A inscrição é feita antes do retrato, para uma mudança
  no meio ser mandada em vez de perdida — no pior caso duas vezes, que quem aplica
  mudanças não distingue de uma.
- **Registro lido de novo só quando muda.** Um registro pode ter megabytes e a
  lista é pedida toda hora: o resumo de cada arquivo fica guardado por tamanho e
  data, e só o que mudou é relido.
- **Uma conversa continuada sem pergunta própria** herda o título da que ela
  continua, seguindo a cadeia alguns passos.

## As invariantes

As cinco linhas novas em §9 são reivindicadas por
`TestTheListJoinsLiveAndRecordedConversationsOnce`,
`TestTheListStreamOpensWithASnapshotThenSendsChanges` e
`TestAConversationThatEndsStaysAsRecorded` (`internal/app`, pela montagem do
próprio daemon, com um modelo roteirizado), e por
`TestARecordIsReadAgainOnlyWhenItChanged` e `TestALiveSummaryMatchesItsRecord`
(`internal/session`).

Rotas e tipos novos, nenhum contrato existente mudado: MINOR.

## O que fica de fora

- **A TUI passar a usar a rota no `-r`**, em vez de ler o disco: mudança da TUI,
  com o seu próprio PR.
- **Conversa podada** do disco não sai do fluxo na hora: sai do próximo retrato.
