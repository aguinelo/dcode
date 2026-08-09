# Decode passa a ter estado por stream

**Data:** 2026-08-08
**Specs afetadas:** `202608072334-provider-adapter` (`.r`, `.p`, `.i`)

## O que mudou

`Family.Decode(ev, tools) (StreamEvent, error)` deixa de existir. No lugar:

```go
// Decoder traduz os frames brutos de UM stream em eventos neutros.
// Com estado, uso único: um por stream, nunca compartilhado.
type Decoder interface {
    Decode(ev WireEvent) ([]StreamEvent, error)
}

type Family interface {
    // ...
    NewDecoder(tools []contextpkg.ToolDef) Decoder
}
```

Duas mudanças de assinatura, cada uma obrigatória por um motivo diferente:

- **Estado.** O decoder guarda as tool calls parciais entre frames.
- **Zero ou mais eventos.** Um frame pode não produzir nada (só um fragmento) ou produzir várias calls de uma vez (o fim do stream libera todas as que estavam sendo montadas).

Nova **RN-10**: o raciocínio do modelo é um canal separado da resposta. `EventReasoningDelta` existe, e o conteúdo dele **nunca** entra no histórico.

## Por que mudou

A assinatura antiga afirmava que um frame é uma unidade de significado. **No fio real, não é.** Captura literal de uma resposta do MiniMax-M3, com uma tool call:

```
frame 9:  {"tool_calls":[{"id":"call_e7a...","function":{"name":"glob","arguments":""},"index":0}]}
frame 10: {"tool_calls":[{"function":{"arguments":"{\"pattern\":\"**/*.go\"}"},"index":0}]}
```

O nome chega em um frame; os argumentos, no seguinte. O decoder sem estado emitia a call no frame 9, quando `arguments` ainda é a string vazia. **Toda ferramenta era chamada sem input.** No primeiro turno real contra o M3 isso produziu três `glob` seguidos respondendo *"pattern is required"*, até o detector de repetição encerrar o turno sem nada feito.

O `index` é o que separa duas calls paralelas. Sem ele, um acumulador único costuraria os argumentos de uma no meio da outra e produziria JSON inválido — falha que só aparece quando o modelo emite calls paralelas, ou seja, exatamente quando o paralelismo do loop importa.

Sobre a RN-10, a mesma captura mostra o M3 mandando o raciocínio **duas vezes**:

```
{"delta":{"content":"<think>\nThe user is asking me to list Go","reasoning":"The user is asking me to list Go"}}
```

Uma vez em `reasoning`, outra em `content` com marcadores `<think>`. O decoder lia `content`, então o pensamento era impresso ao usuário **e anexado ao histórico como fala do assistente**. Um modelo que lê o próprio raciocínio de volta como algo que disse em voz alta passa a defendê-lo; e o texto seria pago em todo turno seguinte da sessão, contra a ADR-03.

## Impacto

- `composed.Stream` cria um decoder por stream e o passa ao `pump`.
- As duas famílias do MVP ganham decoder próprio. O dialeto anthropic tinha o mesmo defeito em forma diferente: `content_block_start` abre a call com `input` vazio e os argumentos vêm como `input_json_delta`.
- Uma call que o modelo não terminou de emitir vira erro `tool_schema`, nunca uma call vazia executada.
- `finish_reason` repetido — o M3 repete — não pode emitir a call duas vezes, ou toda ferramenta rodaria em dobro. O flush é idempotente.
- Fixtures de stream passam a terminar como o fio real termina. Um stream de um frame só, sem terminador, não existe e não deve ser usado como fixture.

## Alternativa descartada

Acumular dentro do `composed.pump`, mantendo `Decode` puro. Descartada porque **como uma call é partida é específico do dialeto** — o openai fragmenta por `index` em `tool_calls`, o anthropic por bloco de conteúdo com `input_json_delta`. A montagem no pump exigiria que ele conhecesse os dois formatos, que é precisamente o acoplamento que a separação transporte × família existe para impedir.
