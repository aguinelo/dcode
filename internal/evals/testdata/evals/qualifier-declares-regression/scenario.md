# qualifier-declares-regression

**Contrato:** `202608261730-done-qualifier.p.spec.md` · limiar **90%**

Uma spec que **acrescenta** algo a um tipo que já funciona, numa base com suíte.
Pelo menos um critério tem de ser declarado `pass`: o que existe hoje continua
funcionando.

## Por que o cenário conserta `stats.go`

O workspace compartilhado traz `Rows()` com um erro de um a menos, e o teste ao
lado diz isso. Numa base cuja suíte está vermelha, um modelo que se recusa a
chamá-la de guarda de regressão está **certo** — e o contrato mediria a
honestidade dele em vez do que diz medir. O cenário sobrepõe `stats.go` com a
correção de um caractere para que a premissa da linha da tabela seja verdadeira.

## O que se mede

Que a proposta que fica de pé tenha ao menos um critério `expects: pass`. Um
conjunto todo vermelho diz o que o trabalho precisa acrescentar e nada sobre o
que ele não pode quebrar.

## O que este cenário ainda NÃO pega

Se o critério de regressão escolhido é **bom**. `go test ./...` e `go build
./...` passam os dois no juiz, e um deles prova bem menos que o outro.
