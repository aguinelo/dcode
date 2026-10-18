# A spec segue o código

**Data:** 2026-10-17
**Specs afetadas:** `202608180133-learned-memory` (`.p` §1 e §11, `.i` passo 0)

Correção de documentação. Nenhum comportamento muda.

## O que mudou

- **`.p` §1.** Dizia "Instável. Nada depende deste componente ainda." Dois
  erros: "instável" não é um nível da tabela do `SDD-HARNESS`, e algo depende —
  o `GET /v1/memory` do daemon lê o `.dcode/memory.md` por este componente, e o
  desktop lê a resposta. O nível passa a ser `experimental`, com a condição que
  a dependência impõe: a forma do arquivo muda livre desde que o leitor continue
  alimentando aquela resposta.
- **`.p` §11.** Os três contratos com limiar zero — `remembers-what-cost-time`,
  `does-not-remember-activity`, `uses-what-it-remembers` — nunca foram medidos:
  não há linha deles em `internal/evals/measured.go`, nem corrida registrada
  aqui. A spec agora diz isso, em vez de deixar o "zero sobe assim que houver
  uma corrida" ser lido como se a corrida tivesse acontecido. Nenhum eval foi
  rodado para esta correção.
- **`.i` passo 0.** Citava `## 11. Invariantes ainda sem cobrança`, seção que
  deixou de existir quando as invariantes da ferramenta voltaram à seção 10 nos
  commits que as implementaram; o §11 do `.p` hoje são os contratos. O passo
  passa a contar o que aconteceu.
- **Índices de changelog** do `.p` e do `.i`.

## Por quê

"Nada depende" convida a mudar o formato sem olhar para o endpoint e para o
desktop, que é exatamente o que quebraria.
