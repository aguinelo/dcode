# Decisões do desktop — D62 em diante

A continuação de [`DECISIONS-4.md`](DECISIONS-4.md), que vai de D49 a D61, e
dos anteriores, a partir de [`DECISIONS.md`](DECISIONS.md), que tem a
introdução, D1 a D20, as decisões em aberto, as lacunas e os pedidos ao núcleo.
A numeração é uma só; este arquivo começa porque o DECISIONS-4 passaria de 500
linhas com D62.

**D62. A revisão mostra as propostas anteriores do turno, recolhidas embaixo da
escrita.** Decisão do dono, de 2026-10-16: construir o histórico que a
`done-qualifier` promete (§4) como defesa contra uma reproposta que troca um
critério por um trivial. O daemon guarda toda proposta do turno qualificador,
mede e escreve a última, e o `done.proposed` passa a levar `sequence`,
`proposed_at` e `earlier` (registro `202610161200`).

- **O card continua sendo o da proposta escrita** (D56). Quando o turno
  repropôs, o título ganha "· 3ª proposta do turno", e embaixo, antes dos
  caminhos protegidos da escrita, um `<details>` fechado: "2 propostas
  anteriores · o turno repropôs; só a última foi medida e escrita".
- **Aberto, cada anterior em ordem, da mais antiga:** "#1 · 3 critérios · 4m00s
  antes da última", e cada critério com o nome, o comando, o que o proponente
  disse dele ("disse que falharia" / "disse que passaria") e o porquê; os
  protegidos dela, se havia. Sem classe, sem exit, sem ✓/✗: as anteriores
  nunca rodaram, e desenhar uma medição que não aconteceu seria inventá-la.
- **O tempo é contado da última, não do relógio.** "4m00s antes da última" não
  depende do fuso da máquina e diz o que importa — quanto o turno pensou entre
  uma e outra.
- **Uma proposta só não muda nada:** sem sufixo no título e sem o `<details>`.
  O caso comum fica com a cara de antes.

As réguas: o card só aparece com um loop esperando revisão, que as telas 02 e 03
não têm; as duas ficam como estavam.

Descartados: as anteriores lado a lado com a escrita, aberto por padrão (o card
já é o mais alto da régua, e o olho deve cair primeiro no que vai ser usado);
um diff entre propostas (custa desenho próprio, e a lista em ordem já mostra a
troca); medir as anteriores para dar classe a elas (rodar critérios que ninguém
vai usar, só para enfeitar a lista).
