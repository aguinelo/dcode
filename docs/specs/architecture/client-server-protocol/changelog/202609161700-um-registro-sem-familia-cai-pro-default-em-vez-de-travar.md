# Um registro sem família cai pro default em vez de travar

**Data:** 2026-09-16
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`, a linha sobre
reconectar ao pacote, adicionada em
[202609161500](202609161500-o-resume-carrega-o-pacote-nao-so-o-nome.md))
**Fonte:** o mesmo usuário, tentando o fluxo relatado ali, minutos depois de
instalado: `dcode -c` numa sessão gravada **antes** do fix trocou "volta pro
MiniMax" por "não constrói a sessão nenhuma" — `internal: provider: no family
claims model "qwen3.5-9b"`, o daemon inteiro recusando a subir.

## O que `202609161500` não cobriu

Aquela mudança lê `Model`/`Family`/`Transport`/`BaseURL` do `session.created`
do registro sendo retomado e aplica os quatro direto, sem checar se `Family`
veio vazio — e um registro gravado por um binário de **antes** dessa mudança
não tem `family` nenhum no payload, porque o campo não existia. Aplicar só
`Model: "qwen3.5-9b"` contra a família default (vazia, resolvida por prefixo)
não bate com nenhuma família conhecida — e diferente do bug original, essa
falha acontece **na construção da sessão**, antes de haver qualquer sessão ou
daemon pra reportar o motivo à pessoa. O `dcode -c` inteiro morre com um erro
de uma linha.

## O que mudou

A condição para aplicar o pacote de origem passou a exigir `Family` também,
não só `Model`. Registro sem `family` — de antes desta funcionalidade
existir — cai pro `Base` do daemon, exatamente a leitura de antes de
`202609161500` existir: errada do mesmo jeito que o bug original era errado
(volta pro default em vez de ficar no modelo certo), mas **constrói**. Um
registro criado depois desta versão sempre tem `family` — vazio quando é de
propósito (perfil que não declarou um, porque o modelo resolve por prefixo
igual) ou preenchido (perfil como `qwen-local`, que declara `generic`) — e
esses continuam reconectando certo, porque `Family` presente e vazio nunca
foi o problema; `Family` ausente por ser de antes é.

## Por que não tentar resolver e só cair pro default se falhar

Tentar construir com o pacote de origem e recuar pro `Base` se
`buildProvider` recusar pediria desfazer parte do que `New(opts, ...)` já fez
antes de falhar, ou construir duas vezes — o mesmo motivo por que
`daemon.build` não tenta-e-recua em nenhum outro lugar. Checar `Family`
antes é mais barato e cobre exatamente o caso que existe: um registro sem o
campo simplesmente não tem a informação que a restauração precisa, e isso é
sabido sem tentar nada.
