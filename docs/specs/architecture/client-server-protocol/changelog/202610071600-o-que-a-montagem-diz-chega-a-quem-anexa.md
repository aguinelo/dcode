# O que a montagem diz chega a quem anexa

**Data:** 2026-10-07
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`, uma linha nova
em §9)
**Fonte:** leitura do código. `app.New` emite `session.error` com
`memory_unreadable` quando a memória do workspace não pode ser lida, e um
`grep` por `memory_unreadable` achava só a emissão — nenhum teste, nenhum
cliente que a tivesse visto.

## O problema

`app.New` recebe um emissor e o usa enquanto monta a sessão. Pelo daemon, esse
emissor é uma closure que repassa para `sess.Emit` só se `sess != nil` — e
`sess` só é atribuída depois que `New` retorna. Tudo o que `New` diz durante a
montagem caía no chão, em silêncio.

Hoje é uma coisa só: a memória ilegível. O comentário ao lado da emissão diz que
ela "vale dizer e não vale parar por ela" — e através do daemon, que é por onde
o TUI e o desktop abrem toda sessão, ela não era dita. A sessão abria como se a
memória tivesse sido lida, e a pessoa não tinha como saber que o que as sessões
anteriores aprenderam não estava no prefixo.

O caminho de execução única (`dcode -p`) não tinha o problema: o emissor dele
escreve no terminal desde o primeiro instante.

## O que mudou

- **O daemon segura** o que `New` emite enquanto a sessão não existe, na ordem
  em que foi dito, em `session.Session.Opening`.
- **A sessão diz depois** de se anunciar: o servidor chama `EmitOpening()` logo
  depois de `session.created`, de `EmitCarried()` e de `EmitNotices()`
  ([202610071242](202610071242-o-aviso-de-abertura-chega-a-quem-anexa.md)), e
  antes da resposta de criação — então `last_seq` conta o que foi dito, e um
  cliente que lê até ele lê. Os avisos de abertura vêm antes porque são sobre a
  sessão inteira; o que a montagem disse é sobre uma entrada dela.
- **Gravado**, ao contrário da conversa carregada: é desta sessão, e o registro
  é onde quem lê depois procura.

## Por que segurar tudo, e não o erro

A alternativa era `New` parar de emitir e devolver o erro num campo de
`app.Session`, para o daemon entregar — a forma que `Notices` já tem. Seria
explícito, mas só para este erro: a próxima coisa que `New` precisar dizer
cairia no mesmo buraco, e ninguém
perceberia, porque o buraco não faz barulho. Segurar no emissor fecha a classe,
não o caso — e deixa `New` e o caminho de execução única como estavam.

O log continua abrindo com `session.created`: nada do que é segurado entra
antes dele.

## O que fica de fora

O aprovador tem a mesma janela: chamado antes de `sess` existir, nega. Nada em
`New` pede aprovação hoje, e uma pergunta não pode ser segurada até depois da
criação sem prender a montagem — é outra decisão, e fica para outra branch.

## Invariante nova

- O que a montagem da sessão diz antes de a sessão existir chega a quem anexa,
  depois de `session.created` e da conversa continuada; nada dito na montagem
  se perde.
