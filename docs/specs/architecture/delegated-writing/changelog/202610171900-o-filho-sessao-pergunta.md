# O filho-sessão pergunta; o filho delegado continua sem perguntar

**Data:** 2026-10-17
**Specs afetadas:** `202608191800-delegated-writing` (`.r` RN-3);
`202610132000-orchestration` (`.r` RN-7, `.p` §5.2 e §10).

## O que mudou

Nada no código desta família. Fica dito onde a RN-3 deixa de valer: ela governa o filho que
`delegate` cria **dentro de um turno** — o `childRegistry` de `internal/loop/delegate.go`, sem
sessão, sem log próprio, sem ninguém do outro lado. Esse continua sem perguntar: escrita que
exigiria consentimento volta ao pai.

O filho da orquestradora é outra coisa: uma **sessão**, com `parent_id`, `role: child`, `owns`,
worktree e log próprios (`client-server-protocol`, changelog `202610171900`). Ele pede aprovação
como qualquer sessão, o pedido não expira, e quem responde é a pessoa, ligada ao filho — nunca uma
ferramenta da orquestradora.

## Por quê

A RN-3 existe porque o filho delegado não tem com quem falar: o turno do pai está em curso, e uma
pergunta no meio dele não chega a ninguém. O filho-sessão tem — é uma conversa que a pessoa abre
na lista e responde ali. Negar a ele o pedido de aprovação o obrigaria a escrever só onde o
envelope já permite sem perguntar, e o trabalho que precisa de consentimento voltaria à
orquestradora, que roda em modo `plan` e não escreve nada (decisão 3 do dono, 2026-10-17).

Registrado aqui, na família cuja regra se estreita, e não só na orquestração: quem ler a RN-3
precisa saber que ela tem fronteira.
