# Uma chamada de `edit`, muitas mudanças

**Data:** 2026-08-17

## O que mudou

`edit` ganha `edits`: uma lista de substituições aplicadas como um ato só.

```json
{"edits": [
  {"path": "a.go", "old_string": "Old", "new_string": "New", "replace_all": true},
  {"path": "b.go", "old_string": "Old", "new_string": "New"}
]}
```

**Todas são conferidas antes que qualquer uma seja escrita.** Uma edição
impossível deixa todos os arquivos como estavam, e a recusa diz qual delas
falhou.

A forma antiga segue idêntica. Ela não virou caminho separado — é o lote de um,
pela mesma função. Duas implementações de "substitua este texto" é como as duas
divergem, e a mais consequente diverge em silêncio.

## O problema

Renomear algo em doze arquivos eram doze chamadas. Não era só lento: cada uma
podia falhar sozinha, e **rename pela metade é pior que rename nenhum** — o
código deixa de compilar e o motivo fica espalhado por uma conversa.

Conferir tudo antes transforma isso em uma recusa, nomeando a edição que não
pôde ser feita, contra arquivos que ninguém tocou.

## Onde diverge do padrão de mercado, e por quê

O `MultiEdit` do Claude Code é **um arquivo só**; várias mudanças em um arquivo,
e vários arquivos são várias chamadas.

Aqui aceita vários arquivos, porque o caso que dói é justamente esse. A
propriedade que torna isso seguro não é atomicidade de sistema de arquivos — é
**validar tudo antes de escrever qualquer coisa**, que elimina a falha real
(trecho não encontrado, casamento ambíguo, arquivo não lido) sem prometer o que
não se pode cumprir.

O que continua possível: uma escrita falhar no meio, por disco cheio ou
permissão que mudou. O lote não previne isso, diz qual arquivo foi, e o `undo`
cobre o turno. Recusar-se a escrever porque a última escrita pode falhar seria
nunca escrever.

## O que o lote não afrouxa

Nenhuma invariante. Cada edição do lote passa pelas mesmas regras:

- **read-before-edit** vale para todo arquivo; o lote não é atalho para editar a
  partir de conteúdo presumido;
- **casamento ambíguo** recusa o lote inteiro — escolher a primeira ocorrência
  acerta quase sempre e, quando erra, edita o lugar errado em silêncio;
- **contenção** vale por caminho; um caminho fora do workspace para tudo;
- **toda rota é declarada** à política, e arquivo repetido é uma pergunta, não
  duas.

## Detalhes que a implementação decide

- Edições ao **mesmo arquivo** aplicam em ordem, cada uma sobre o que a anterior
  deixou. Sem isso um lote só faria mudanças independentes.
- O relatório presta contas **por arquivo**. Total único esconde o que não mudou.
- Num lote o diff sempre acompanha: é a única prestação de contas do que
  aconteceu entre arquivos. Numa edição só, a regra antiga de eco continua.
