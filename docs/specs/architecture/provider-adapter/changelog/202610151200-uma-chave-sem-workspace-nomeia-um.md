# Uma chave sem workspace nomeia um

**Data:** 2026-10-15
**Specs afetadas:** `202608072334-provider-adapter` (`.config`, seção 2; `.p`, seções 2.2 e 7)
**Fonte:** uso real — a chave Anthropic do dono do projeto não tem escopo de
workspace, e a API recusa toda requisição com `invalid_request_error`
pedindo o cabeçalho `anthropic-workspace-id`.

## O que mudou

Nova chave, `model.anthropic_workspace` / `DCODE_ANTHROPIC_WORKSPACE`, string,
vazia por padrão. Quando presente, o transporte `anthropic` manda o valor no
cabeçalho `anthropic-workspace-id` em toda requisição. O transporte `openai`
nunca manda. `dcode config` mostra a chave com a origem, e um perfil em
`models.toml` pode trazer o seu, `anthropic_workspace`, aplicado só quando não
vazio.

E a recusa passa a dizer o que fazer. A API fala em cabeçalho, e não há
cabeçalho que se configure no dcode: quem lia a mensagem não tinha para onde
ir. Uma recusa `bad_request` do transporte `anthropic` cujo corpo cita
`anthropic-workspace-id` agora nomeia a chave e a variável, com as palavras da
API no fim. Com um workspace já configurado, diz que foi aquele o recusado.
Qualquer outra recusa segue como o provedor disse.

## Por que esse nome

`workspace` já quer dizer outra coisa no dcode: a pasta em que a sessão roda
(`--workspace`, `workspace.gates`). `model.workspace` seria lido como isso. O
nome carrega o provedor porque o conceito é do provedor, e fica em `model.`
porque viaja com o endpoint, igual a `model.base_url`.

## Por que no transporte e não na família

É um cabeçalho do dialeto, como `x-api-key` e `anthropic-version`, que já
moram em `HTTPTransport.Do` sob o mesmo `if` de transporte. A família não vê
cabeçalho nenhum (RN-1): ela serializa o corpo. Quem fala `anthropic` é quem
precisa mandar, seja qual for a família — o MiniMax M3 também fala esse
dialeto, e um gateway que exija o cabeçalho recebe o mesmo tratamento.

## Por que o valor é conferido na entrada

Ele vira cabeçalho HTTP. Um espaço ou uma quebra de linha seria recusado pelo
cliente HTTP no primeiro turno, como erro sobre um cabeçalho que ninguém
digitou. Só letras, dígitos, `_` e `-` passam, na inicialização e na leitura
do perfil.

## Por que não é tratado como credencial

O id de workspace não abre nada sem a chave e aparece no console da Anthropic
como identificador, não como segredo. Por isso pode morar no `config.toml` — o
nome da chave não casa com o filtro de credencial, e não devia casar.

## O que não mudou

Uma sessão retomada volta com o endpoint da sessão original
(`session.Origin`), mas o workspace vem da configuração atual, não do registro
da sessão. Para quem tem um workspace só, é o mesmo valor. Gravar o workspace
no registro da sessão fica para quando alguém tiver dois.
