# O daemon embutido morre e diz

**Data:** 2026-09-17
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção sobre o que encerra o programa)
**Fonte:** regra do usuário cravada nesta sessão — "qualquer falha tem que
falhar com aviso, seja falha de llm, timeout, entrou num beco sem saída ou de
código" — e um levantamento do próprio código encontrando este como o caso
mais grave: um daemon inteiro podia morrer sem que ninguém fosse avisado.

## O que quebrava

`startEmbedded` (`cmd/dcode/tui.go`) sobe um daemon dentro do próprio
processo do cliente e roda `d.Serve(serveCtx)` numa goroutine, descartando o
erro de retorno com `_ = d.Serve(serveCtx)`. `Serve` devolve `nil` quando o
encerramento foi pedido (o `context` cancelado mapeia `http.ErrServerClosed`
pra `nil`) e o erro de verdade em qualquer outra saída — mas "qualquer outra
saída" ia direto pro `_`.

Se o daemon caísse sozinho no meio de uma sessão — um panic recuperado em
outro lugar que ainda deixou o listener morto, um erro de I/O no socket, o
que for — toda chamada seguinte do cliente contra ele passava a falhar contra
um socket que ninguém responde mais. Do lado de quem está na tela, isso é
indistinguível de uma resposta lenta até a pessoa desistir e sair — sem nunca
saber por quê. Exatamente o "beco sem saída de código" que a regra nomeia.

## O que mudou

`watchServe(ctx, serve)` (`cmd/dcode/tui.go`, extraída de dentro de
`startEmbedded`) roda `serve` numa goroutine e só manda pro canal `failed`
quando `serve` devolve um erro — nunca no encerramento pedido, porque esse já
devolve `nil` pelo contrato que `Server.Serve` já tinha. `startEmbedded` passa
esse canal adiante; `tui.Options` ganha `DaemonFailed <-chan error`, `nil`
quando o cliente está anexado a um daemon que não é dele (`--socket`, ou um
`dcode serve` já rodando — esse sobreviver ao cliente é o esperado, não uma
falha).

`program.Init` só agenda `watchDaemon()` quando `DaemonFailed` não é `nil`.
Quando dispara, vira um `errMsg` (`gen: 0`, tratado como atual por definição —
isso não pertence a geração de stream nenhuma) que encerra o programa com o
motivo na tela normal — o mesmo caminho que uma sessão remota derrubada já
usava, agora cobrindo o caso que morria em silêncio.

## Por que extrair `watchServe` em vez de testar contra um servidor de verdade

Forçar `Server.Serve` a falhar de verdade (sem ser via o `context` cancelado)
pediria fechar o listener por baixo dele ou algo igualmente frágil — um teste
que descobriria menos sobre o contrato do que quebraria por acidente de
timing. `watchServe` isola exatamente a parte que importa — "só reporta
quando `serve` sai sem ter sido pedido" — contra uma função `serve` falsa que
o teste controla inteiramente: devolve `nil` depois do `ctx` cancelar, ou
devolve um erro na hora. As duas leituras do contrato ficam testáveis sem
nunca precisar de um HTTP de verdade.
