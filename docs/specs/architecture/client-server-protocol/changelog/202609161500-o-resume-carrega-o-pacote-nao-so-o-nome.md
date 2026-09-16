# O resume carrega o pacote, não só o nome

**Data:** 2026-09-16
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`, §5 e a lista de
invariantes sobre continuar sessões)
**Fonte:** relato do usuário — trocou de modelo com `/model qwen-local`,
fechou, e ao continuar com `dcode -c` a sessão voltou pro MiniMax-M3 em vez
de ficar no Qwen local. "Precisamos garantir essa persistência na sessão."

## O que quebrava

`session.created` só gravava `Model` — o nome resolvido do modelo (ex.:
`"qwen3.5-9b"`), nunca `Family`, `Transport` ou `BaseURL`. Um `/model
qwen-local` abre sessão nova (`internal/tui/program.go`, `newSession`) — por
design, o prefixo não pode ser reescrito — e essa sessão nova grava seu
próprio `session.created` com o modelo resolvido do perfil. Até aqui, certo.

O problema aparecia depois. `dcode -c`/`dcode -r` (`cmd/dcode/tui.go`) sempre
mandavam `Model: opts.Model` no `CreateSessionRequest` — o que `model.name`
resolve *nesta* execução, a partir de `config.toml`/env, com default
`MiniMax-M3`. Mesmo se o daemon soubesse restaurar um modelo a partir do
registro, faltava família/transporte/endpoint pra reconectar no Qwen local:
`Model: "qwen3.5-9b"` sozinho, contra a família e o transporte default,
tentaria falar com o endpoint errado usando o nome de modelo errado.

## O que mudou

`protocol.Session` ganhou `Family`, `Transport` e `BaseURL` — o resto do
pacote que um nome de perfil resolve (`202608072334-provider-adapter.p.spec.md`
§2.2), com a mesma leitura de vazio que `model.family`/`model.transport`/
`model.base_url` já davam pra um override ausente. `session.Session` carrega
os mesmos três campos, atribuídos ao lado de `ContextWindow` logo depois de
`session.New` (`internal/app/daemon.go`), e `Describe()` os inclui — então
`session.created` agora grava o pacote inteiro, não só o nome.

`Daemon.build`, ao continuar (`req.Resume != ""`) sem um modelo explícito no
pedido (`req.Model == ""`), lê o `session.created` do registro sendo
retomado (`session.Origin`, novo, lê só a primeira linha — o mesmo evento que
`Browse` já lia pro nome, devolvido como o tipo de fio em vez de um segundo
recorte dos mesmos campos) e aplica `Model`/`Family`/`Transport`/`BaseURL`/
`ContextWindow` diretamente aos `Options` — sem passar por `applyModelRequest`,
porque o nome gravado é o modelo resolvido (`"qwen3.5-9b"`), não
necessariamente o nome de um perfil ainda configurado. Um `req.Model`
explícito sempre vence, retomando ou não — a checagem roda primeiro.

O cliente (`cmd/dcode/tui.go`) parou de mandar `Model: opts.Model` sem
condição. `modelOverride` olha a proveniência de `model.name`
(`config.Resolved.Get`, já usado pros avisos de override): se a única fonte é
o default embutido (`config.SourceDefault`) E a execução está continuando
(`carry != ""`), o pedido vai com `Model` vazio, deixando o daemon restaurar
o pacote do registro. Qualquer fonte mais forte — flag, env, `config.toml` de
projeto ou de usuário — significa que alguém disse algo sobre o modelo nessa
execução, e isso continua valendo, retomando ou não.

## Por que não reaproveitar `applyModelRequest`

Essa função resolve um **nome** contra `opts.Profiles` — é o caminho de
`/model <nome>`, onde alguém digitou algo que pode ser um perfil ou um nome
de modelo cru. O que `session.Origin` devolve já é o pacote **resolvido**: se
`opts.Profiles["qwen-local"]` mudar ou for removido entre a sessão original e
o resume, a reconexão ainda usa o endpoint exato que a sessão gravou, não uma
segunda resolução que pode ter mudado de sentido. Atribuição direta é o
contrato certo aqui — não é o mesmo problema que `applyModelRequest` resolve.
