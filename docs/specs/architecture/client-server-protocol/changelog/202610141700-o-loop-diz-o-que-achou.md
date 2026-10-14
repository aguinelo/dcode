# O loop diz o que a pergunta e a proposta acharam

**Data:** 2026-10-14
**Specs afetadas:** `202608072240-client-server-protocol` (`.p` §5.0). A
decisão está no changelog de mesmo instante da `loop-command`.

## O que mudou

Campos novos, aditivos:

| Tipo | Campo | O quê |
|---|---|---|
| `LoopRun` | `args` | o que foi digitado depois de `/loop` |
| `LoopSpecRun` | `criteria`, `unmet`, `unavailable`, `error` | o que a pergunta do loop achou da pasta antes de qualquer sessão abrir |
| `LoopSpecRun` | `proposal_criteria`, `proposal_summary` | quantos critérios a proposta escreveu, e o resumo que uma pessoa lê |

As pastas de um objetivo entram no loop todas de uma vez, num quadro só, para
nenhum cliente ver meio plano.

## Por quê

Um cliente que desenha o plano ou a proposta precisaria medir as pastas de
novo — rodando os critérios outra vez — ou ler o disco, que o desktop não lê.
O loop já tinha medido; agora diz. `args` é como um cliente distingue "o loop
que eu pedi" de outro esperando no mesmo workspace.

Todos são `omitempty`, e `criteria` ausente é zero: um cliente que já lia o
`LoopRun` antes deles continua lendo, e o desktop os decodifica como
opcionais.

`experimental`, abaixo de 1.0: campos novos são **MINOR** como feature.
