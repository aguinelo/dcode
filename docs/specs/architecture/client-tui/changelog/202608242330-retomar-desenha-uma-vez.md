# Retomar desenha uma vez

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** relato de quem usa — "no `dcode -c` fica carregando um monte de coisa
e a tela não para de rolar"

## O que mudou

Enquanto o histórico de uma conversa continuada está chegando, a tela mostra
**uma linha**. A conversa é desenhada uma vez, quando ela alcança.

## Por que rolava

Continuar escreve o log antigo inteiro dentro da sessão nova
(`internal/app/daemon.go:132`), então o cliente assina de `seq 1` e recebe todos
os eventos de novo — **3544**, numa sessão real desta máquina.

Cada um chega como sua própria mensagem, e o Bubble Tea desenha depois de cada
mensagem. Retomar repintava a tela três mil quinhentas e quarenta e quatro
vezes, com a janela seguindo o próprio fim. É essa a tela que não parava.

## Uma linha, e não nada

A alternativa barata seria não desenhar. Mas quem pediu para continuar uma
conversa e recebeu um terminal vazio não tem como distinguir "carregando" de
"perdeu tudo" — e depois de um dia inteiro deste produto, é essa a leitura para
a qual a pessoa vai.

A linha diz o que está fazendo e **quantas linhas já leu**. Fiapo sozinho
responde que está trabalhando e não responde quanto falta; numa sessão de três
mil e meio de eventos, é a segunda que impede alguém de apertar uma tecla.

## E ela se move

O tique para quando a sessão está parada — e uma sessão lendo histórico **está**
parada, nada está rodando. Sem tratar isso, o fiapo congelava numa tela que diz
"lendo", que é exatamente como uma tela travada se parece.

## Onde o limite mora

`Options.Backlog` é o `LastSeq` que a resposta de criação de sessão já devolve.
A borda o injeta, como injeta a janela e a língua; o cliente não vai perguntar.
Tudo até ali é história, e o que vem depois está acontecendo.

Isto é a metade barata do que foi pedido. Assinar do fim e carregar para trás
sob demanda vem em seguida e mexe na assinatura.
