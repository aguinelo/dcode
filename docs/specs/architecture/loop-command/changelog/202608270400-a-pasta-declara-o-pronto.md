# A pasta da spec declara o próprio pronto

**Data:** 2026-08-27
**Specs afetadas:** `202608252000-loop-command` — três invariantes novas na §8.

## Por que

`/loop` ficou digitável e, medido contra as 17 specs reais do Code Plain,
devolveu **zero critérios em todas**. Nenhum `tasks.md` de lá tem marcador
`verify:`.

Os critérios existem — 27 seções "Critérios de aceitação" — e são **prosa**:

> 1. Home carrega em < 1s em 4G.
> 2. Lighthouse ≥ 95 Performance.
> 3. Bundle JS ≤ 100KB gzipped.

É o que uma pessoa escreve, e é exatamente o que nenhum parser pode transformar
em comando sem inventar um. Ensinar o parser a "derivar" daqui produziria o
critério fraco que a `done-qualifier` foi escrita para recusar — `test -f
arquivo.ts` vermelho hoje, verde no instante em que o arquivo existir, medindo
nada.

## O que passou a existir

Um `done.toml` **dentro da pasta da spec** é a definição de pronto dela.

```
specs/2026-08-25-home-page/
  spec.md      ← prosa, para o humano
  tasks.md     ← as tarefas
  done.toml    ← como se sabe que ficou pronto
```

Mesmo nome e mesmo formato do arquivo do workspace, porque é a mesma coisa:
dois nomes para um conceito é como alguém aprende um e não encontra o outro. Um
parser só, compartilhado.

Quem escreve o arquivo é indiferente — a mão, um turno comum do agente, ou um
dia o qualificador. Ele é diffável, revisável, e sobrevive à sessão que o
produziu, que é mais do que qualquer aprovação em tela consegue.

## As três decisões

**Vence o `tasks.md`, que não é consultado.** Duas fontes para uma pasta são
duas respostas a "contra o que isto está sendo medido", e a que ninguém lê é a
que deriva.

**`done.toml` presente e sem critério é erro.** Definição de pronto sem nada
dentro relata pronto. Cair no `tasks.md` mediria o turno contra outra coisa que
o arquivo da própria pasta.

**Ausente é o caso comum e não é erro.** A pasta declara em `tasks.md`, ou não
declara.

## O que isto não é

Não é o qualificador. Ninguém deriva critério aqui, ninguém mede antes, ninguém
assina. É só um lugar para o operador dizer, em comandos, o que a spec dele diz
em frases — e é o que faz `/loop` sair de "não tem definição de pronto" para
rodando, hoje, sem que nada seja inventado.

O Passo 5 da `.i` — congelar o formato do `tasks.md` com o Code Plain — continua
aberto, e agora tem uma alternativa que não depende dele.
