# `^R`: a coluna toma o teclado

**Data:** 2026-08-21
**Specs afetadas:** `202608081250-client-tui` (`.p`, seções 7 e 10)
**Fonte:** `refs/design/HANDOFF.md` (v5, §2 — modo *navegando*)

## O que mudou

`^R` foca a lista de conversas. Enquanto ela está aberta, **a coluna é dona do
teclado**: `↑↓` movem, letra filtra, `enter` continua a conversa sob o cursor,
`esc` limpa o filtro e depois fecha.

Faltava o segundo dos três modos que o design nomeia. O terceiro, nomear,
continua no backlog por não ter onde ser guardado.

## Dona do teclado não é floreio

Lista que se percorre com teclas que também digitam na linha de entrada é lista
em que toda tecla faz duas coisas — e a única vez em que ela faz a errada, abre a
tarde de outra pessoa.

Por isso o bloco fica **acima** do guarda do menu de autocompletar, e não dentro
dele. O changelog `202608150200` registra o custo de errar isso: o bloco da cópia
morava dentro daquele guarda, o menu só abre depois de algo ser digitado, a cópia
só abre com a linha vazia, e as duas condições nunca eram verdadeiras juntas. O
modo era decorativo e nada dizia.

## Decisões pequenas, cada uma com o motivo

**Letra é filtro, não atalho.** É exatamente o caso que a RN-16 deixa aberto: a
regra é sobre linha em que se digita, e aqui não se digita — a cópia e o modal de
aprovação já usam letras pelo mesmo motivo.

**`↑↓` não dão a volta.** O `Picker.Move` já escrevia o porquê e é reusado:
*"dar a volta transformaria 'um a mais' em 'outro lugar completamente', e esta é
uma lista onde parar em outro lugar abre o trabalho da tarde errada."*

**O cursor é caractere, não cor.** Linha destacada só por cor não está destacada
num terminal sem cor. E ele vence a marca de conversa aberta, porque com o
teclado na coluna a pergunta na tela é "qual vou abrir", não "em qual estou".

**`esc` recua uma coisa por vez** — filtro, depois modo. É a mesma escada que o
`esc` já tem: fecha a expansão, depois a seleção, depois o modal.

**Digitar volta o cursor ao topo.** Depois da tecla a pessoa está olhando uma
lista diferente, e manter posição nela seria manter posição em algo que ela não
leu.

**Filtro sem resultado escolhe nada**, e vazio precisa ser distinguível da
primeira linha — senão um filtro que não casou abriria a conversa mais recente.
E a lista **diz** que nada casou, em vez de ficar em branco: lista que se esvazia
se lê como lista que perdeu o conteúdo.

**Escolher a conversa já aberta não faz nada**, em vez de recarregá-la.

**Lista vazia não abre o modo**: modo que abre sobre lista vazia engole a próxima
tecla à toa.

## Um defeito de novo, e o guarda que enfim o pega

O cursor do filtro era `▌` cravado — **quinta** ocorrência de runa Unicode sem
alternativa ASCII neste pacote.

O guarda que o #243 criou não o pegou, porque enumerava as runas à mão. Agora ele
**deriva a lista dos próprios conjuntos de glifos** por reflexão: glifo novo
entra na proibição sozinho. Enumerar à mão obrigava a lembrar em dois lugares, e
o segundo é um teste que ninguém edita ao acrescentar uma marca.
