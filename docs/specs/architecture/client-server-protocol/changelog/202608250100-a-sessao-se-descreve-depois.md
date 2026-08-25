# A sessão se descreve depois de a conversa entrar

**Data:** 2026-08-25
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`, `Session` e invariantes)
**Fonte:** relato com imagem — `dcode -c` desenhando a tela de abertura, saindo
sozinho, e deixando lixo de escape no prompt do shell

## O que mudou

`POST /sessions` monta a resposta **depois** de `EmitCarried`, e `Session` ganha
`first_seq`: o evento mais antigo que a sessão ainda guarda.

## Três falhas em fila, cada uma escondendo a próxima

**1. A resposta era montada antes da conversa entrar.** `desc := sess.Describe()`
vinha antes de `sess.Emit(session.created)` e de `sess.EmitCarried()`, e a mesma
`desc` era devolvida ao cliente. Uma sessão que tinha acabado de receber dezoito
mil eventos se descrevia com `last_seq: 0`.

**2. O cliente acreditou, e pediu do começo.** Ele assinava de `1` sempre. Uma
conversa continuada mais longa que a retenção já teve os eventos mais antigos
descartados no próprio ato de carregar — o primeiro que restava era `8411`, e
`Subscribe` recusa `from` abaixo do horizonte, como deve.

Antes de `202608242330`, o registro guardava uma cópia de tudo que a sessão
continuava, e o cliente abaixo do horizonte se recuperava por ela. Tirar a cópia
foi certo — ela crescia quadraticamente — mas ela era a rede que segurava este
caso, e o caso ficou sem rede.

**3. A recusa se perdeu na corrida com o fechamento.** O transporte escreve o
motivo em `errs` e fecha os dois canais; o `select` do cliente via o canal de
eventos fechado primeiro e devolvia "fluxo fechado", jogando o motivo fora. O
cliente saía sem dizer nada.

E o que ele teria dito morreria de qualquer forma: o fatal era desenhado no
último quadro, e a tela alternativa leva o último quadro embora.

## O lixo no prompt

Sintoma, não causa. O Bubble Tea pergunta ao terminal por DECRQM se ele suporta
os modos 2026 e 2027; a resposta chega em `stdin`. Saindo em menos de um segundo,
o processo já não estava lendo quando ela chegou — e ela apareceu no prompt do
shell. Consertado o encerramento prematuro, some com ele.

## Invariante

| Invariante | Teste |
|---|---|
| A resposta descreve a sessão depois de a conversa estar nela | `TestContinuingDescribesTheSessionTheConversationIsIn` |
