# Os aliases sem data têm o teto publicado

**Data:** 2026-10-17
**Specs afetadas:** `202608072334-provider-adapter` (`.p`, seção 2.1)
**Fonte:** a lacuna que `202610171700-a-spec-alcanca-o-codigo.md` registrou
sem corrigir.

## O que mudou

`claude-sonnet-4` e `claude-opus-4`, sem data nem `-0`, não casavam com
nenhuma linha da tabela por prefixo e caíam nos defaults (32.000 de teto,
200.000 de janela). Agora `claudeAliases`
(`internal/provider/claude_models.go`) leva cada um à linha do modelo que
nomeia — `claude-sonnet-4-0` e `claude-opus-4-0`, que são
`claude-sonnet-4-20250514` e `claude-opus-4-20250514` — antes da busca por
prefixo:

- `claude-sonnet-4` — 64.000 de teto (antes 32.000), 200.000 de janela;
- `claude-opus-4` — 32.000 de teto e 200.000 de janela, os mesmos de antes,
  agora por ser o modelo e não por coincidir com o default.

## Por que os valores vêm da linha irmã

A referência de modelos da Anthropic que consultamos lista o alias `-0` e o
id datado de cada um, mas não repete os limites dos modelos depreciados. Os
números são os que a tabela já dava às linhas `-0` e datadas, que nomeiam o
mesmo modelo; o alias não traz número novo.

## Por que nome exato, e não prefixo

Como prefixo, `claude-sonnet-4` casaria com toda `claude-sonnet-4-x` que a
tabela ainda não tem e daria a ela 64.000. Teto acima do aceito é requisição
que a API recusa; o desconhecido continua recebendo `claudeDefaultOutput`.
