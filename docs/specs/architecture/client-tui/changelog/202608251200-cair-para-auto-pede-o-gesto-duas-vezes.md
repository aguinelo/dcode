# Cair para `auto` pede o gesto duas vezes

`/mode` mostra o modo em vigor ou troca para `plan`, `assist` ou `auto`;
`shift+tab` cicla os três; a barra ganha o crachá, discreto em `plan` e
`assist`, em cor de aviso em `auto`.

O que precisa ficar escrito é o **armamento**, porque é a única parte que não
sai da leitura do código.

Entrar em `auto` retira a fronteira. Os dois caminhos até lá mostram o mesmo
aviso e exigem a repetição do gesto — e é *o mesmo* aviso porque as duas
primeiras versões avisavam diferente: o comando escrevia a nota na transcrição
**e** no rodapé, a tecla só no rodapé. Duas maneiras de avisar sobre a mesma
coisa são duas maneiras de divergir; agora ambos passam por `armAuto`.

O aviso vive no rodapé e não na transcrição, como o segundo `^C`: ele pertence à
tela enquanto a decisão está pendente, não ao registro do que aconteceu. Nada
aconteceu ainda — é justamente esse o ponto.

O armamento é **por cliente e definitivo**, e isso difere do `^C` de propósito.
Lá, o armado dura exatamente enquanto o aviso está na tela, porque o risco é uma
segunda batida acidental. Aqui o risco é a primeira decisão, e uma vez tomada,
perguntar de novo a cada troca ensinaria a confirmar sem ler. O comentário no
`Model` dizia que o armamento era desfeito ao sair de `auto`; nada nunca fez
isso, e quem estava errado era o comentário — a frase que a pessoa lê promete
exatamente o comportamento que o código tem.

Modo que **mantém** a fronteira passa de primeira. Armar `plan` seria treinar o
gesto de confirmar em um caso onde não há nada a perder, que é como se ensina
alguém a atravessar o caso onde há.

Duas correções de honestidade vieram junto, e são a mesma que o daemon levou:
`/mode` sem argumento respondia "modo atual: assist" sempre que não sabia — o
limite sem nome agora é dito como tal, com as três opções. E a recusa de um nome
inválido estava em inglês cravado no meio do código; passou ao catálogo, como
todo texto que a pessoa lê.
