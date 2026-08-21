# O verbo da barra de atividade, e o tick que enfim para

**Data:** 2026-08-21
**Specs afetadas:** `202608081250-client-tui` (`.p`, `.config`)
**Fonte:** `refs/design/HANDOFF.md` (v5, §5)

## O que mudou

A linha de atividade ganha um gerúndio ao lado da ferramenta que roda —
`⏺ lendo grep \.Save\( 2,1s 1,4k tok ^C interrompe` —, sorteado do conjunto da
fase e trocando a cada 20 quadros, que são os 2,4 s do design no tick de 120 ms.

`DCODE_ACTIVITY_VERBS=0` desliga.

E o tick **para** quando a sessão fica ociosa.

## A regra que o arquivo inteiro existe para segurar

**O verbo nunca aparece sozinho.** Gerúndio com nada ao lado é movimento fingindo
informação: a tela parece viva e quem lê não aprende nada. Então ele só é
desenhado ao lado de uma ferramenta rodando, e sem ferramenta a linha diz a
palavra única que sempre disse, parada.

Isso também decide a tipografia: verbo em `dim`, fato em `bold`. O que se mexe é
o acompanhamento; o que é verdade é a ênfase.

## Um achado durante a implementação

`working` era, ao mesmo tempo, a palavra do estado sem ferramenta **e** um verbo
do conjunto `other`. Com a mesma string nos dois papéis, ninguém distingue verbo
girando de verbo parado — nem o leitor, nem o teste. O conjunto `other` passa a
ser `processando · organizando · anotando`, e nenhum verbo repete a palavra de
fallback.

A palavra de fallback também entrou no catálogo de idiomas. Ela era literal em
inglês dentro do `render.go`, e uma linha meio traduzida ao lado de verbos
traduzidos é pior que uma linha inteiramente em inglês.

## O tick

O `case tickMsg` já não avançava o quadro com a sessão ociosa, e o comentário
dizia por quê: *"tela ociosa que fica repintando queima bateria de laptop por
informação nenhuma."* Só que ele **reagendava assim mesmo** — a tela repintava
oito vezes por segundo para um número que não se movia. A frase estava certa; ela
apenas não estava sendo cumprida.

Agora o tick para, e volta quando um turno começa, com uma guarda (`ticking`)
para religar **exatamente um**. Sem ela todo evento acrescentaria um tick e o
contador de quadros dispararia — movimento dizendo que a máquina está mais
ocupada do que está.

Nada se perde parando: o `Now` é atualizado em **todo evento**, então o relógio
de onde um turno parte está fresco tenha ou não passado um tick.

## Por que catálogo próprio, e não campo em `Strings`

O guarda `TestEveryDeclaredLanguageCoversEveryString` lê cada campo com
`reflect.Value.String()`. Campo de slice volta não-vazio seja lá o que contenha,
então um catálogo de listas apoiado nesse guarda passaria **sem ser conferido**.

Tipo novo de entrada ganha guarda própria em vez de se esgueirar por um escrito
para outra coisa.

## O que não entrou

Crossfade de 420 ms na troca do verbo. No terminal não há opacidade, e o próprio
handoff diz que onde a maquete usa opacidade o terminal usa degrau de cor. Com
verbos de tamanhos diferentes, um degrau na troca só produziria tremor de
largura — e a primeira invariante desta família é que estilo nunca altera largura
medida.
