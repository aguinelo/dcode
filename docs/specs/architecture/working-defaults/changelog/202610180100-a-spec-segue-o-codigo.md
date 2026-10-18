# A spec segue o código

**Data:** 2026-10-18
**Specs afetadas:** `202608262200-working-defaults` (`.p`, `.config`)

> **Regra:** correção de documentação. Nenhum comportamento muda; a spec passa
> a dizer o que `internal/behavior`, `internal/workspace` e `internal/app` já
> fazem.

## O que estava errado

A família ainda se descrevia como desenho aprovado. O `.p` dizia que a seção de
práticas, a origem e o inventário de portões eram "desenho aprovado, não
implementado"; o `.config` dizia "nenhuma chave declarada ainda". As quatro
etapas da §8 estão em código e os contratos foram medidos:

- `Doctrine.Practices`, `DoctrineOverlay.Practices`, `practices.md` e
  `SectionOrigins.Practices` existem, e `DoctrineAudit` imprime a linha;
- `internal/workspace/probe.go` sonda `package.json` e `Makefile`, e desde
  `202610162300` devolve também os manifestos ilegíveis;
- `DCODE_WORKSPACE_GATES` é lida, e está declarada em
  `behavior-definition.config` §3, por onde o código entrou.

O que mudou na spec:

- **§1** diz que a família foi entregue, e por que continua sem `.i`.
- **§2** lista os arquivos que existem — o tipo `Workspace` mora em
  `internal/behavior/repo.go`, não num `workspace.go` que nunca foi criado — e
  diz que `Probe` é `workspace.Probe`.
- **§5** traz a assinatura real, `Probe(ctx, dir) (gates []Gate, unreadable
  []Unreadable)`, o tipo `Unreadable`, a regra de alvo de `Makefile` como o
  código a aplica (a spec dizia "sem receita vazia", e o código não olha a
  receita), o teto `MaxGates` e o aviso `manifest_unreadable`.
- **`.config` §1** troca "chave prevista" pela chave que existe, apontando a
  linha de tabela na `behavior-definition` em vez de repeti-la.
- **`.config` §4** deixa de dizer "cinco contratos, três abertos, um a 5%" —
  números de agosto que a `.p §6` já tinha substituído — e aponta a `.p §6`.
- **Os índices de changelog** passam a listar `202609012200` e, no `.config`,
  as entradas de `202608280200` em diante.

## Por que as invariantes não foram promovidas

A §7 chamava-se "Invariantes previstas". Promovê-las a "verificáveis" exigiria
reivindicá-las com `specguard.Check` nesta família. Mas os testes que as
afirmam — `internal/behavior/practices_test.go` e `repo_test.go` — já são
reivindicados pela `behavior-definition §8`, onde o código foi entregue, e duas
famílias reivindicando o mesmo teste é a duplicata que a guarda não pega.

A §7 passa a se chamar "Invariantes — reivindicadas na `behavior-definition`" e
cada linha diz onde é reivindicada. Três não são em lugar nenhum, e ficam
nomeadas como lacuna, com os testes que já existem para fechá-las:
`DoctrineAudit` imprimindo `Practices`, `Probe` sem manifesto devolvendo lista
vazia, e o manifesto ilegível dito ao abrir. Fecham na `behavior-definition §8`.
