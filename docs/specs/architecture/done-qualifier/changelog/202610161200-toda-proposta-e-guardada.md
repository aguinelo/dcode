# Toda proposta do turno é guardada e chega ao operador

**Data:** 2026-10-16
**Specs afetadas:** `202608261730-done-qualifier` (`.p` §2, §4, §6.1, §7, §9,
§10; `.config` §1.2, §5). Decisão do dono, de 2026-10-16: construir o histórico
que a §4 prometia.

## O que mudou

A §4 dizia que a defesa contra uma reproposta manipulada é o operador receber
**todas** as propostas, em ordem; a `.config` §5 dizia que toda proposta fica no
`Record`; a §7 descrevia `Qualify`, `Record` e `SourceQualified`. Nada disso
existia. O código guardava **uma** proposta, trocada a cada `done_propose`, e a
§9 reivindicava exatamente isso: "uma segunda proposta substitui a primeira".

Agora:

- `Proposals` (`internal/app/proposals.go`) guarda cada proposta do turno
  qualificador, na ordem em que chegou, com o número (`Sequence`, de 1) e a hora
  (`At`). `Take` devolve o histórico inteiro e o apaga.
- O commit mede e escreve a **última**, e anuncia `done.proposed` com `sequence`,
  `proposed_at` e `earlier` — as anteriores como o modelo as mandou, sem medição.
  `CommitDoneResponse` ganha `proposals`, e o `Summary` lista as anteriores.
- `MaxProposals` é uma constante, 8. A proposta que passaria dele é recusada ao
  modelo com o motivo, e o commit seguinte não grava nada, não anuncia nada e
  diz por quê, com o código `invalid_input`.
- A §7 descreve o que existe: `Proposals`, `Take`, `MaxProposals`. `Qualify`,
  `Record` e `SourceQualified` saem do texto, e a linha de `SourceQualified` sai
  da §10. A régua qualificada chega pela pasta que o `/loop <pasta>` nomeou.
- A §9 troca "uma segunda proposta substitui a primeira" por três linhas: o
  histórico guardado em ordem, o teto que reprova, e o aviso que leva as
  anteriores. A linha prevista do `Record` sai da §10, porque agora é verificável.
- A chave prevista `DCODE_QUALIFIER_MAX_PROPOSALS` sai da `.config` §1.2 antes de
  existir: o teto é constante.

## Por quê

A troca de uma proposta por outra é o caminho de manipulação que a §4 nomeia:
um critério que mede de verdade substituído por um trivial. Com um slot só, o
operador via a última e nunca a primeira, e a manipulação ficava invisível
exatamente onde a spec dizia que ela seria vista.

**Por que o teto reprova em vez de descartar.** Com toda proposta guardada e
mostrada, o histórico precisa de fim. Descartar a mais antiga para abrir espaço
apagaria justamente a sequência que o histórico existe para mostrar; mandar a
última com a contagem à vista, como a chave prevista dizia, trataria o teto como
fim normal. Um turno que repropôs oito vezes é um turno que a pessoa precisa ver
falhar.

**Por que constante e não chave.** Oito é muito acima do que corrigir um comando
ou apertar um critério pede, e ninguém pediu para ajustar. Uma chave declarada
sem quem precise dela é superfície que promete controle.

**Por que as anteriores não têm medição.** Só a última é medida, fora do turno e
sob a fronteira do trabalho (§7). Dar classe a uma anterior exigiria rodá-la, e
rodar critérios que ninguém vai usar é custo sem leitor; dizer uma classe sem ter
rodado seria inventá-la.

## O que isto não muda

- A última proposta é a medida, a escrita e a que o trabalho usa depois de
  `POST /v1/loops/{id}/continue`.
- O protocolo é `experimental`: campos novos e um teto novo são **MINOR**.
