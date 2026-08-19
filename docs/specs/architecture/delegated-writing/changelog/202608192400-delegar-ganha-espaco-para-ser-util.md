# Delegar ganha espaço para ser útil

Decisão do autor: **liberar o potencial agora, controlar custo quando custo for
problema.** Este changelog registra o que foi liberado, e — o que importa mais —
o que **não** foi, e por quê.

## O que estava travando

Duas rodadas medidas, o caso mais fácil possível de trabalho divisível, e
**zero delegações nas duas**. O log da segunda não menciona a ferramenta uma
única vez: ele nem cogitou.

A descrição dizia:

> *"Do not use it for something you have already read … a delegated turn costs
> more than one read."*

Escrito quando o filho **só lia**, e a única coisa que delegar economizava era
leitura. Com `owns`, a unidade delegada virou "leia isto **e** escreva aquilo" —
e ler primeiro é como todo trabalho de escrita começa. Um modelo seguindo a
descrição corretamente nunca delegaria escrita nenhuma.

## O que foi liberado

**A descrição deixa de desaconselhar o caso que a feature existe para servir.**
Passa a dizer para pegar a ferramenta sempre que a tarefa se divide em pedaços
independentes — um filho por pedaço, numa mensagem só, para correrem lado a
lado — e diz com todas as letras que **ter lido não é motivo para segurar o
trabalho**: a leitura é a parte barata, e um filho que relê no contexto dele não
custa nada além da resposta.

**O teto do filho vai de 20 para 100 iterações.** 20 foi dimensionado para quem
responde uma pergunta. Quem lê um pacote e escreve uma nota faz mais que
responder, e 20 trunca isso antes de começar.

**O teto do relatório vai de 8 KB para 32 KB.** O teto é sobre a **resposta**,
não sobre o trabalho.

## O que NÃO foi liberado, e a distinção importa

Liberdade aqui é sobre **custo**, não sobre **fronteira**. Duas coisas ficaram
fechadas, e nenhuma das duas é conservadorismo de gasto:

**`bash` num filho que escreve.** Comando de shell é opaco — o scheduler já roda
um sozinho por isso —, então contenção estreitada a `owns` não teria o que
conter: o filho escreveria fora do que possui pelo shell, e a propriedade
declarada viraria papel. Isso não é custo, é a fronteira inteira.

**Aninhamento.** Continua impossível por ausência da ferramenta no registro do
filho. Não é o custo exponencial que decide — é que o erro passa a aterrissar
longe da causa, e este projeto já paga caro por diagnóstico difícil.

## Um teste foi reescrito, e o motivo fica aqui

`TestExploreDescribesWhenNotToUseIt` exigia que a descrição avisasse sobre
"already read" e "single known file". Os dois avisos eram a decisão antiga
escrita em asserção.

O modo de falha que o teste protege **não sumiu, mudou de lugar**: delegar tudo
continua sendo o erro barato, e o caso a segurar passou a ser trabalho que tem de
concordar consigo mesmo. O teste passou a cobrar isso, e a cobrar que "already
read" **não** apareça.

## O que ainda não se sabe

Se agora ele delega. Duas rodadas mediram duas descrições diferentes e deram
zero. Esta é a terceira formulação, e a pergunta só tem resposta honesta pelo
contrato `delegates-writing-when-work-is-disjoint` da §6 do `.p`, medido em N
execuções — não por mais uma rodada e mais uma reescrita de prosa.
