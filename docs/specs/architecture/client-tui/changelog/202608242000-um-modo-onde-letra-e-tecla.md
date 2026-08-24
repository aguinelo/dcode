# Um modo onde letra é tecla

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seções 7 e 10)
**Fonte:** design `Coding Agent TUI v2` — o badge NAV do rodapé

## O que mudou

O fluxo ganhou um modo que **toma o teclado**, e com ele os quatro temas do
design, percorridos por `t`.

## A tecla que faltava era o modo

O rodapé do design oferece `j/k mover`, `↵ abrir`, `t tema`. São letras. Letra
numa linha em que se digita é o defeito que este produto corrigiu **duas vezes
hoje**, nas duas por relato de quem usa.

Não implementá-las teria sido descartar metade do desenho; implementá-las soltas
teria sido reintroduzir o defeito. A resposta estava no próprio desenho: ele põe
um **badge NAV** ali. Um badge é o nome de um modo.

Dentro de um modo que engole toda tecla que não nomeia, letra é segura — é
exatamente como o modal de aprovação e a lista de conversas já funcionam.

## Entrar é `esc`

É a única tecla com convenção real para "parar de digitar, começar a navegar", e
não é letra. E lê como o que este produto já quer dizer com ela: **sair do que
você está dentro**. Sair de uma linha vazia é entrar no que está acima dela.

Com algo escrito ela não faz nada, como antes: abandonar uma linha que alguém
está no meio de escrever não é para isso que serve o escape aqui.

## A seta na borda passou a rolar

`↑` numa sessão sem histórico **caminhava para dentro do fluxo**. Foi por esse
caminho que o bug do `v` voltou: o cursor ia para o fluxo em uma tecla, sem nada
na tela dizendo isso, e o `v` seguinte virava atalho.

A correção que saiu hoje devolveu a letra, o que removeu o sintoma. Remover o
**caminho** remove o estado que o produzia. O teste que eu escrevi de manhã para
reproduzir aquele relato foi reescrito para afirmar que o caminho não existe.

## O badge diz de quem é o teclado

Aceso no modo, apagado fora. É a única coisa na tela que diz qual dos dois
teclados uma tecla vai alcançar — e esse estado era invisível, que é metade da
razão de uma letra ligada a um modo poder comer uma tecla sem causa aparente.

O rodapé troca de teclas junto: fora do modo anuncia `esc navegar`, dentro
anuncia o que o desenho pede.

## Quatro temas, uma tabela de papéis

`neon`, `ashes`, `ember`, `mono` — os quatro do design, com os valores dele.

O **mapeamento de papéis é escrito uma vez** e compartilhado pelos quatro. É o
que faz um tema ser um tema em vez de quatro telas: mude de que cor é um título,
e os quatro mudam junto. Um mapeamento por tema divergiria na primeira vez que
alguém acrescentasse um papel.

O teste de contraste roda nos **quatro**. A margem mais apertada de cada um é o
cromo, entre 1.98:1 e 2.24:1 — que é o piso que eu mesmo escolhi por medida
quando desviei do `--bd` do design.

## O que não entrou

O tema escolhido **não persiste**: sair e voltar traz o neon. Guardá-lo é uma
preferência no disco, e `internal/tui` não lê disco por desenho — a borda
injeta. Mesma pergunta que a coluna já tem em `docs/ROADMAP.md` §10, e a mesma
resposta: o lugar é a borda, e a decisão é se "eu troquei de tema" é preferência
ou é gesto.
