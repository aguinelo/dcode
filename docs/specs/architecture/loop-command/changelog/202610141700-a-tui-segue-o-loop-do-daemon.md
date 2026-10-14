# A TUI segue o loop do daemon

**Data:** 2026-10-14
**Specs afetadas:** `202608252000-loop-command` (`.p` §2, §8). Passo 5 da §8 de
`desktop/docs/proposals/loop.md`; as escolhas abaixo foram decididas pelo dono
antes do código.

## O que mudou

A TUI deixa de conduzir o `/loop`. Ela reconhece o comando (RN-3), confere a
gramática com o `loopcommand.ParseLoopArgs` só para dizer "uso" e "flag
desconhecida" no idioma da pessoa, e manda o argumento como foi digitado para
`POST /v1/loops`. Depois segue o fluxo `GET /v1/loops/events` e diz o que
mudou, com os textos de sempre: o plano de um objetivo, a sessão para onde o
loop foi (e passa a mostrá-la), cada proposta escrita com a contagem e o
resumo, e o fim — com o veredito só para um objetivo, como antes.

Saíram da TUI: `loopQueue`, `loopGoal`, `loopWorked`, `loopResults`,
`loopQualified`, `loopToQualify`, `loopOne`, `loopEverySpec`, `nextSpec`,
`commitProposal`, `recordLoopResult`, `loopVerdict`, o despacho no `eventMsg` e
a cópia da gramática. A gramática fica só em `loopcommand`, e os testes dela
vieram junto, com os mesmos nomes.

## As decisões do dono

- **Medir uma vez.** O plano usa o que a pergunta do loop achou de cada pasta
  (`criteria`, `unmet`, `unavailable`, `error` no `LoopSpecRun`); a TUI não mede
  de novo.
- **A proposta carrega o que a nota diz:** `proposal_criteria` e
  `proposal_summary` no `LoopSpecRun`.
- **Digitar `/loop` de novo segue o loop que espera revisão.** A nota da
  proposta sempre disse "revise, e então `/loop` de novo"; com o loop esperando
  em `awaiting_review`, o mesmo `/loop` (mesmas palavras, o espaçamento não
  conta) é o `continue` dele. Um `/loop` diferente é recusado, nomeando o que
  espera, em vez de seguir em silêncio uma corrida que a pessoa não pediu. Para
  comparar, o `LoopRun` carrega `args`.
- **O fim só de um objetivo é anunciado**, como antes: `/loop <pasta>` termina
  na sessão dela.
- **Os testes da TUI mudam de forma, não de expectativa.** Os que liam estado
  interno passam a dirigir a TUI por um daemon falso que roda o `Runner` de
  verdade, com os mesmos nomes e as mesmas asserções sobre o que a pessoa vê;
  os dois que guardavam "a proposta é escrita no fim do turno, não com a
  sessão ociosa" foram para `internal/app`, contra o daemon de verdade, porque
  a regra agora é dele.

## Uma sexta divergência, declarada

**Interromper o turno do próprio loop na TUI para o loop** (`stop now`), e a
nota diz isso. Antes a TUI seguia para a próxima pasta; com o daemon
conduzindo, um turno que termina `interrupted` sem parada pedida faria o loop
falhar. Parar é o que a pessoa quis dizer.

## O que ainda difere em tempo real

A TUI segue quadros de um fluxo: num loop muito rápido, a linha de uma sessão
intermediária pode não aparecer, porque a TUI vai direto para a sessão em que o
loop está. Num loop de verdade cada turno leva segundos, e o fim, o plano e as
propostas são ditos a partir do estado inteiro do loop, que nenhum quadro
perde.
