# `dcode -c` não pisa no modelo da sessão

**Data:** 2026-09-16
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção sobre `-c`/`-r`)
**Fonte:** o mesmo relato que motivou
[202609161500 (client-server-protocol)](../../client-server-protocol/changelog/202609161500-o-resume-carrega-o-pacote-nao-so-o-nome.md)
— este arquivo documenta só a metade que vive aqui, no cliente.

## O que mudou, do lado do cliente

`cmd/dcode/tui.go` mandava `Model: opts.Model` em todo `CreateSessionRequest`,
sem olhar se a execução estava retomando (`carry != ""`) nem de onde
`model.name` veio. `modelOverride` agora só deixa isso passar sem condição
quando **não** está retomando; retomando, checa a proveniência
(`config.Resolved.Get("model.name")`) e só envia o modelo se algo além do
default embutido o definiu — flag, env, `config.toml` de projeto ou de
usuário. Só o default e retomando: o pedido vai com `Model` vazio, e o
daemon (`202609161500`) reconecta ao pacote que a sessão retomada de fato
usava.

## Por que isso e não travar o default em algum outro lugar

A alternativa óbvia — nunca resolver `model.name` pro default quando o
comando é `-c`/`-r` — moveria a decisão pra dentro de `Resolve`, que não sabe
(e não deveria saber) se está sendo chamado por um resume. `modelOverride` é
puro sobre o que já existe: o modelo efetivo, a proveniência dele, e se há
algo pra continuar. Três testes cobrem as três leituras — nada pra continuar
sempre aplica o efetivo; continuando com o default, some; continuando com
algo explícito, o explícito vence.
