# O filho leva um título curto

**Data:** 2026-10-18
**Specs afetadas:** `202610132000-orchestration` (`.p` §10, §13).

## O que aconteceu

No teste ao vivo de 2026-10-18, com três tarefas, a lateral do desktop listava cada filho pela
primeira linha da tarefa ("Crie no pacote `listas` (Go, módulo …") e cada revisor ("Review commit
3aa49175…"). O dono decidiu que a lateral mostra só a orquestradora (desktop D69) e que cada filho
tem um título curto de tópico dado pela orquestradora no despacho, como "listas: Unicos", nunca a
primeira linha da tarefa — o mesmo na aba Agentes, no plano, nas notificações e nos cabeçalhos do
cartão.

## O que mudou

- **Núcleo.** `ShortTitle` (`internal/orchestrate/title.go`): o título do item quando é uma linha
  de até `TitleLimit` = 40 caracteres; senão um derivado dele, ou da primeira linha da `task`
  quando falta, cortado na última palavra que cabe, com "…". `Delegate` dá a cada item o título
  curto antes de validar o lote; a linha `dispatch` guarda o pedido em `asked_title`.
- **Sessão do filho.** `ChildRequest` leva o título, e o daemon nomeia a conversa do filho com ele
  (`session.renamed`) logo depois do `session.created`. A lista de conversas e o desktop leem esse
  nome; antes leriam a primeira linha da tarefa.
- **Ferramenta.** A descrição de `title` no esquema de `delegate` pede um título de tópico de até
  40 caracteres, com o exemplo; a resposta diz, item a item, o título que o harness derivou.

## O que o código decide onde o dono deixou a forma em aberto

- **Derivar em vez de recusar.** Título longo ou ausente não recusa o lote: recusar o despacho
  inteiro por causa de um nome custaria uma volta da orquestradora, e o dono pediu "exigir ou
  derivar". A recusa sem `task` ou sem `why` continua.
- **40 caracteres.** O bastante para "pacote: função" e para caber numa linha da lateral e no
  cabeçalho do cartão; o limite é o mesmo para a branch, que já sai do título.
- **Dito, não silencioso.** O título derivado vai à resposta de `delegate`, para a orquestradora
  falar do filho como a pessoa o lê, e o pedido fica no log.
- **Nome que não grava falha o filho.** Se o `session.renamed` não entra no log, o daemon tira a
  sessão e devolve o erro: um filho cuja conversa não tem o nome do despacho seria lido pela
  primeira linha da tarefa outra vez.
