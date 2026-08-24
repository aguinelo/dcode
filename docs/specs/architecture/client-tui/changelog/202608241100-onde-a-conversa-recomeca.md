# Onde a conversa recomeça

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** recomendação de redesenho a partir da sessão real `1a030dd642cd50b9f68`

## O que mudou

Toda pergunta abre com uma régua.

Antes, a pergunta era um `>` no mesmo peso da prosa em volta. Na sessão real,
no meio da rolagem, não dava para ver onde um turno acabava e o outro começava —
e é a pergunta mais frequente que alguém faz de um histórico de conversa.

## Por que esta primeira

De todas as mudanças de desenho, é a que custa **zero colunas**. A coluna
lateral, o painel e a barra disputam largura com o texto; uma régua custa uma
linha por turno e devolve a estrutura que faltava.

É também a reclamação nomeada pela comunidade contra a TUI de referência da
categoria: separação visual clara entre turno da pessoa e turno do modelo.

## Régua e não cor

Pergunta destacada só por cor não está destacada num terminal sem cor, e este é
o ponto de referência para onde o olho rola. A régua é caractere, com forma
ASCII, como todo glifo desta família.

## Recuada, não de ponta a ponta

A primeira versão desenhava de borda a borda. Ela encostava nos dois divisores
de coluna e passava a ler como **linha de tabela** em vez de costura entre dois
turnos.

Recuar para a calha de duas colunas que todo o resto do fluxo usa resolveu, e
isso foi visto olhando o quadro renderizado a partir do log real — não deduzido
do código. É o mesmo método que achou os quatro defeitos anteriores, e a razão
de ele continuar sendo o método.

## A lacuna vem antes, nunca depois

A régua chama `gapBefore`, e o turno marca `blocked` para que a resposta ganhe
a própria lacuna na passada seguinte. É a convenção que o fluxo já tinha, e ela
existe porque branco no fim custa uma linha da janela, que está ancorada ali.

O primeiro turno não recebe lacuna acima: fluxo que abre em branco gasta uma das
poucas linhas que tem com nada.
