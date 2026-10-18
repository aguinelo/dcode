# A spec segue o código

**Data:** 2026-10-17
**Specs afetadas:** `202608261730-done-qualifier` (`.r` RN-6; `.p` §5, §6, §8, §9, §10, §11, §12; `.config` §1 e §4)

Correção de documentação. Nenhum comportamento muda. O histórico de propostas
foi reescrito no `202610161200` e não é refeito aqui.

## O que mudou

- **Os números medidos.** O `.config` §4 dizia que os três contratos foram
  medidos em 2026-08-27 e que "um deles fechou; dois não". O `.p` §8 e o
  `internal/evals/measured.go` dizem 2026-08-28 e **dois de três**: aquela
  primeira medição foi substituída no `202608280200`, que nunca chegou ao
  `.config`. Agora os dois dizem a mesma coisa, com a primeira registrada como
  substituída.
- **`Measured.Truncated`.** O `.p` §5 mostrava a struct sem o campo, que o
  código tem (`internal/loop/qualifier/measure.go`), e chamava o teto de
  `qualifier.output_limit` onde o código e a invariante dizem `MaxOutput`.
- **O portão da assinatura.** O `.p` §6 dizia que "o laço para ali" sem dizer
  onde nem como sai. Agora diz: `awaiting_review`, sem prazo, e só o
  `POST /v1/loops/{id}/continue` leva ao trabalho — o primeiro vence, o segundo
  recebe `loop_not_awaiting_review` —; `stop` encerra sem trabalho. A RN-6 da
  `.r` ganhou a frase correspondente: assinar é continuar.
- **O turno qualificador que termina sem propor.** Não estava escrito. O commit
  recusa com *"nothing was proposed for <sessão>"*, nada é escrito nem
  anunciado, e o laço termina com o erro.
- **Invariante promovida.** *"`done_propose` não existe no registro de um turno
  que não é de qualificação"* estava na §10, prevista, e o teste que a assere
  já existe: `TestTheToolRefusesOutsideAQualifyingTurn`. O teste mostra o que o
  código faz — fora do turno qualificador a ferramenta **recusa** e diz por
  quê —, e a linha promovida à §9 diz isso. A reivindicação entrou em
  `internal/loop/invariants_test.go`.
- **As outras previstas** falavam da `DoneSet` "que voltou na assinatura", o
  ida-e-volta que saiu no `202610140900`. Continuam previstas, agora sobre o
  `done.toml` deixado ao continuar.
- **Referências de seção.** A §9 apontava a etapa 1 "da §11", e a §11 a etapa 2
  "da §11" — a ordem de entrega é a §12; a §12 apontava o número "da §10", que
  é a §11; a §6 apontava `render.go` na §5, que está na §2; o `.config` chamava
  as previstas de §9 (são a §10), a pergunta de §10 (é a §11) e o
  `MaxProposals` de §5 (é a §7).
- **Encurtado.** As subseções históricas da §8 sobre a primeira leitura e o
  contrato retirado foram condensadas, com ponteiro para os changelogs que as
  contam inteiras. O `.p` já passava de 500 linhas e continua passando (530);
  esta correção não o aumentou.
- **Índices de changelog** da `.r`, do `.p` e do `.config`.
