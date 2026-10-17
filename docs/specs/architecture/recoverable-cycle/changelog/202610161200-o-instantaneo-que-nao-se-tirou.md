# O instantâneo que não se tirou não vira arquivo criado

**Data:** 2026-10-16
**Specs afetadas:** `202608301200-recoverable-cycle` — a `.p §7` ganha duas
invariantes, e a RN-6 da `.r` passa a ser cumprida pelo código.

> **Estado.** `Snapshot` e `BeginCycle` distinguem "não existe" de "não li", e o
> `Undo` recusa o que não leu. Corrige um defeito que apagava arquivo.

## O defeito

A RN-6 dizia desde a primeira versão: *instantâneo que não pôde ser tirado é
dito, não presumido*. O código presumia. `Snapshot` e `BeginCycle` tratavam
**qualquer** erro de leitura como `existed: false` — permissão negada, erro de
E/S, o que fosse — e o `Undo` traduz `existed: false` como "o turno criou este
arquivo", e o remove.

Um arquivo que existia e não pôde ser lido virava, portanto, arquivo criado pelo
turno. O `/undo` da pessoa, ou o `UndoCycle` automático do laço numa regressão,
o **apagava**. É a pior falha possível de um seguro: o mecanismo que existe para
devolver trabalho destruindo o que nunca foi do turno.

## A correção

- **Só ausência é ausência.** `os.IsNotExist` é "não existia"; qualquer outro
  erro fica registrado como instantâneo **não lido**, que não é ponto de retorno
  nenhum.
- **O `Undo` recusa e nomeia.** Caminho sem instantâneo lido vai para a lista de
  recusados, a mesma que a pessoa e o modelo já leem. Nunca é removido, nunca é
  reescrito.
- **A falha é dita na hora.** `Snapshot` passa a devolver o erro, e `write` e
  `edit` o acrescentam ao resultado como aviso: a escrita segue — recusar o
  trabalho porque o seguro falhou seria o seguro segurando o trabalho como
  refém, que é a própria RN-6 — mas o modelo sabe que aquela escrita não se
  desfaz.
- **No ciclo, a falha é dita onde importa.** O instantâneo do `BeginCycle` só
  serve se o ciclo for desfeito; quando é, o `UndoCycle` recusa o caminho e o
  nomeia no lembrete. Um ciclo que nunca é desfeito nunca precisou dele.

## O que fica de fora, de propósito

A recusa não carrega o **motivo**. O lembrete do modelo e a nota do `/undo` dizem
"mudou no disco desde então", que é o motivo da outra recusa. Separar os dois é
mudança no protocolo (`UndoResult`) e no desktop, e não cabe numa correção cujo
ponto é parar de apagar arquivo. O caminho é nomeado, e a causa já foi dita
quando a escrita aconteceu.

## Testes

- `TestUndoNeverRemovesAFileItCouldNotRead` — reprova na `main`: o arquivo some.
- `TestUndoCycleNeverRemovesAFileItCouldNotRead` — idem, pela camada do ciclo.

Os dois pulam sob root, e dizem por quê: modo 000 não impede root de ler, e um
teste que passa sem reproduzir nada passaria pelo motivo errado.
