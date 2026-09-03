# O frame que trazia tudo junto

**Data:** 2026-09-04
**Specs afetadas:** `202608072334-provider-adapter` (`.p`, invariantes)
**Fonte:** primeira execução de eval contra o Gemini, ao medir custo — a
medição não achou comportamento nenhum e o motivo não era o modelo

## O que mudou

O decodificador do dialeto OpenAI lê **o conteúdo do frame antes do uso**. Era o
contrário, e frame que trazia os dois tinha o conteúdo descartado.

## O defeito

```go
if c.Usage != nil {
    out, err := d.flush()   // nada foi absorvido ainda: devolve vazio
    return append(out, d.terminal(...)...), err
}
// as escolhas do frame só eram lidas DEPOIS daqui
```

Quem manda uso e chamada de ferramenta no mesmo frame perdia a chamada. O
`flush` rodava sobre um decodificador que ainda não tinha absorvido nada,
o evento terminal saía, e a função retornava antes de olhar `Choices`.

**A família Gemini inteira não conseguia chamar uma ferramenta.** Nenhuma.

## Por que sobreviveu tanto tempo

OpenAI e MiniMax mandam o uso num frame **próprio**, depois do que termina. Com
esses dois a ordem nunca importou, e a família que expõe a diferença foi
declarada sem nunca ter rodado contra uma chave — o que o changelog dela já
dizia, com todas as letras, sobre `AcceptsImages`.

O Gemini manda **um frame só**: a chamada, o `finish_reason` e o `usage` juntos,
e depois `[DONE]`.

## O sintoma não apontava para cá

De fora, modelo que responde nada e decodificador que joga a resposta fora são
idênticos. Pior: **os números de uso voltavam corretos**, então a chamada parecia
saudável. A primeira medição leu 0% e a evidência dizia `1 round(s): no tool
calls`, que é exatamente o que um modelo recusando produziria — e recusa era o
que aquele contrato estava medindo.

O que separou os dois foi mandar **o corpo da requisição do próprio cliente**
para o provedor, por fora do cliente, e receber de volta uma chamada de
ferramenta que o cliente não tinha visto.

Vale registrar como método: a evidência que a suíte guarda é do que o modelo
disse, e não havia nada nela capaz de distinguir "não chamou" de "chamou e
alguém perdeu". Quando a diferença está entre o fio e o objeto, o teste é
comparar o fio.

## `index` ausente não era o problema

O `tool_calls` do Gemini vem sem o campo `index`, que a OpenAI sempre manda. Foi
a primeira suspeita e está errada: ausente decodifica como zero, e zero é a
primeira chamada. O fixture guarda o frame **verbatim**, com a ausência, em vez
de arrumá-lo — frame editado para o que o fio "deveria" dizer é fixture que
parou de testar o fio.

## Os dois lados ficam presos

O teste de regressão afirma o frame do Gemini, e ao lado dele o formato
OpenAI/MiniMax: uso em frame separado continua terminando **sem reemitir a
chamada**. Comprar a chamada perdendo a contabilidade, ou emitindo a chamada
duas vezes, seria trocar este defeito por um pior — chamada emitida duas vezes
roda toda ferramenta duas vezes.
