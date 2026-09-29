# A pergunta carrega o prazo

**Data:** 2026-09-29
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`: §5.1, o
comentário de `ExpiresAt`; §6, passo 4; uma linha nova em §9)
**Fonte:** conferência do handoff de design do desktop contra o código
(`refs/design/desktop/CONFERIDO.md`, divergência 9, na branch
`docs/desktop-design-brief`).

## O defeito

`tool.approval_required` saía com `expires_at` zerado, sempre. O laço anunciava
a pergunta em `askApproval` e só depois a entregava a quem responde; no daemon,
quem responde é a sessão, que punha o prazo numa cópia só dela — e nenhuma rota
expõe essa cópia: `Pending()` só é usado em teste. Passado o
`-approval-timeout` do daemon, dois minutos por padrão, a pergunta era negada
sozinha.

Um cliente não tinha como mostrar quanto tempo a aprovação ainda tinha, e uma
sessão de fundo podia ser negada sem que a pessoa visse contagem nenhuma. O
contrato prometia o campo — §6, passo 8: "sem resposta até `ExpiresAt` →
negado" — e o entregava vazio.

## O que mudou

O prazo é posto **antes** de a pergunta sair, por quem vai aplicá-lo, e é
aplicado a partir do valor que a pergunta carrega:

- quem nega sozinho depois de um tempo diz quando, antes do anúncio
  (`loop.Deadliner`, opcional). No daemon é o adaptador que já segurava o
  `-approval-timeout`;
- o laço carimba esse instante em `ExpiresAt` e só então emite o evento;
- a sessão segura a pergunta até o `ExpiresAt` que ela carrega, em vez de
  calcular um prazo seu. `Session.Approve` perdeu o parâmetro `timeout`.

Um valor, posto uma vez: o instante que o cliente conta e o instante que nega
são o mesmo campo.

## Por que o prazo é de quem responde, e não do laço

A saída mais curta era dar o `timeout` ao laço e carimbar todo pedido. Não
serve: no `dcode once` quem responde é o terminal, que espera o quanto a pessoa
levar, e o filho de uma delegação recusa na hora. Um prazo posto pelo laço
anunciaria, para esses, uma contagem que ninguém cumpre — o mesmo defeito com o
sinal trocado. Por isso o prazo é perguntado a quem responde, e quem não tem
prazo anuncia sem prazo: `expires_at` zerado passa a querer dizer "sem prazo",
e não "ninguém contou".

A interface já dizia isso: negar por tempo é responsabilidade de quem
implementa `Approver`, não do laço. Faltava o prazo chegar ao anúncio.

## A invariante

A linha nova em §9 é reivindicada por
`TestAnApprovalIsAnnouncedWithTheDeadlineTheSessionEnforces`, que passa pela
montagem do próprio daemon — modelo roteirizado sobre HTTP de verdade, o
`build` do daemon, uma sessão de verdade. O defeito morava entre três partes, o
laço que anuncia, a sessão que aplica e o daemon que segura o prazo, e cada uma
estava certa sozinha.

## O que um cliente vê

Pelo fio só o daemon responde, e ele sempre tem prazo: todo
`tool.approval_required` passa a carregar o instante em que a pergunta será
negada. Nada mais muda — nem a ordem dos eventos, nem o `410 approval_expired`
a quem responde tarde. Continuar uma sessão não reproduz
`tool.approval_required` (`carry.go`), então um prazo gravado não reabre
pergunta nenhuma.

Estabilidade da spec: `experimental`. Não é quebra: o campo existia, e passa a
dizer o que prometia.
