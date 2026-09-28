# Um lote volta inteiro, na ordem em que foi feito

**Data:** 2026-09-28
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`, duas linhas
novas em §9, ao lado de "Chamada de ferramenta sem resultado não entra no
histórico reconstruído")
**Fonte:** defeito no `Rebuild`, confirmado com uma sonda descartável chamando
`rebuildFrom` direto sobre os eventos de um lote de duas chamadas.

## O defeito

`rebuildFrom` transforma o registro de uma sessão de volta no histórico que o
modelo recebe quando ela é continuada. Ele segura aberta a mensagem do
assistente — texto e chamadas — e a despachava no primeiro `tool.completed`,
levando só as chamadas respondidas **até ali**, e zerando a lista em seguida.

Com uma chamada por resposta isso não aparece. Com um lote aparece sempre: o
loop roda leituras e buscas independentes ao mesmo tempo (RN-3 do agent-loop),
e para `tool.requested` c1, `tool.requested` c2, `tool.completed` c1,
`tool.completed` c2 o histórico reconstruído era

    user
    assistant  chamadas=[c1]
    tool       resultado=c1
    tool       resultado=c2   ← responde a uma chamada que nenhuma mensagem fez

A Anthropic exige que todo `tool_result` corresponda a um `tool_use` da
mensagem anterior; a OpenAI, que toda mensagem `tool` tenha o seu
`tool_call_id` numa entrada de `tool_calls` anterior. Uma sessão continuada com
um lote paralelo em qualquer ponto do registro falhava no primeiro turno depois
de retomada — e, como o histórico só é acrescentado, em todos os seguintes.
Nenhum teste cobria mais de uma chamada por mensagem.

## O que mudou

Os resultados ficam guardados, por chamada, até o lote acabar. Um resultado não
fecha nada sozinho, porque o resto do lote ainda pode estar terminando; o que
fecha é o modelo voltar a falar — texto ou chamada depois de resultados é a
resposta seguinte —, a próxima pergunta, ou o fim do registro. Aí sai uma
mensagem do assistente com o texto e todas as chamadas que tiveram resultado,
seguida de um resultado por chamada.

Chamada sem resultado continua fora, e sozinha: numa interrupção no meio de um
lote, o que terminou fica e o que não terminou sai, sem levar as irmãs junto.

## A ordem dos resultados

O registro guarda os `tool.completed` na ordem em que terminaram: cada um é
emitido de dentro da execução da sua chamada. O histórico ao vivo, não — o loop
anexa os resultados pelo índice em que o modelo emitiu cada chamada, "nunca
pela ordem de término" (RN-3.1 do agent-loop).

A reconstrução segue as chamadas, não o registro, por dois motivos:

- **Continuar é continuar a conversa que o modelo teve.** Na ordem de término,
  a sessão retomada mandaria ao modelo o mesmo conteúdo noutra ordem — uma
  conversa que ele nunca recebeu.
- **A ordem de término é o resultado de uma corrida.** O mesmo lote pode
  terminar em ordens diferentes cada vez que roda. Reconstruir por ela faria o
  histórico, e o prefixo que ele vira, depender de qual leitura foi mais
  rápida — exatamente o que a RN-3.1 existe para impedir no histórico ao vivo.

A ordem real de execução não se perde: continua no registro, em
`started_at`/`finished_at`, que é onde a RN-3.3 a põe.

## Por que "respondida" vale para a mensagem aberta, e não para o registro

A saída óbvia era uma primeira passada no registro inteiro recolhendo as
chamadas respondidas, para o despacho saber quais **vão** ser respondidas. Não
resolve a ordem — os resultados continuariam saindo na ordem de término — e
confia numa unicidade que o registro não garante: o id de uma chamada é único
dentro de uma resposta, não ao longo de uma cadeia. O comando digitado com `!`
numera `x1-1`, `x2-1`… a partir do motor, e cada trecho continuado é um motor
novo, que recomeça do um; um servidor compatível com OpenAI cunha os ids que
quiser. Um `!` recusado num trecho e um `!` que rodou no seguinte, ambos
`x1-1`, fariam o primeiro parecer respondido e voltar como chamada sem
resultado.

Guardar os resultados da mensagem aberta resolve as duas coisas numa passada
só.

## O que a reconstrução ainda não é

Idêntica ao histórico ao vivo, em dois pontos que esta mudança não toca:

- Chamada respondida sem executar — recusada, não permitida, ferramenta
  desconhecida — não emite `tool.completed`. Ao vivo ela recebeu um resultado
  de erro; na reconstrução sai, pela regra de sempre ("sem resultado não
  entra"), que é válida para o provedor mas perde o que o modelo viu.
- Comando digitado com `!` entra ao vivo como mensagem da pessoa, e volta na
  reconstrução como chamada do assistente — na última mensagem dele, se ela
  ainda estiver aberta.
