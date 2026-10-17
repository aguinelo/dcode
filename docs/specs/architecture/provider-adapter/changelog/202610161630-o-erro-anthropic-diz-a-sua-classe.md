# O erro Anthropic diz a sua classe

**Data:** 2026-10-16
**Specs afetadas:** `202608072334-provider-adapter` (`.p`, seções 4 e 7)
**Fonte:** leitura de como as falhas da Anthropic chegam ao laço, contra o mapa
de decisão da seção 4.

## O que mudou

Duas falhas da Anthropic chegavam ao laço com a classe errada:

- **Prompt longo demais.** A Anthropic responde 400 `invalid_request_error`,
  com `prompt is too long: N tokens > M maximum` (ou `input length and
  max_tokens exceed context limit`) na mensagem — nunca 413. Só o 413 virava
  `context_size`; o 400 virava `bad_request` e abortava o turno. Agora
  `ClassifyStatus` reconhece as duas frases num 400 e devolve `context_size`,
  que força compactação e repete.
- **Erro dentro do stream.** Um frame `type: "error"` (por exemplo
  `overloaded_error`) que chega depois do 200 era ignorado pelo decodificador.
  O stream terminava sem evento terminal, classificado `transport`, e a
  mensagem do provedor se perdia. Agora o frame vira `ProviderError` com a
  classe do seu par HTTP e a mensagem do provedor (sanitizada):
  `overloaded_error`, `api_error` e tipo desconhecido → `provider`;
  `rate_limit_error` → `rate_limit`; `authentication_error` e
  `permission_error` → `auth`; `billing_error` → `quota`;
  `request_too_large` → `context_size`; `invalid_request_error` →
  `bad_request`, ou `context_size` quando diz que o prompt não cabe.

## Por que importa

`context_size` é a única falha que a compactação existe para resolver, e era
justamente a que não chegava a ela: a sessão longa morria no ponto em que um
resumo a teria salvo. No stream, o erro como `transport` repetia por acaso o
que devia repetir e por engano o que não devia — uma credencial recusada ou
um pedido malformado voltavam ao provedor até o teto —, sempre sem dizer o que
o provedor tinha dito.

## Por que frases, e não "todo 400"

Todo outro 400 continua sendo bug nosso e precisa aparecer como tal.
Compactar e repetir um pedido malformado o esconderia atrás de um contexto
menor. A correspondência é por frase, sem diferenciar maiúsculas, e qualquer
400 que não diga que não cabe continua `bad_request`.

## Por que `overloaded_error` vai para `provider`

É a forma, dentro do stream, do 529 — que `ClassifyStatus` já manda para
`provider`, repetido com recuo exponencial. Dar a ele outra classe faria a
mesma sobrecarga ser tratada de dois jeitos conforme o momento em que chegou.
