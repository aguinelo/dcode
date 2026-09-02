# Um tema que não pinta o chão

**Data:** 2026-09-01
**Specs afetadas:** `202608081250-client-tui` (`.p`, seções 7.3 e 10)
**Fonte:** pedido do usuário — "tenho pensado bastante sobre o visual do dcode e
preciso cada vez mais o claude, você acha que conseguimos chegar próximo? mesma
fonte, etc?"

## O que mudou

Há um quinto tema, `claude`, e ele é o primeiro **sem chão**: não pinta fundo,
não carrega um RGB sequer, e desenha a tela com o que o terminal já tem — três
pesos, itálico, e as dezesseis cores nomeadas.

## A fonte nunca foi do Claude

O pedido começa por "mesma fonte", e a fonte é o item que não custa nada: um
programa de terminal não escolhe fonte, tamanho, espaçamento nem itálico. Tudo
isso é do emulador. O dcode no mesmo terminal já usa a mesma fonte que o Claude
Code, com os mesmos itálicos e as mesmas ligaturas.

O que faz o Claude Code parecer o Claude Code é o resto, e o resto é pequeno:
**ele herda o fundo do terminal**, usa quase nenhuma cor, marca o que é
secundário com SGR 2 e o que é raciocínio com itálico, e reserva um verde para
um sinal só.

## O que o neon decidiu, este tema decide ao contrário

O `neon` afirmou, em 24 de agosto, que a interface possui as próprias cores,
fundo incluído, e que "âmbar sobre fundo desconhecido é uma cor; âmbar sobre
`#120d24` é um sinal". Continua verdadeiro, e continua sendo o tema em que esta
interface é desenhada.

Este tema toma a outra ponta da mesma decisão. Herdar o fundo é herdar **o
problema** que o neon resolveu pintando: um cinza cravado escolhido para tema
escuro é ilegível em tema claro. A saída não é escolher cinzas melhores — é não
escolher cinza nenhum:

- **Peso, não cor, para o texto.** Prosa em peso normal, rótulo em negrito,
  meta e dica e cromo em SGR 2. É a regra revogada pelo `neon` — *"papel de texto
  usa só os pesos que um terminal mantém sobre fundo desconhecido"* — e ela volta
  a valer **aqui**, porque aqui o fundo voltou a ser desconhecido.
- **Índice, não RGB, para os estados.** O que precisa de cor — ok, erro, aviso,
  acento, diff — usa as dezesseis cores ANSI nomeadas. Elas são exatamente as que
  o tema do terminal definiu para se lerem contra o próprio fundo. Um tema que
  herda o chão herda a paleta pelo mesmo caminho.
- **Itálico é um quarto peso.** `paint` ganha `italic` (SGR 3), o único atributo
  além dos três pesos que sobrevive a fundo desconhecido. O `claude` o dá ao
  raciocínio do modelo (seção 3.0): é a coisa na tela que é o modelo pensando
  alto, e é assim que o Claude Code a desenha. Os outros quatro temas não mudam
  por isso — o atributo existe na tabela, e a tabela de cada tema decide.

## O raciocínio ganhou papel

Para o itálico ter onde pousar, o corpo do raciocínio deixou de ser desenhado em
`StyleChrome` e passou a ter `StyleReasoning`. Nos quatro temas com chão o papel
é mapeado para **a mesma cor de antes**: nenhum pixel muda neles.

A calha à esquerda continua cromo. Ela é moldura, e uma barra vertical em
itálico é uma barra vertical torta — o papel é do pensamento, não do que o
emoldura.

Que ele seja apagado como cromo nos quatro temas é herança, não decisão: era
assim antes de existir papel para discutir. Agora existe, e mudá-lo é uma
decisão que se toma em uma linha.

## O guarda de contraste não tem o que medir, e diz outra coisa

`TestEveryRoleIsLegibleAgainstTheGround` mede WCAG contra `Ground`. Num tema sem
chão não há contra o que medir, e pular o teste seria um tema fora de todo
guarda. O guarda passa a afirmar, para tema sem chão, a condição que torna a
legibilidade **do terminal**: nenhum papel carrega RGB, em `fg` nem em `bg`. Um
RGB num tema sem chão é o cinza cravado voltando pela porta dos fundos, e é
exatamente o que o teste tem de pegar.

## O tipo diz qual cor é qual

`paint` carrega `fg`/`bg` (RGB) e `fgIdx`/`bgIdx` (índice), e não uma cor só com
duas grafias. Não é redundância: um RGB é uma cor **que este produto escolheu**,
e um índice é uma cor **que o terminal escolheu**. Só a segunda espécie pode ser
confiada a um fundo que este produto não escolheu, e é o tipo que impede a
confusão em vez de um comentário pedindo cuidado.

O zero de `ansi` é "nada", e por isso as dezesseis são deslocadas de um: preto é
uma cor que alguém pode querer, e papel não preenchido tem de desenhar sem cor.

## Profundidade

O `neon` diz "duas profundidades e não três": um terminal de dezesseis cores não
desenha violeta-acinzentado sobre violeta. Este tema é desenhável em **qualquer**
profundidade, dezesseis incluída, porque só usa as dezesseis. É o segundo motivo
para ele existir: é o tema que sobra quando o terminal não alcança os outros.

Cor desligada continua não recebendo nada. `Palette{}` não emite escape nenhum,
e o `claude` com cor desligada é a mesma tela monocromática que os outros.

## O que este tema não faz

Um tema é cor. Duas coisas que também fazem a semelhança **não** são cor, e
ficam para decisões próprias:

- **As marcas.** O Claude Code abre o turno com `●`, pendura o resultado com `⎿`
  e separa fatos com `·`. A tabela de glifos do dcode é indexada por Unicode/ASCII,
  não por tema, e misturar as duas é fazer o tema decidir forma. Se um conjunto de
  marcas vier, vem como conjunto, com o próprio fallback ASCII.
- **A densidade.** Sem coluna, sem moldura, resultado indentado sob a chamada.
  Isso é layout, e o layout do dcode foi medido em quatro larguras antes de ser
  o que é. Um tema não desfaz uma medida.

O nome diz o que ele copia. Não há por que fingir que é outra coisa.
