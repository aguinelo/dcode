# O crachá basta; a confirmação não

A 0.7.0 saiu ontem exigindo o gesto duas vezes para entrar em `auto`, e o
raciocínio está registrado em
[`202608251200`](202608251200-cair-para-auto-pede-o-gesto-duas-vezes.md).
Ele durou até o primeiro uso.

O argumento era a simetria com o `^C`: lá a primeira batida pode ser reflexo, e
o segundo toque distingue reflexo de intenção. A simetria é falsa. **`/mode auto`
são onze caracteres deliberados** — não há reflexo a desambiguar. Pedir que a
pessoa repita o que acabou de digitar não é salvaguarda; é um degrau que se
aprende a pular, e um degrau que se aprende a pular é pior que degrau nenhum,
porque treina o gesto de confirmar sem ler.

O `shift+tab` tinha um caso melhor — tecla pode ser acidental. Mas manter o
armamento só ali reintroduz a divergência entre caminhos que o próprio
`202608251200` tinha acabado de eliminar, e o custo de um `shift+tab` acidental
é uma tecla a mais para desfazer, com o crachá na barra dizendo o que aconteceu.

O que fica no lugar é o que já estava lá e é melhor: **o crachá em cor de aviso,
que diz que não há fronteira enquanto isso for verdade.** Uma confirmação fala
uma vez, a quem já decidiu, e cala para sempre. Um estado visível fala o tempo
todo, para quem chegou depois, e não pode ser aprendido a ignorar sem que a
pessoa esteja olhando exatamente para a informação que importa.

O `Model` perde `AutoArmed` e o catálogo perde `AutoConfirmOnce`. O invariante
passa a ser o oposto do que era: **todo modo alcança o daemon na primeira, pelos
dois caminhos**, e nada fica na tela pedindo repetição.
