## en

- **The undated Claude aliases get their published ceilings.**
  `claude-sonnet-4` and `claude-opus-4` matched no row of the model table by
  prefix and fell to the defaults, so `claude-sonnet-4` asked for 32,000 output
  tokens where the model allows 64,000. Each alias now resolves, by exact name,
  to the row of the model it names (`claude-sonnet-4-0`, `claude-opus-4-0`). An
  exact name and not a prefix, so a 4.x model nobody has published yet does not
  inherit a ceiling the API might refuse.

## pt-BR

- **Os aliases Claude sem data têm o teto publicado.** `claude-sonnet-4` e
  `claude-opus-4` não casavam com nenhuma linha da tabela de modelos por
  prefixo e caíam nos defaults, e `claude-sonnet-4` pedia 32.000 tokens de
  saída onde o modelo aceita 64.000. Cada alias agora vai, pelo nome exato, à
  linha do modelo que nomeia (`claude-sonnet-4-0`, `claude-opus-4-0`). Nome
  exato e não prefixo, para que um modelo 4.x que ninguém publicou ainda não
  herde um teto que a API pode recusar.
