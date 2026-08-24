# A pergunta fica no fluxo

**Data:** 2026-08-25
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 6)
**Fonte:** pedido de quem usa — "faz o approval inline", e o desenho v2 da
importação do Claude Design, onde a aprovação é um bloco da transcrição

## O que mudou

O modal de aprovação saiu. A pergunta é desenhada **no fluxo, na posição em que
foi feita**, numa quarta raia (`┃`, `?` em ASCII), e **permanece lá depois de
respondida**, com a resposta no lugar das teclas.

```
  ┃ aprovar bash cruza: network
      curl -X POST https://api.exemplo.com/v1
      regra: network.deny
      Comandos deste projeto podem alcançar a rede.
      [d] não   [a] uma vez   [P] este projeto   [G] sempre
```

## Por que a caixa não era o que protegia

A RN-6 diz que uma fronteira pendente bloqueia a entrada, e o modal era lido
como se fosse *ele* que garantia isso. Não era: quem garante é o cliente
recusar entregar a tecla ao campo de texto enquanto há pendência — o que
continua exatamente como estava, e é o que mantém letra solta legal dentro do
fluxo.

A caixa cobrava dois preços por uma garantia que não dava:

1. **Escondia o trabalho que estava sendo julgado.** A RN-7 já proibia mostrar o
   plano dentro dela, e por bom motivo — mas o resultado era decidir sobre um
   comando com o contexto dele coberto pela própria pergunta.
2. **Sumia ao ser respondida.** A decisão de fronteira é o registro mais durável
   que uma sessão produz, e era o único que a interface apagava. Meia hora
   depois não havia como ver o que tinha sido permitido nem o que foi
   perguntado.

## Casar por `ApprovalID`, não por "a última"

Duas fronteiras podem estar em voo. Resolver "a última pergunta" grava a resposta
na linha errada, e uma linha que diz que alguém permitiu algo que não permitiu é
pior que nenhuma linha. `EventApprovalResolved` traz o id; a entrada é
encontrada por ele.

## O que não foi feito

O desenho v2 oferece `e` — editar o comando antes de permitir. Não existe
equivalente no protocolo do `dcode`: um `ApprovalRequest` é sobre o comando que o
modelo pediu, e devolver outro seria responder uma pergunta diferente da que foi
feita. Fica de fora até haver fato de protocolo que o sustente.

## Invariantes

| Invariante | Teste |
|---|---|
| Aprovação pendente bloqueia a entrada | `TestKeystrokesDoNotFallThroughTheModal` |
| Bloco renderiza `ApprovalRequest.Command` | `TestTheApprovalBlockShowsTheCommandAndDefaultsToDeny` |
| Bloco não exibe o plano (RN-7) | `TestTheApprovalBlockAsksOneQuestion` |
| Respondida permanece, casada por id | `TestTheAnsweredQuestionStaysInTheStream` |
