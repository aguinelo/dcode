# Regras por caminho e por comando

**Data:** 2026-08-09
**Specs afetadas:** `202608072336-sandbox-policy` (`.p`, `.config`), `202608081203-configuration` (`.p`), `202608072240-client-server-protocol` (`.p`)

## O que mudou

Nova camada de **atenção** sobre a de contenção: padrões de caminho e de comando
que fazem `Evaluate` escalar dentro do workspace, onde antes tudo passava em
silêncio.

`ApprovalRequest` ganha `Reason` e `Rule`. `Access` ganha `Rel`, o caminho
relativo ao workspace, preenchido por `Resolve`.

## Por que mudou

Medido antes de projetar: dentro do workspace, `src/main.go`, `.git/hooks/pre-commit`,
`.env` e `.dcode/config.toml` recebiam o mesmo `allow`. Três dessas não são um
arquivo de código:

- **`.git/hooks/`** — escrever ali é execução adiada **fora do sandbox**, como o
  usuário, no próximo `git commit`. É fuga da fronteira por outro caminho.
- **`.dcode/`** — configura o agente. Agente que edita a própria configuração
  amplia o próprio alcance.
- **segredos** — `read` manda o conteúdo ao provedor do modelo. Leitura *fora*
  do workspace já escalava; dentro, não. E `.env` mora dentro.

## O que estas regras não são

**Não contêm.** Um padrão de comando é contornado por `bash -c`, por alias, por
script. Um padrão de caminho só vê o caminho que a ferramenta declara. Quem
contém é o sandbox — foi ele que barrou escrita fora do workspace e a rede, não
o prompt.

As fronteiras se chamam `rule:write`, `rule:read` e `rule:command` justamente
para que ninguém leia uma recusa de regra como fronteira aplicada pelo SO.

## Duas decisões que só apareceram sob teste

**Regra dispara em `full-access`.** A primeira versão do teste assumia que não —
"o usuário pediu para não ser perguntado". Errado: pela ADR-02 os eixos são
ortogonais. `full-access` é o eixo do **sandbox**; regra é atenção, e atenção
vive no eixo de **aprovação**. Quem está em full-access com `on-request` pediu
para ser consultado — e sem mais nada segurando, a pausa é o que restou.

**`never` não transforma regra em negação.** `applyPolicy` converte escalada em
negação sob `never`, o que é certo para travessia real — com ninguém para
perguntar, negar é a leitura segura. Aplicado a regra, produziria o absurdo de
`never` ser **mais restritivo** que `on-request`: escrever em `.git/` passaria a
falhar em silêncio para quem escolheu "não me pergunte".

Regra é pedido de atenção de uma pessoa. Sem pessoa, não há pergunta. O sandbox
segue intocado nos dois casos, e é ele que contém.

## Impacto

- `Evaluate` ganha o parâmetro `Rules`. Assinatura, não campo escondido: é o
  núcleo de segurança, e o compilador aponta todo chamador.
- `allow session` passa a ser chaveado pela regra que casou. Editar três
  arquivos sob `.git/` é uma decisão, não três — e três perguntas é como se
  aprende a aprovar sem ler.
- O parser de TOML passa a aceitar array de strings, porque lista de padrões é
  lista de verdade. Item com vírgula é recusado: a vírgula é o que junta de
  volta.
- As regras entram na camada de defaults da configuração, e não só no código, de
  modo que `--config rules.confirm_write` responde valor **e** procedência.
  Regra que governa comportamento e não pode ser inspecionada é o buraco que o
  par de auditoria existe para fechar.

## Alternativa descartada

Lista de bloqueio de comandos como proteção. Descartada como proteção e mantida
como atenção, com a spec dizendo qual das duas é. Uma lista que parece conter e
não contém é pior que lista nenhuma: ensina a confiar onde não dá.
