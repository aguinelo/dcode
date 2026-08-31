# O critério que não dizia o que era

**Data:** 2026-08-31
**Specs afetadas:** `202608281900-failure-feedback` ganha a RN-3 e três
invariantes. Achado rodando o produto, não lendo o código.

> **Estado.** Corrigido e medido. `fixes-what-the-output-named` seguiu em 100%
> de 20, e `finishes-work-that-takes-more-than-one-cycle` foi de **95% para
> 100%** — o único cenário da suíte com critérios que uma pessoa escreveria.

## Como apareceu

Um teste de ponta a ponta com o binário instalado, contra um workspace de
verdade, com um `done.toml` de duas linhas:

```toml
[tests]
command = "go test ./..."

[changelog]
command = "test -f CHANGELOG.md"
```

A tarefa pedia só o conserto de uma função. O laço rodou os critérios, o
`changelog` estava vermelho, e o modelo continuou — o que é o laço funcionando.

O que ele fez em seguida é o achado:

```
⏺ bash ls -la .dcode/ && find .dcode -type f
⏺ read .dcode/done.toml
```

**Foi descobrir o que `changelog` queria dizer.** Duas rodadas e duas chamadas
de ferramenta atrás de uma informação que o laço tinha em mãos: foi ele que
executou o comando.

## Por que a família não pegava isto

`test -f` não imprime nada quando falha. Saída vazia, bloco não renderiza, e o
lembrete volta a ser o que era antes desta família existir:

> *"You changed files and this is not done yet: changelog did not pass."*

O nome, e nada. A RN-1 entrega evidência quando há evidência, e uma classe
inteira de critério — teste de existência, comparação de saída silenciosa,
qualquer coisa que decida por código de saída — não tem nenhuma.

## A regra, e o que ela recusa

Sem saída, o lembrete **nomeia o comando**:

```
changelog:
  (it printed nothing) test -f CHANGELOG.md
```

**O comando é identidade, nunca evidência.** Ele diz o que o critério é, não o
que aconteceu, e por isso só entra quando não há o que mostrar. Saída existente
vence — mostrar os dois sempre gastaria contexto repetindo o que o nome já
sugere na maioria dos casos, e há teste recusando essa versão.

## O que este achado diz sobre o método

As duas famílias anteriores foram desenhadas lendo o código e medidas contra
cenários que eu mesmo escrevi. Este defeito não apareceu em nenhum dos dois
lugares: os cenários da suíte usam critérios que **imprimem**, porque foi assim
que eu os escrevi.

Precisou de um `done.toml` que uma pessoa escreveria — duas linhas, uma delas um
`test -f` — para a lacuna aparecer. Fica registrado que **rodar o binário contra
um workspace de verdade encontrou o que a suíte de eval não encontrava**, e que
isso não é falha da suíte: é o limite de qualquer cenário escrito por quem
também escreveu o código.
