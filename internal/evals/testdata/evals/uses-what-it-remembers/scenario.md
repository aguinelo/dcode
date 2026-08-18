# uses-what-it-remembers

## O que mede

Que o agente **age** sobre uma memória que já está no prefixo, em vez de
redescobrir a mesma coisa ao mesmo custo.

## O material

`.dcode/memory.md` traz uma `gotcha`: não há gerador neste checkout, então campo
novo em `schema.yml` só existe depois que a função é acrescentada à mão em
`generated.go`. A tarefa pede **só** o campo no schema.

Tocar em `generated.go` é a memória sendo usada: nada mais no workspace pede
isso.

O bloco chega ao prefixo pelo **leitor do produto** — `memory.Read` e
`memory.Render` —, nunca por texto copiado para a fixture. Fixture que copia
texto do produto é fixture que diverge dele, e esta suíte já pagou isso quatro
vezes.

## O juiz

`CalledWith("edit", "generated.go")`.

## O que a primeira versão errou

Media `CalledWith("bash", "generate")`. A evidência mostrou o modelo lendo a
memória, **citando-a pelo nome** — *"Per the repo memory, run `make generate`
first"* — e indo procurar um alvo que a fixture nunca teve. As outras execuções
descobriam que o harness recusa shell e paravam de tentar.

Ou seja: o contrato era honrado e reprovado assim mesmo.

**Um contrato tem de ser honrável com as ferramentas que o harness de fato
permite**, ou mede o harness. Sem shell agora, e a ação medida é uma edição.

## Limiar

Declarado como ≥ 0% — que significa **"meça e me diga"**, não "qualquer coisa
serve". O primeiro número honesto vem da primeira medição, e limiar antes de
medição é limiar que a medição depois justifica.
