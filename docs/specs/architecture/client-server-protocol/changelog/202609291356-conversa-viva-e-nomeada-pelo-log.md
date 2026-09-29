# Conversa viva é nomeada pelo log

**Data:** 2026-09-29
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`, §9: a linha da
sequência lida passa a valer só para conversa que não está carregada, e quatro
linhas novas ao lado dela)
**Fonte:** `refs/design/desktop/CONFERIDO.md`, divergência 12, na branch
`docs/desktop-design-brief` (#400) — o handoff do desktop conferido contra o
código.
**Revê:** "Escreve no registro, não na sessão viva", em
[202608210800 — Nomear uma conversa](202608210800-nomear-uma-conversa.md).

## O defeito

`POST /sessions/{id}/name` escrevia o `session.renamed` direto no arquivo do
registro, estivesse a conversa carregada ou não. Para uma conversa carregada,
isso era escrever pelas costas do log dela, e deu errado dos dois lados:

- **Ninguém anexado ficava sabendo.** O evento não entrava no log em memória,
  então não chegava a nenhum inscrito. O cliente que renomeou atualizava a
  própria lista a partir do que mandou; qualquer outro seguia mostrando o
  título antigo.
- **O registro ganhava `seq` repetido.** O `Rename` do arquivo lia o maior
  `seq` gravado e acrescentava o seguinte — justamente o número que o log
  entregaria ao próximo evento ao vivo. Uma conversa renomeada que seguia
  terminava com `[1 2 2]`: dois eventos com o mesmo número, num registro cujo
  contrato é que isso não acontece (§3: "sem lacunas, sem reuso").

E a mesma sessão passava a ter duas leituras: reproduzida do registro, trazia o
nome; observada ao vivo, não.

## O que mudou

A rota pergunta ao `Manager` se a conversa está carregada.

- **Carregada:** `Session.Rename` limpa o nome com o mesmo `CleanName` e o
  acrescenta pelo log, do jeito que `session.mode_changed` já viaja. O log
  grava o registro sob a mesma trava que dá o número, e só então entrega o
  evento a quem está anexado (RN-2): o registro recebe o nome no lugar dele na
  sequência, e o cliente o recebe ao vivo.
- **Não carregada:** o caminho de sempre. Sem log, o nome vai para o arquivo,
  com a sequência lida antes de acrescentar, sob o `renameMu`.

## A decisão que isto revê

[202608210800](202608210800-nomear-uma-conversa.md) decidiu escrever **no
registro e não na sessão viva**, com um motivo que continua certo: a trilha
lista o que o workspace gravou, quase nada disso está carregado, e um rename que
só funcionasse na sessão aberta funcionaria na única linha que não precisa dele.
É por isso que o caminho do arquivo fica.

O que estava errado era usar o arquivo **também** para a conversa carregada. Ali
o registro não é a única coisa que a conversa tem: ela tem um log com numeração
própria e clientes ouvindo, e o arquivo é a cópia que esse log escreve. Escrever
nele por fora dava à conversa dois lugares atribuindo número.

## Sessão que fecha no meio

Um `DELETE` de outro cliente pode fechar a sessão entre a rota encontrá-la e o
nome ser acrescentado. Pelo log já fechado, o nome iria para uma memória que
ninguém lê e para um registro já fechado — a falha de escrita fica guardada no
log e não volta para quem chamou —, e a resposta seria `204`.

`Session.Rename` confere o estado e acrescenta sob a mesma trava da sessão. O
`Close` marca a sessão como fechada **antes** de fechar o log, então um nome que
a encontra aberta chega ao registro antes de ele fechar, e um que a encontra
fechada recebe `session_not_found` — o código que o §8 já dá a sessão "já
encerrada". Quem tentar de novo vai pelo arquivo: a essa altura o `Manager` já
soltou a sessão.

## O que isto não resolve

- **Conversa carregada em outro daemon.** Cada `dcode` que não encontra um
  daemon compartilhado sobe um embutido, num socket privado, gravando no mesmo
  diretório de registros. Uma conversa viva no daemon de outro terminal é "não
  carregada" para este, e o nome vai para o arquivo com a sequência lida — o
  mesmo `seq` repetido, agora entre processos. Resolver isso pede coordenação
  entre processos (trava no arquivo, ou um daemon só), e não cabe aqui.
- **A janela do `Remove`.** O `Manager` solta a sessão antes de fechá-la. Um
  nome que chega entre as duas coisas vai para o arquivo enquanto o registro
  ainda pode ter `message.delta` no buffer, e pode repetir um `seq`. A janela é
  a do próprio fechamento, com um turno ainda escrevendo.
