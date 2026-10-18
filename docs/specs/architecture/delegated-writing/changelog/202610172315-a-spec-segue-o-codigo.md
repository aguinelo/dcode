# A spec segue o código

**Data:** 2026-10-17
**Specs afetadas:** `202608191800-delegated-writing` (`.r` §6), `202608191900-delegated-writing` (`.p` §1, §6, §11)

Correção de documentação. Nenhum comportamento muda.

## O que mudou

- **`.p` §1.** Dizia "desenho aprovado, não implementado… nenhuma linha dele
  existe em código ainda", enquanto a própria seção registrava que os três PRs
  da §9 tinham entregado tudo. Agora declara o nível pela tabela do
  `SDD-HARNESS` — `experimental`, porque o `owns` é lido pelo modelo e não por
  um cliente — e diz onde o código vive.
- **`.p` §6.** Dizia que o harness de eval recusa a chamada delegada e que
  "70% é piso sobre a taxa do produto". O limiar é 25% (`contracts.go`), e o
  harness roda o turno filho desde o `202608200100`, cuja correção já mostrava
  que consertá-lo moveu o número dois pontos, dentro do ruído. O parágrafo
  passa a dizer isso, e o piso é 25%.
- **`.r` §6.** A pergunta "um filho que escreve deve poder rodar comando?" já
  estava respondida pela invariante do `.p` §8 — *filho que escreve não carrega
  ferramenta opaca nem a de delegar* —, asserida por
  `TestAWritingChildCarriesWritingToolsAndNothingOpaque`. Fica fechada, com o
  motivo.
- **Índices de changelog.** O `.p` dizia "sem alterações desde a criação" com
  oito entradas na pasta.

## Por quê

Uma spec que diz "nada disto existe" sobre código que existe faz quem a lê
procurar o que fazer num trabalho que já foi feito — e um número de limiar que
não é o do código é o número que a próxima revisão discute.
