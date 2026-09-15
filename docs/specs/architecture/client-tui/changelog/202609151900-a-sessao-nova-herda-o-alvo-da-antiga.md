# A sessão nova herdava o alvo da antiga

**Data:** 2026-09-15
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** relato do usuário — `/model qwen-local` travava para sempre na tela
"reading the conversation · 0 lines · ^C", com captura de tela anexada.

## O que quebrava

`p.catchingUp()` decide se a tela de carregamento fica em pé comparando
`p.model.LastSeq` contra `p.opts.Backlog`: enquanto a sessão ainda não leu até
lá, a linha do "lendo o histórico" (`202608081310`) continua desenhada. Esse
`Backlog`, e o `From` que diz de onde `attach()` assina os eventos, eram
escritos uma vez, na abertura do programa, e nunca mais tocados.

`case switchedMsg:` — o mesmo caminho para `/model`, `/clear` e `/resume` —
troca `p.model` por um `NewModel` zerado e reconecta o transporte para a
sessão nova, mas nunca atualizava `p.opts.Backlog`/`p.opts.From`. Uma sessão
aberta com histórico grande (`Backlog` alto) e depois trocada por `/model`
para uma sessão nova, sem histórico nenhum, ficava com o alvo antigo: `0 <
Backlog-antigo` é verdade para sempre, porque a sessão nova nunca vai ter
lido `Backlog-antigo` eventos — ela não tem esse histórico. A linha "lendo o
histórico" não tinha como se mover, e a tela cheia nunca chegava.

## Por que `/resume` não mostrava o mesmo sintoma

`/resume` busca a sessão escolhida via `GetSession` e monta `Options` com o
`Backlog`/`From` **dela** antes de reconstruir o programa — o valor chega
certo por um caminho que não passa por `switchedMsg`. `/model` e `/clear`
share o mesmo `switchedMsg`, mas nascem de uma sessão **nova**, sem backlog
próprio: o bug só aparecia nesses dois, e só quando a sessão anterior tinha
histórico suficiente para deixar `Backlog` acima de zero.

## O que mudou

`case switchedMsg:` agora lê o backlog e o `From` da resposta que criou a
sessão nova (`msg.session.LastSeq`/`msg.session.FirstSeq`) e os escreve em
`p.opts` antes de montar o modelo novo — a mesma fonte que `/resume` já usava,
agora compartilhada pelos três comandos em vez de só um deles. Uma sessão
nova tem `LastSeq == FirstSeq == 0`: `catchingUp()` lê isso como "nada para
recuperar" e a tela de carregamento nunca aparece, exatamente o comportamento
que a spec já descrevia para uma sessão sem histórico.

## O que a spec não dizia

A regra "retomar desenha uma linha enquanto lê o histórico" (seção 10) nunca
condicionou isso a "vindo de `/resume`" — ela é sobre `catchingUp()`, e
`catchingUp()` não sabe por qual comando a sessão trocou. O bug era uma
lacuna de implementação, não uma leitura errada da regra: os três comandos
que passam por `switchedMsg` precisam da mesma disciplina de adotar o
backlog da sessão que **de fato existe agora**, nunca o da sessão anterior.
