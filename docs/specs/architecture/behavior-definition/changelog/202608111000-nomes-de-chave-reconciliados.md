# Os nomes declarados passam a ser os que o código lê

**Data:** 2026-08-11

## O que mudou

Nove chaves que o produto **honra** e nenhuma spec declarava passam a estar
declaradas, com o nome que o código de fato lê:

| Spec | Chaves |
|---|---|
| `behavior-definition` | `DCODE_BEHAVIOR_INSTRUCTIONS_ENABLED`, `DCODE_BEHAVIOR_SKILLS_ENABLED`, `DCODE_BEHAVIOR_REMINDERS_ENABLED`, `DCODE_SHOW_REASONING` |
| `sandbox-policy` | `DCODE_CONFIRM_WRITE`, `DCODE_CONFIRM_READ`, `DCODE_CONFIRM_COMMAND` |
| `configuration` | `DCODE_CREDENTIAL_BACKEND` |
| `distribution` | `DCODE_UPDATE_CHANNEL` |

## Como isto apareceu

Pelo avesso. A spec declarava `DCODE_INSTRUCTIONS_ENABLED`, `DCODE_SKILLS_ENABLED`
e `DCODE_REMINDERS_ENABLED`; o código sempre implementou `DCODE_BEHAVIOR_*`. A
varredura de `202608110900` viu os nomes declarados, não os achou em lugar
nenhum, e removeu as três — corretamente, porque aqueles nomes de fato não
existiam.

O que sobrou foi o defeito espelho, e o mais silencioso dos dois: **configuração
real, funcionando, que ninguém consegue descobrir.** A primeira falha frustra
quem tentou; esta nunca chega a ser tentada.

## Por que o nome do código venceu

Duas saídas cabiam — renomear no código para o nome da spec, ou corrigir a spec
para o nome do código. Venceu o segundo, por três motivos:

1. O prefixo `BEHAVIOR_` casa com a seção `behavior.*` do `config.toml`, e a
   bijeção entre chave e variável é asserção de teste.
2. Renomear no código quebraria a configuração de quem já a usa, para ganhar
   quatro caracteres.
3. É o precedente de `sandbox.policy` → `sandbox.approval_policy`, registrado no
   `DECISIONS.md`: o nome interno passou a ser o que o usuário escreve, não o
   contrário.

`DCODE_RELEASE_CHANNEL` continua aceita como grafia antiga de
`DCODE_UPDATE_CHANNEL`. É a única chave do produto com duas, e está dito por
extenso em vez de descoberto por quem tentar.

## A guarda

`TestEveryKnownKeyIsDeclaredInSomeSpec` fecha o laço no sentido que faltava. O
par completo:

| Guarda | Pega |
|---|---|
| `TestEveryKnownKeyIsAccountedFor` | chave em `KnownKeys` que código nenhum consome |
| `TestEveryKeyTheSpecsDeclareIsReadSomewhere` | chave que a spec promete e o código ignora |
| `TestEveryKnownKeyIsDeclaredInSomeSpec` | chave que o código honra e spec nenhuma menciona |

Nenhuma das três existia quando o defeito entrou. As três foram verificadas
quebrando o que deviam pegar.
