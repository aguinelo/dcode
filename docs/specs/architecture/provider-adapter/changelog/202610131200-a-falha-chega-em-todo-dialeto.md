# A falha chega em todo dialeto

**Data:** 2026-10-13
**Specs afetadas:** `202608072334-provider-adapter` (`.r` — nova RN-12; `.p` — §3.1 e invariantes)
**Fonte:** auditoria de "harness > modelo" — as mesmas premissas valem qualquer que seja o modelo

## O que mudou

Um `ToolResult` com `IsError` passa a ser codificado, nos dois dialetos, com o
conteúdo aberto por um texto fixo — `The tool call failed.` e uma linha em
branco — seguido da saída da ferramenta. O dialeto `anthropic` continua
mandando `is_error: true` além do texto. Resultado bem-sucedido não muda.

## O defeito

`encodeOpenAI` montava a mensagem de ferramenta com `tool_call_id` e
`content = Output`, e só. `IsError` não ia para lugar nenhum: o dialeto OpenAI
não tem campo de erro na mensagem de ferramenta. `encodeAnthropic` mandava o
mesmo resultado com `is_error`.

Quem usa o dialeto OpenAI: `minimax-m3` no transporte preferido, `gemini`, e
`generic` — todas as famílias menos `claude`. Para elas, uma falha era lida
exatamente como um sucesso. Um `Edit` recusado por correspondência ambígua
chegava como o texto do motivo, sem nada dizendo que a edição não aconteceu; o
modelo tinha de deduzir pela prosa.

A RN-4 do loop diz que erro de ferramenta alimenta o modelo. Ela valia inteira
para uma família e pela metade para as outras três, e nenhum teste olhava os
dois dialetos lado a lado.

## Por que o texto vai também no `anthropic`

A alternativa era marcar só onde falta o campo. Ela deixa o fato chegar nos
dois, mas com palavras diferentes — campo num, frase no outro —, e a regra que
esta mudança escreve é que o modelo lê **a mesma coisa** venha por onde vier.
Com o texto nos dois, o teste compara o conteúdo dos dois dialetos e exige
igualdade, em vez de comparar duas formas de dizer.

`is_error` fica: é o jeito daquele dialeto, e o provedor pode tratá-lo além do
que o modelo lê.

## Por que na codificação e não no histórico

O histórico guarda o resultado neutro. Marcar na gravação faria do texto um dado
da sessão — uma sessão gravada antes desta mudança reidrataria diferente de uma
gravada depois. Marcando no `Encode`, que é função pura do histórico, as duas
saem iguais, e o prefixo continua sendo função do histórico (ADR-03).

## Efeito nas medições

Medições feitas antes disto no dialeto OpenAI viram falhas sem marca. Os
contratos que dependem de o modelo perceber a falha — `tool-error-recover`,
`no-blind-retry`, `toolcall-recover` — foram medidos nessa condição. Não se
rebaixa nem se eleva limiar nenhum aqui; a próxima medição é a que vale.

## Como fica preso

`TestAFailedToolResultSaysSoOnEveryDialect` codifica o mesmo resultado pelos
dois transportes do `minimax-m3` — a única família que fala os dois — e exige o
mesmo conteúdo nos dois, terminado pela saída da ferramenta, diferente da saída
crua quando falhou e igual a ela quando não falhou. Vermelho antes do conserto:

```
openai content = "no such file: a.txt": a failure reads exactly like a success
```
