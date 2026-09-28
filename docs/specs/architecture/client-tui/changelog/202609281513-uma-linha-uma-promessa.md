# Uma linha, uma promessa

**Data:** 2026-09-28
**Specs afetadas:** `202608081250-client-tui` (`.p`, invariantes)

## O que mudou

**Quinze linhas eram reivindicadas por mais de um fragmento**, cada fragmento com
o seu teste, e viraram uma linha por promessa: a chamada ainda chegando;
progresso e resultado; o build na barra; o cursor da lista; o título cortado; o
caminho que ninguém tocou; a coluna escondida, a visível e a vazia (três); a
coluna que aparece sozinha; o bloco e o que vem depois dele; a linha de quem
retoma; a moldura; a régua; o tick; a linha selecionada do seletor; o sandbox no
topo.

O texto é o mesmo, dividido. Onde a divisão deixava uma metade sem sujeito, ela
ganhou o sujeito — "Coluna visível…", "O corte do título…", "Trabalho recomeçando
religa exatamente um tick." — e, quando o fragmento dela no mapeamento deixava de
casar, o fragmento mudou junto.

## Uma reivindicação que nunca casou

"Nenhuma linha da coluna ultrapassa a largura dela" foi escrita no #241 junto do
fragmento `não ultrapassa a largura dela`, que nunca esteve em linha nenhuma: a
linha diz "Nenhuma linha da coluna **ultrapassa**". `TestNoSidebarRowOverflowsTheColumn`
existe desde então e a guarda nunca o procurou. A linha virou duas — a outra
metade é a da cor — e o fragmento passou a ser `Nenhuma linha da coluna ultrapassa`.

## Duas reivindicações sobre nada

- `A coluna **nasce escondida**` → `TestTheSidebarStartsHidden`: linha do #259,
  substituída em `202608241930-a-coluna-lateral-v2`.
- `abaixo de 100 colunas ela some` → `TestTheSidebarDisappearsOnANarrowTerminal`:
  linha do #241, substituída no #259.

As duas linhas deram lugar a comportamento novo e os testes saíram com elas. Os
fragmentos ficaram no mapeamento, sem casar com linha nenhuma e nomeando testes
que não existem mais. Saem.

## Uma linha que volta

"Turno que não tocou nada não abre coluna." foi escrita no #241, reescrita no
#259 ("turno que não tocou nada não a abre") e perdida em
`202608241930-a-coluna-lateral-v2`, sem uma palavra naquele changelog. O código
não a perdeu: sem conteúdo, `ShowRail` devolve `false` antes de olhar largura ou
escolha, e `TestATurnThatTouchedNothingGetsNoSidebar` continuou asserindo isso.
Volta com o texto do #241, reivindicada pelo fragmento que sempre esteve no
mapeamento.

A linha de cima diz que a escolha explícita vence nos dois sentidos, e o código
dá precedência a esta: sem conteúdo, nem a escolha abre a coluna. As duas
conviviam assim na linha do #241; esta mudança não mexe nisso.

## A moldura e a caixa

`desenha exatamente as linhas que o layout reservou` casa com a linha da moldura
e com a da caixa de entrada. Com a moldura dividida, as duas linhas que esse
fragmento casa ficam com `TestTheFrameReservesExactlyWhatTheBoxDraws`, e a da cor
com `TestTheFrameIsTheSameShapeWithAndWithoutColour`. Um teste reivindicar duas
linhas é permitido; uma linha ter duas reivindicações, não.

## Por quê

O `specguard` reivindicava cada linha de invariante pelo primeiro fragmento do
mapeamento que ela contivesse, percorrendo um `map` — e o Go sorteia a ordem de
um `map` a cada `range`. Em cada uma daquelas quinze linhas só um dos testes era
procurado por execução, e qual deles era sorteio: renomear o outro deixava a
guarda vermelha numa fração das execuções e verde no resto. E um fragmento que
não casava com linha nenhuma não era achado de ninguém, que é como as quatro
reivindicações acima passaram semanas sem ser vistas.

A guarda agora exige que **cada linha seja reivindicada por exatamente um
fragmento**, e reprova o fragmento que não reivindica linha nenhuma. A decisão
inteira, com a medição, está no changelog de mesmo nome do `provider-adapter`,
onde o defeito foi achado.
