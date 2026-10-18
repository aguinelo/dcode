## en

- **A criterion's output is cut on a whole character.** With no line break in
  the last 2000 bytes, the tail was cut on the byte, and when that byte fell
  inside a multibyte character the model got invalid UTF-8 at the head of the
  text — half of an "é". The cut now moves forward to the first byte that
  starts a character, so the ceiling still holds and at most three bytes more
  are dropped.

## pt-BR

- **A saída de um critério é cortada num caractere inteiro.** Sem quebra de
  linha nos últimos 2000 bytes, o fim era cortado no byte, e quando esse byte
  caía dentro de um caractere de vários bytes o modelo recebia UTF-8 inválido
  na cabeça do texto — meio "é". O corte passa a avançar até o primeiro byte
  que começa um caractere, de modo que o teto continua valendo e no máximo
  três bytes a mais se perdem.
