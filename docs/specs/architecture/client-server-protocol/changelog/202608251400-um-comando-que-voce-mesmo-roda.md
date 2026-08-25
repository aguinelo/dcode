# Um comando que você mesmo roda

**Data:** 2026-08-25
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`), `202608081250-client-tui` (`.p`, seção 7)
**Fonte:** pedido de quem usa — "pq o ! nao funcionou pra executar local? deve ter
um aviso quando digitar ! pra ele avisar"

## O que mudou

Uma linha começando com `!` não é enviada ao modelo: ela **roda**. A saída chega
à tela como os mesmos eventos de ferramenta que a transcrição já desenha, e
entra no histórico como uma mensagem do usuário — porque foi o usuário quem
rodou.

## Por cima do modelo, nunca por cima do sandbox

O comando passa pela ferramenta `bash`, pelo mesmo `execute` que o turno usa:
declara os caminhos, a política avalia, e um cruzamento é posto à pessoa
exatamente como seria se o modelo tivesse pedido. Um atalho de shell que
pulasse isso seria um furo na única fronteira em torno da qual este produto é
construído.

O que o `!` pula é o modelo. É só isso que ele pula.

## Por que mensagem de usuário e não resultado de ferramenta

Um resultado de ferramenta sem uma chamada antes dele é uma forma que provedor
nenhum aceita. Inventar uma chamada do assistente para segurar uma saída que ele
não pediu seria pôr palavras na boca do modelo. A saída entra como *"I ran `x`
myself. It printed:"* — que é o que aconteceu.

E ela **precisa** entrar: sem isso o modelo responde sobre um workspace cujo
estado ele não pode ver.

## Rota própria

`POST /sessions/{id}/exec`, e não um campo em `SubmitTurnRequest`. Enviar começa
um turno e isto não começa. Bloqueia até o comando terminar, porque a pessoa
está esperando a saída; o corpo da resposta é vazio, e a saída chega pelo fluxo
de eventos — então um cliente que caiu no meio e voltou vê o mesmo que um que
ficou.

## O aviso

Assim que o primeiro caractere é `!`, o rodapé da área de digitação diz
*"! roda aqui, sem enviar — o modelo lê a saída"*. Enquanto a linha ainda pode
ser apagada, e sem tecla nenhuma a mais: uma confirmação para `!ls` seria ruído.
