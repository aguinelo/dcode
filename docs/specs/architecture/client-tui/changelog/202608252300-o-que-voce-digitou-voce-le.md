# O que você digitou, você lê

`!ls -la` desenhava uma linha, `exit 0`, e mais nada. A saída não estava
perdida: chegava inteira ao cliente, ficava guardada na entrada, e reaparecia
com `esc`, `↑`, `tab`. Isso é pior que perder — a tela respondia um pedido de
**ver** com um código de status, e parecia certa fazendo isso.

A regra era `Expanded = !d.OK`, com o comentário *"errors open, successes stay
collapsed: failure needs attention, success needs only confirmation"*. Ela está
certa, e continua valendo, **para a chamada que o modelo faz**: ali a saída é
meio. Ele roda `ls` para se orientar e depois diz o que importava, e abrir tudo
enterraria justamente a prosa que carrega o ponto.

Um comando digitado não tem prosa depois. A saída **é** o ponto — ninguém
escreve `ls -la` para saber que terminou com zero. A regra foi desenhada num
contexto e aplicada noutro, que é a forma de defeito mais difícil de ver: nada
está errado no código, só está no lugar errado.

A origem viaja no evento, como `ToolRequested.Typed`. A alternativa era inferir
do formato do id da chamada — o `exec` monta `<turnID>-1` — e isso seria ler uma
coincidência como se fosse um fato: **o formato de um id não diz quem quis a
chamada.**

Junto saiu uma duplicação que só apareceu depois. O `bash` prefixa a saída com
`exit N` porque o modelo lê a saída como texto e precisa do código ali dentro; a
linha já mostra esse mesmo código na coluna dela. Recolhido, ninguém via.
Aberto, a entrada dizia `exit 0` duas vezes. A primeira linha do corpo é
descartada quando é **exatamente** o resumo ao lado — só exatamente, porque uma
linha que diz algo a mais é uma linha que alguém quis.

O teste pergunta ao **quadro desenhado**, não ao campo `Expanded`. Um teste que
assertasse o campo passaria numa build em que o renderizador o ignora, e o
defeito que se está consertando é precisamente a distância entre "o cliente tem
o dado" e "a pessoa vê o dado".
