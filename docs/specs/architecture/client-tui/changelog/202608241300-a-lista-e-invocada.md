# A lista de conversas é invocada

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seções 7 e 10)
**Fonte:** medição sobre a sessão real `1a030dd642cd50b9f68`

## O que mudou

A lista de conversas saiu da coluna e virou sobreposição. `^R` a abre sobre o
fluxo, como o modal de aprovação já fazia.

## A convenção contradizia o uso

`^R` no readline é **busca invocada**: aparece, você escolhe, some. Foi essa
convenção que justificou tomar a tecla emprestada, no changelog de teclado.

E então nós a transformamos em vinte e seis colunas permanentes. A tecla
emprestada passou a contradizer aquilo de que foi emprestada.

## O que a coluna mostrava

Na sessão real, quatro linhas:

```
SESSÕES  ^r
  Write DCODE.md at the r…
● Write DCODE.md at the r…
  Write DCODE.md at the r…
  Write DCODE.md at the r…
```

Os títulos derivam da primeira pergunta, as quatro conversas começaram com a
mesma, e o corte caiu no mesmo lugar. Nada ali as distinguia — e isso ocupava
metade de uma coluna permanente para algo que se abre uma vez por tarde.

Na sobreposição sobram sessenta e quatro colunas, e o título inteiro cabe.

## `RailNav` não se moveu

O cursor que para nas duas pontas, o filtro que o puxa de volta ao topo, o modo
de nomear que toma o teclado, o `esc` em camadas — nada disso mudou, e nenhum
dos testes deles precisou mudar. **Só o desenho se mudou de lugar.**

É a propriedade que justifica a separação entre reduzir e desenhar: mover uma
lista de uma coluna para uma caixa não é uma mudança de comportamento, e o
diff mostra isso porque só toca no renderizador.

## Duas decisões da caixa

**Dez linhas, e depois diz quantas faltam.** Lista que enche o terminal esconde
a conversa sobre a qual foi invocada, e a décima primeira se alcança pelo filtro
— que é para isso que o filtro existe.

**A janela segue o cursor.** Cursor levado além da décima linha andaria para fora
de uma lista que parou de desenhar em dez.

## Duas coisas que a guarda pegou

O cursor era marcado na linha zero mesmo com a lista sem o teclado — herança de
quando a coluna era desenhada sempre. Agora ele exige `Nav.Active`.

E a linha de atalhos que eu acabei de escrever no catálogo trazia `·`. O guarda
de ASCII a pegou na primeira execução, o que é exatamente o que ele passou a
existir para fazer hoje de manhã.
