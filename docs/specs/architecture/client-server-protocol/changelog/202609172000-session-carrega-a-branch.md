# `Session` carrega a branch

**Data:** 2026-09-17
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`, §5, tipo `Session`)
**Fonte:** o mesmo pedido que motivou
[202609172000 (client-tui)](../../client-tui/changelog/202609172000-a-branch-chega-na-barra.md)
— este arquivo documenta só a metade que vive no protocolo.

## O que mudou

`Session` ganha `Branch string` (`json:"branch,omitempty"`). Lido uma vez por
`app.New` a partir do `vcs.Read` que já monta o prompt
(`behavior-definition`, `202608170200-onde-o-agente-esta.md`), copiado pra
`session.Session` na construção (`daemon.build`) e incluído em `Describe()`
— então `session.created` passa a carregar o fato que antes só existia numa
variável local de `app.New`.

Não é restaurado num resume, ao contrário de `Family`/`Transport`/`BaseURL`:
é um fato do **workspace**, não do modelo, e uma sessão continuada lê o seu
próprio, na hora — não o da sessão que ela continua, que pode nem estar na
mesma branch mais.
