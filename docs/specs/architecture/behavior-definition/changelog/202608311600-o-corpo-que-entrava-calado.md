# O corpo que entrava calado

**2026-08-31** — RN-7 ganha a regra do anúncio; `EventSkillLoaded` passa a
existir, e o fluxo nomeia a skill que carregou.

## O que estava errado

`internal/loop/turn.go` casava as skills e anexava o corpo ao histórico como
lembrete. Nenhum evento era emitido. `grep -i skill internal/tui/`, fora de
teste, devolvia **nada**.

Ou seja: um bloco de texto entrava no turno, gastava contexto e mudava o
comportamento do modelo, e a pessoa não tinha como saber que aconteceu, nem
qual skill foi.

## Por que isto é incoerência interna, e não pedido novo

Três lugares deste repositório já decidiram a questão:

1. `protocol.go`, no bloco dos próprios tipos de evento: *"this protocol's whole
   shape is that every observable fact travels the same way."*
2. `skills.go`, **trinta linhas acima** da injeção, o teto do índice anuncia o
   que deixou de fora em vez de truncar calado — *"a skill missing from the
   index is one the model never learns exists"*.
3. `dcode config <chave>` responde de onde um valor veio, e `--dump-prompt`
   imprime o índice.

A auditoria existia para tudo, menos para o que de fato entrou. O anúncio é a
regra já escrita, aplicada ao lugar onde ela tinha sobrado.

## O que o evento carrega, e o que não carrega

`Name` e `WhenToUse`. **Não** o caminho: o log de eventos é lido por outro
cliente, possivelmente em outra máquina, e caminho absoluto da máquina que
escreveu não é fato lá. De qual raiz a skill veio é pergunta que o
`--dump-prompt` e o sistema de arquivos respondem; **qual skill disparou** era a
pergunta que ninguém respondia.

A linha de quando-usar é a mesma que o modelo leu no índice, de propósito: a
pessoa e o modelo passam a olhar para a mesma frase.

## Turno que não carrega nada não diz nada

Um evento por turno, independente de ter havido carga, faria o fluxo carregar
uma linha cujo único conteúdo é que a funcionalidade existe. Anúncio sem nome
também não desenha — dizer que "uma skill foi carregada" sem dizer qual é linha
gasta em informar que o recurso existe.

## Invariantes

- `TestALoadedSkillIsAnnounced` — o evento sai, com nome e com a linha do índice.
- `TestATurnThatLoadsNoSkillAnnouncesNothing` — turno sem carga é turno mudo.
- `TestTheStreamNamesTheSkillThatLoaded` — o cliente nomeia, e ignora anúncio
  sem nome.
