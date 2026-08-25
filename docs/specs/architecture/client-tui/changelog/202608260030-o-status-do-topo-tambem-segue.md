# O status do topo também segue

O crachá aprendia o modo novo e o status do topo não. Uma sessão em `auto`
seguia anunciando `workspace-write` — e a §2.1 chama justamente esse campo de
**"o único em que estar errado é perigoso"**, motivo pelo qual ele está fora da
ordem de descarte da barra.

O topo anunciava um limite que tinha acabado de ser retirado. É o pior sentido
possível para esse campo errar: quem lê `workspace-write` age como quem tem uma
fronteira, e não tinha nenhuma.

`SessionModeChanged` passa a carregar `sandbox_mode`. Carregado, e **não**
derivado pelo cliente a partir do nome do modo: a tabela que liga nome a par
tem uma casa só, no `sandbox-policy`, e um cliente que a recalculasse seria uma
segunda cópia — que é exatamente a coisa que este dia inteiro vem consertando.
Anúncio sem `sandbox_mode` deixa o valor anterior de pé, porque uma barra que se
apaga quando o daemon cala é pior que uma que mantém a última coisa que soube.

Quarta aparição do mesmo defeito em um dia, e vale contar a série inteira porque
o padrão é mais útil que qualquer um dos quatro:

1. o **nome** do modo guardado ao lado do par, em vez de derivado dele;
2. o **par** lido fora da trava que o escreve;
3. a **fronteira do SO** copiada de um par que ainda ia mudar;
4. o **sandbox no topo** copiado de um estado que ainda ia mudar.

Sempre a mesma forma: um valor copiado de uma verdade que se move. E sempre o
mesmo sintoma — a tela afirmando com confiança algo que deixou de ser verdade,
o que é pior que não afirmar nada, porque convida a agir.
