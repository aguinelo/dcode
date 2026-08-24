# A coluna lateral tem dois painéis

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seções 7 e 10)
**Fonte:** design `Coding Agent TUI v2`

## O que mudou

A lista de arquivos virou **coluna à direita com dois painéis**: o diff em cima,
a sessão embaixo. Legenda de raias no topo do fluxo, barra de navegação no
rodapé.

## O padrão foi invertido — de novo

Hoje de manhã a coluna passou a nascer escondida, por medida. À tarde ela volta
a aparecer sozinha acima de 120 colunas.

**A razão de esconder continua válida e não se aplica mais.** O que foi medido
foi que a lista de arquivos era uma segunda cópia do que o fluxo tinha acabado
de dizer — cada `⏺ write DCODE.md` seguido de um `✓ DCODE.md` três linhas ao
lado — e vinte e seis colunas é caro para uma repetição.

Estes dois painéis não são isso. Barra de adicionado contra removido, medidor
de contexto, quanto do que foi pedido a pessoa permitiu, as últimas chamadas
pelo relógio: nada disso está em outro lugar da tela. A objeção nunca foi
"coluna é cara", foi "**aquela** coluna não comprava nada".

O teste de largura foi escrito de três formas num dia só. Está dito no
comentário dele, em vez de editado em silêncio pela terceira vez.

## Duas coisas do design que não dá para desenhar honestamente

**A coluna M/A/D.** O design marca cada arquivo como modificado, adicionado ou
apagado. Uma ferramenta relata o que mudou, nunca se o caminho existia antes. A
primeira versão desenhava `A` quando não havia remoções — e rotulava igual um
arquivo de teste novo e um `append` num arquivo velho. Palpite vestido de fato,
numa coluna cuja função inteira é ser lida de relance. O glifo é o **estado**,
que a gente sabe.

**A barra contra o tamanho do arquivo.** O design desenha adicionado, removido e
o resto do arquivo. O resto do arquivo é o comprimento dele, que ninguém
reporta. A primeira versão usou a própria mudança como denominador, e toda
mudança sem remoção virou uma barra cheia: uma fileira de barras idênticas, cada
uma dizendo "100% do que fiz neste arquivo, fiz neste arquivo".

Contra a **maior mudança do turno** as barras dizem algo verdadeiro e útil:
quais arquivos este turno realmente atacou. E o painel **diz qual é a escala** —
barra cuja escala ninguém vê é barra que vai ser lida como porcentagem.

## A legenda das raias

Uma legenda vale a linha quando o que ela explica é um **caractere** cujo
significado não se adivinha. `▏` e `╎` não dizem nada sozinhos, e quem não foi
avisado lê como enfeite e para de ver.

Uma linha só, no topo, e só quando a tela está fazendo mais de uma raia. Por
causa dela a raia `você` ganhou glifo de volta: contra três nomes, um branco não
é marca que alguém procure.

## A barra de navegação

`NAV` com as teclas ao lado, como o design pede — mas **só as que são teclas**.
O rodapé do design também oferece `j/k mover` e `t tema`, que são letras, e
letra numa linha em que se digita é o defeito corrigido duas vezes hoje. Elas
pertencem a um modo que tome o teclado — que o próprio design sugere ao pôr um
badge NAV ali — e enquanto esse modo não existe, anunciá-las seria anunciar
teclas que comem o que você está digitando.

É o **primeiro segmento a cair** quando a barra fica sem espaço: toda tecla que
ele nomeia se alcança pelo `?`, o que o torna a coisa mais reconstruível da
linha.

## Duas guardas minhas me pegaram

A paleta na barra fazia **a cor mudar o que a tela diz** — o segmento só existia
com cor ligada, e `TestColourNeverChangesWhatIsOnTheScreen` apontou. E o olho do
mascote deixou de ser "o único terracota da interface", que é regra escrita.

As duas apontam para a mesma coisa: uma tira de amostras que mostra um tema
imutável é enfeite. Ela volta quando houver tecla que troque o tema.
