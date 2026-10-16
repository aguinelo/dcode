# O teto de saída é do modelo

**Data:** 2026-10-15
**Specs afetadas:** `202608072334-provider-adapter` (`.p`, seções 2.1 e 7);
`202608072335-agent-loop` (`.p`, seções 2, 3 e 8; `.config`, seção 1.1)
**Fonte:** uso real com a família `claude` — respostas longas cortadas no meio,
sem nada na tela dizendo por quê.

## O que mudou

`encodeAnthropic` mandava `max_tokens: 8192` sempre que a requisição não
trazia um, e o laço nunca trazia. O dialeto exige o campo, e 8.192 é um
dezesseis avos do que a maioria dos modelos atuais aceita.

Agora o teto quando nada foi configurado é da família, por modelo:
`Claude.OutputCeiling(model)`, uma tabela casada pelo prefixo mais longo com os
limites que a Anthropic publica (128.000 nos modelos atuais, 64.000 na linha
4.5 e em `claude-haiku-4-5`, 32.000 em `claude-opus-4`/`4-1`, 4.096 em
`claude-3-haiku`). `minimax-m3` no dialeto anthropic segue com 8.192.

Nova chave, `limits.max_output_tokens` / `DCODE_MAX_OUTPUT_TOKENS`, inteira,
zero por padrão. Maior que zero, vai como `max_tokens` de toda requisição do
laço, nos dois dialetos, e o filho delegado herda.

E o corte deixa de ser silencioso. O decoder anthropic lê `stop_reason` do
`message_delta`, o openai lê `finish_reason: length`, e os dois marcam
`Truncated` no `EventDone`. O laço termina o turno em `StopMaxTokens` quando a
resposta cortada não tem chamada — a resposta fica no histórico, e o turno não
passa pela verificação de pronto, que leria uma resposta cortada como
terminada.

## Por que 32.000 para o modelo desconhecido

É o menor teto entre os Opus em uso, e todo modelo lançado desde `claude-opus-4`
aceita. Errar para cima é pior que errar para baixo: teto acima do que o modelo
aceita é requisição recusada antes de começar; teto abaixo é uma resposta que
para antes e agora diz que parou.

## Por que na família e não na interface

`Family` não ganhou método. O teto só existe porque um dialeto exige o campo, e
quem codifica para esse dialeto é a família — `Claude.Encode` passa
`OutputCeiling` para `encodeAnthropic`, `MiniMaxM3.Encode` passa o seu número.
Um método na interface obrigaria `generic` e `gemini` a responder uma pergunta
que o dialeto deles não faz.

## O que não mudou

Uma chamada cortada no meio dos argumentos continua virando erro
`tool_schema` (RN-8), que o modelo lê e corrige; a mensagem não diz que o
motivo foi o teto. E a TUI diz o mesmo texto para os dois tetos que terminam
em `StopMaxTokens` — o do turno e o da resposta.
