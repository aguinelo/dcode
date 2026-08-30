# O laço não sabia voltar

**Data:** 2026-08-30
**Specs afetadas:** nasce `202608301200-recoverable-cycle` com `.r`. A
`failure-feedback` deixa de listar isto como fora de escopo quando o `.p` desta
existir.

> **Estado.** Só `.r`. Sem `.p`, sem código, sem invariante. Contratos
> comportamentais: **nenhum**, e a §2 diz por quê.

## De onde veio

De uma pergunta sobre checkpoints no ciclo longo, e da constatação de que o laço
é **fechado na detecção e aberto na recuperação**: ele sabe dizer que piorou e
não sabe voltar.

```go
if Progressed(*unmet, now) || *unmet == nil {
    *stall = 0
} else {
    *stall++
}
```

`Progressed` devolve um booleano onde cabem três respostas. Empatar, regredir e
trocar uma falha por outra colapsam em `stall++`, e a informação de que o ciclo
**piorou** morre nessa linha.

## A objeção que caiu

A `.r` da `failure-feedback` pôs isto fora de escopo por acreditar que exigia
atravessar uma fronteira declarada — *o `vcs` deste produto lê e não escreve*.

**O ponto de retorno não precisa ser um commit.** O laço já sabe quais caminhos a
sessão escreveu, e um instantâneo do conteúdo deles não commita, não indexa e
não move branch. A fronteira fica inteira e a única objeção real desaparece.

Vale registrar como uma decisão de escopo se sustentou por uma premissa que
ninguém tinha conferido. A frase estava certa — git é do usuário — e a conclusão
não vinha dela.

## As duas regras que decidem o desenho

**Voltar é decisão do laço, nunca do modelo (RN-3).** Não vai existir ferramenta
de desfazer. Um agente que pode reverter o próprio trabalho pode reverter a
**evidência**, e apagar o que ficou vermelho é a saída mais limpa de um laço que
só termina quando o vermelho acaba. Mesma razão pela qual `done_propose` não
existe num turno de trabalho.

**Só se volta de regressão nomeada, nunca de "não progrediu" (RN-4).** Empatar é
comum e legítimo — um ciclo que leu, entendeu e não fechou critério nenhum não
estragou nada, e revertê-lo joga fora trabalho bom. O que justifica voltar é um
critério que **passava e parou de passar**, e isso o `Report` já sabe apurar.

## O limite, declarado antes de construir

**O `bash` escreve fora do instantâneo.** `State.Written()` sabe de `write` e
`edit`; um comando de shell que gera código, roda um formatador ou apaga um
diretório não passa por ali. Restaurar sobre uma árvore que mudou por fora pode
produzir um estado que nunca existiu — metade do ciclo desfeita, metade não.

Não é contornável sem vigiar o filesystem inteiro, que é máquina que este produto
não quer e que erraria em silêncio. A resposta é **declarar a cobertura** e não
prometer mais: o ponto de retorno é sobre o que o agente escreveu com as
ferramentas.

## O que isto destrava

É o primeiro dos quatro passos recomendados, e é primeiro porque os outros três
dependem dele:

| | passo | por que depende |
|---|---|---|
| 1 | ponto de retorno | — |
| 2 | contrato que mede correção | independente, mas mede 3 e 4 |
| 3 | progresso por aproximação | só paga quando "piorou" tem consequência |
| 4 | subir o teto de ciclos parados | mais ciclos irreversíveis é mais dano |
