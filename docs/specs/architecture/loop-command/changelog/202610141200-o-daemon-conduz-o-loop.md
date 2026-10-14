# O daemon conduz o `/loop`

**Data:** 2026-10-14
**Specs afetadas:** `202608252000-loop-command` (`.p` §2 e §8). Passo 4 da §8 de
`desktop/docs/proposals/loop.md` (N4, pedido por D24), decisões Q2 A e Q7 A,
aprovadas pelo dono em 2026-10-13. As rotas e os tipos estão no changelog de
mesmo instante da `client-server-protocol`.

## O que mudou

A sequência do `/loop`, que morava no processo da TUI (`loopOne`,
`loopEverySpec`, `nextSpec`, `commitProposal`, `recordLoopResult`,
`loopVerdict` e o despacho no `eventMsg`), passa a existir também no daemon,
como `loopcommand.Runner`:

1. `/loop <pasta>` pergunta o que a pasta declara sem rodar nada; pasta sem
   critério, ou palavra solta que não nomeia pasta, abre a sessão
   **qualificadora**; o resto abre a sessão de trabalho. Survey que falha não
   impede: a sessão abre e responde.
2. A qualificadora termina → o daemon mede e escreve o `done.toml` → o loop
   fica em **`awaiting_review`**. Só o `continue` abre o trabalho, com o que a
   pessoa escreveu.
3. `/loop <objetivo>` mede todas as pastas, trabalha as pendentes uma sessão
   por vez, na ordem, e termina com o veredito da corrida inteira — critérios
   somados, nomes uma vez, pastas que ficaram por fazer nomeadas. Objetivo sem
   pasta é qualificado em `.dcode`.
4. Parar: `now` interrompe o turno e encerra; `after_spec` deixa a pasta
   corrente terminar e não abre a próxima.

A gramática (`ParseLoopArgs`, `LoopTask`, `GoalToQualify`,
`ArgumentToQualify`) ganhou casa em `loopcommand/args.go`. **A TUI não muda
neste PR**: ela continua com a cópia dela e conduzindo o próprio loop até o
passo 5 da proposta, que a faz chamar `POST /v1/loops` e apaga a cópia. Até lá
são duas cópias da mesma regra, de propósito e por um PR: mudar a TUI aqui
juntaria dois temas num PR só.

Dez invariantes novas na §8, cada uma com teste em `runner_test.go` ou no
teste de ponta a ponta do daemon (`TestTheDaemonRunsALoopOnAFolder`, sessão e
motor de verdade, modelo roteirizado).

## Por quê

Dois clientes veriam dois loops diferentes. A fila, a pasta corrente e o
veredito viviam na memória da TUI; um desktop que reimplementasse a sequência
seria uma segunda cópia da regra — o que o D24 recusou, e o que esta família
já viu derivar uma vez (o fim era silêncio, a proposta chegava cedo). Com o
daemon conduzindo, o loop sobrevive ao cliente que o pediu e morre com o
daemon, como toda sessão.

## Onde este PR diverge da proposta, e por quê

- **`continue` além do que a TUI faz.** Na TUI, depois da proposta escrita a
  pessoa digita `/loop` de novo. O `awaiting_review` com `continue` é o
  desenho aprovado (Q3 A); é a única transição que a TUI não tinha.
- **Num objetivo, pasta pendente sem critério é trabalhada, não
  qualificada** — é o que a TUI faz hoje (`specsFoundMsg` enfileira toda
  pendente e abre sessão de trabalho), apesar de o comentário do
  `proposalWrittenMsg` e a §3.3 da proposta dizerem "qualifica todas numa
  passada". A mudança foi fiel ao código; a divergência fica registrada para o
  dono decidir num PR próprio.
- **`StartLoopRequest` sem `mode`.** `CreateSessionRequest` não tem modo; o
  daemon não teria como passá-lo adiante sem um tema a mais.
- **`/loops/events` sem `removed`.** Um loop não sai da lista enquanto o
  daemon vive; um tipo de frame que nada emite seria promessa sem dono.
- **Código `loop_not_found`.** Continuar ou parar um id que o daemon não guarda
  precisava de um código que dissesse isso; nenhum dos existentes diz.
