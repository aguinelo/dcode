## en

- **The `/loop` limit flags and the four `loop.*` keys leave the spec, which
  now follows the code.** At least MINOR: a declared configuration surface is
  removed. The spec declared `--max-iterations`, `--max-identical`,
  `--max-tokens`, `--max-stall` and `--done-timeout`, which `/loop` never
  accepted, and `DCODE_LOOP_SPEC_PATH`, `_SOURCE`, `_PROTECT` and
  `_SESSION_PREFIX`, which were in `KnownKeys` and read only by a function with
  no caller — one that turned an invalid source into `auto` in silence. The
  keys, that function and `SessionConfig` are gone; the only flag is
  `--protect`, and a loop's limits come from the family and the configuration
  like any session's.

## pt-BR

- **As flags de limite do `/loop` e as quatro chaves `loop.*` saem da spec,
  que agora segue o código.** No mínimo MINOR: sai uma superfície de
  configuração declarada. A spec declarava `--max-iterations`,
  `--max-identical`, `--max-tokens`, `--max-stall` e `--done-timeout`, que o
  `/loop` nunca aceitou, e `DCODE_LOOP_SPEC_PATH`, `_SOURCE`, `_PROTECT` e
  `_SESSION_PREFIX`, que estavam em `KnownKeys` e eram lidas só por uma função
  sem chamador — que transformava uma origem inválida em `auto` em silêncio.
  As chaves, essa função e o `SessionConfig` saíram; a única flag é
  `--protect`, e os limites de um loop vêm da família e da configuração como
  os de qualquer sessão.
