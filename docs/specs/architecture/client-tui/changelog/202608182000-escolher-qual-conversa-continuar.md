# Escolher qual conversa continuar

**Data:** 2026-08-18

## O que mudou

`dcode -r` abre uma lista das conversas gravadas neste workspace, navegável, e
continua a que for escolhida. `dcode -c` continua a última sem perguntar.

Antes as duas grafias eram a mesma coisa: pegar a última.

## Por que separar

São perguntas diferentes. **Pegar a última** serve para a sessão que você acabou
de fechar. **Perguntar qual** serve para o workspace em que se trabalhou a
semana inteira — e aí adivinhar é errar quase sempre.

Colapsar as duas fez `-r` não ser nenhuma das duas coisas: quem tinha oito
conversas gravadas recebia uma, sem escolha e sem saber que havia outras. É a
divisão que outros harnesses já fazem, e a familiaridade aqui vale mais que a
economia de uma opção.

## O que a lista mostra

A pergunta que foi feita, quando, e quantos turnos custou. **Coluna de id é
coluna que ninguém escolhe** — é a mesma razão que faz `dcode sessions` titular
pela primeira pergunta.

Sessão em que nada foi perguntado não entra. É a maioria do que um diretório de
gravação guarda — uma é escrita toda vez que a interface abre — e oferecê-las
enterraria as quatro conversas reais debaixo de trinta vazias.

## Três decisões pequenas que o desenho carrega

**O cursor para nas duas pontas.** Dar a volta transforma "passei um" em "estou
em outro lugar", e aqui o outro lugar é a tarde errada de trabalho.

**A marca da seleção é um caractere, não uma cor.** Lista cuja seleção só existe
em cor é lista inutilizável com cor desligada — e este projeto renderiza tudo
sem cor por princípio.

**Sair sem escolher começa do zero.** Abrir em silêncio a primeira linha seria
escolher pela pessoa exatamente quando ela decidiu não escolher.

## Invariantes que entraram

- `-c` continua a última; `-r` **pergunta qual**.
- Cada linha da lista carrega a pergunta que foi feita.
- O cursor para nas duas pontas.
- A linha selecionada é marcada **sem cor**, e sair sem escolher começa do zero.
- A lista não oferece sessão em que nada foi perguntado.
