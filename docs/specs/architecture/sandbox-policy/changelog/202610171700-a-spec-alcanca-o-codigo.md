# A spec alcança o código

**Data:** 2026-10-17
**Specs afetadas:** `202608072336-sandbox-policy` (`.p`, §3, §3.1 e §5)
**Fonte:** revisão das specs contra o código. Só documentação: nenhum
comportamento muda.

## O que estava desatualizado

- **`Evaluate` sem o parâmetro da rede.** A assinatura no `.p` §3 terminava
  em `rules Rules, inWorkspace func(Access) bool`; no código há
  `net NetworkGrant` entre os dois, e é ele que faz a rede concedida deixar de
  ser pergunta no `workspace-write`. `NetworkGrant` e suas três formas
  (`GrantedNetwork`, `WithheldNetwork`, `NetworkGrantFunc`) entraram na spec, e
  a célula de rede da tabela 3.1 diz o que a concessão muda.
- **`Access` sem `Rel`.** O caminho relativo ao workspace, preenchido por
  `Resolve`, é contra o que as regras casam.
- **A interface `Sandbox` era outra.** O `.p` §5 mostrava
  `Wrap(ctx, cmd Command, mode, workspace string)` e não tinha `Name()`. No
  código não existe tipo `Command`: é
  `Wrap(ctx, workdir, command string, mode policy.SandboxMode)`, e `Name()`
  existe. `New` também entrou.

Nenhum changelog registrou essas mudanças de assinatura quando aconteceram. É a
segunda vez que o §3 recebe uma "correção de assinatura" — a primeira, de
`202608091700`, também chegou depois do código.
