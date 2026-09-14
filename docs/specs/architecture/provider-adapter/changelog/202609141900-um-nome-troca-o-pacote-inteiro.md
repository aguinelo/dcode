# Um nome troca o pacote inteiro

**Data:** 2026-09-14
**Specs afetadas:** `202608072334-provider-adapter` (`.p`, seção 2.2), `202608081203-configuration` (`.p`, seção 2)
**Fonte:** pedido do usuário — "consigo ter vários modelos já configurados e
trocar isso on the fly?", depois de configurar o Qwen local via
`--family generic` e perguntar como alternar de volta pra nuvem sem editar
arquivo toda vez.

## O que mudou

`/model <nome>` já existia, mas só trocava o texto do modelo — família,
transporte e endpoint continuavam os da sessão anterior, porque eram
decididos uma vez na inicialização do daemon. Trocar de/para um endpoint
customizado exigia reiniciar o processo com env vars diferentes.

Novo: `models.toml`, um arquivo separado (mesma razão de `requirements.toml`
ser separado — a forma não cabe no schema bijetivo de `config.toml`), guarda
perfis nomeados:

```toml
[profile.qwen-local]
model     = "qwen3.5-9b"
family    = "generic"
transport = "openai"
base_url  = "http://192.168.0.149:1234/v1"
window    = 32000
```

`/model qwen-local` agora resolve contra esses perfis **antes** de tratar o
nome como modelo cru, e aplica o pacote inteiro — não só o nome.

## Por que deu pra fazer sem mudar o protocolo

A descoberta que tornou isto pequeno: `buildProvider(opts)` já roda **por
sessão**, dentro de `app.New`, chamado por `Daemon.build` a cada
`CreateSessionRequest` — não uma vez por processo do daemon. O comentário do
próprio `build` já dizia isso: *"each session gets its own sandbox, resolver
and provider"*. `/model <nome>` já enviava uma nova sessão a cada troca; só
faltava o daemon saber olhar o nome num catálogo antes de tratá-lo como
modelo cru.

Isso significou: nenhuma mudança em `protocol.CreateSessionRequest`, nenhuma
mudança em `Family.Window` nem em nenhuma outra interface de `.p` §2. A
resolução inteira vive em `Daemon.build`, no mesmo lugar que já aplica
`req.Model` sobre `opts.Base` — um `if` a mais, não uma superfície nova.

## Por que `ParseSections` e não estender `config.toml`

A primeira tentativa de desenho — `[model.custom.<nome>]` dentro do próprio
`config.toml` — não funciona: `ParseTOML` é bijetivo por contrato
(`internal/config/toml.go`), e uma seção com nome escolhido pela pessoa não
tem como casar com `KnownKeys`, que é um schema fechado. A saída não foi
abrir uma exceção no schema fechado — foi notar que `ParseSections` já existe
exatamente pra isso, já usado por `grants.toml`: "a seção NÃO é checada
contra `KnownKeys` — o nome da seção É o dado". Um perfil é dado que a
pessoa escolheu, não uma chave que o produto declarou.

## O que ficou de fora

**Um daemon servindo vários workspaces não recarrega perfis por
workspace.** `opts.Profiles` é resolvido uma vez, na inicialização, contra o
workspace com que o daemon subiu — o mesmo comportamento que toda outra
configuração de `Options` já tinha (regras, instruções). Não é regressão:
é a primeira vez que perfis existem, herdando um limite que já valia pra
tudo o resto. Fica registrado, não resolvido.

**Nenhuma família nova ganhou limiar medido.** Um perfil que aponta
`family = "generic"` continua emitindo o aviso de que nenhum contrato
comportamental foi medido contra aquele endpoint — perfil é conveniência de
nomear, não uma nova alegação de qualidade.
