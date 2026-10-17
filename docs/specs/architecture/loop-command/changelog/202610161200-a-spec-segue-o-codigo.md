# A spec segue o código: flags de limite e chaves `loop.*` saem

**Data:** 2026-10-16
**Specs afetadas:** `202608252000-loop-command` (`.p` §2, §5, §6, §8; `.config`
estado, §1, §2, §3; `.i` estado, Passos 3 e 4).
**Decisão do dono:** a spec segue o código.

## O que a spec declarava e o código nunca fez

**Flags de limite.** A `.p` §6 e a `.config` §1 declaravam `--max-iterations`,
`--max-identical`, `--max-tokens`, `--max-stall` e `--done-timeout` como flags
de `/loop`, com precedência sobre a configuração. O `ParseLoopArgs` só aceita
`--protect`, e o teste dele afirma que `--max-iterations` é flag desconhecida.
A spec prometia um controle que, digitado, para o comando com erro.

**Chaves `loop.*`.** A `.config` §2 e §3 declaravam `DCODE_LOOP_SPEC_PATH`,
`DCODE_LOOP_SOURCE`, `DCODE_LOOP_PROTECT` e `DCODE_LOOP_SESSION_PREFIX`. Estavam
em `KnownKeys` — portanto aceitas, resolvidas e mostradas por `dcode config`
com origem — e eram lidas só por `OptionsFromConfig`, que não tinha chamador:
existia para o guard de fiação achar os literais no código. Pior, uma origem
inválida virava `SourceAuto` em silêncio, contra a própria `.config`, que dizia
"inválido é erro fatal no startup". `SessionConfig`, que consumiria o prefixo,
também não tinha chamador: a sessão do loop é criada pelo daemon.

## O que mudou

- As cinco flags saem da `.p` §6 e da `.config` §1. A única flag é `--protect`;
  os limites de cada sessão do loop vêm da família e da configuração, pela
  mesma resolução de qualquer sessão. A suíte agora afirma que cada uma das
  cinco é flag desconhecida, e não só `--max-iterations`.
- As quatro chaves saem da `.config` e de `KnownKeys`. Com elas saem
  `OptionsFromConfig`, o tipo `Options`, `ParseSource` (só ela o chamava),
  `SessionConfig`, `SessionOptions` e `sessionName`, e os testes que existiam
  só para eles.
- As isenções correspondentes saem dos guards: as quatro entradas de
  `nonSession` no guard de fiação, e `SessionConfig` e `OptionsFromConfig` em
  `exportedWithoutUser`. Nenhum guard foi afrouxado para outra chave — as
  entradas saem porque o que elas desculpavam deixou de existir.
- As quatro invariantes da `.p` §8 que descreviam `SessionConfig` saem com ele.
  A de que a fachada não tem orçamento próprio continua verdadeira, e está
  escrita na `.p` §6.

## Versão

Remover chave de configuração e flag declarada é mudança na superfície que a
versão cobre (`VERSIONING.md` §2): **no mínimo MINOR**. Nenhum usuário perde
comportamento — as flags nunca foram aceitas, e as chaves nunca mudaram nada —,
mas `dcode config` deixa de listar as quatro, e um `config.toml` que traga
alguma delas passa a ser recusado com `unknown key`, como qualquer chave fora
do esquema.
