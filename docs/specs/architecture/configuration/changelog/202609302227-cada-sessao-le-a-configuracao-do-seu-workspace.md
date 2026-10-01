# Cada sessão lê a configuração do seu workspace

**Data:** 2026-09-30
**Specs afetadas:** `202608081203-configuration` (`.p`: duas linhas novas em §7;
a tabela de raízes da §2 deixa de dizer que o socket mora no estado)
**Fonte:** o pedido N2 do cliente desktop (`desktop/docs/DECISIONS.md`, D20),
decidido com a pessoa em 2026-09-29.

## O defeito

Um daemon resolvia a cadeia de configuração uma vez, no workspace em que subia,
e toda sessão usava essa — `d.opts.Base`, copiado com o workspace trocado. Num
daemon só servindo vários projetos, uma sessão rodava sob as regras de um
projeto em que não estava:

- **o desktop** sobe o daemon fora de qualquer projeto, então nenhuma sessão
  dele lia o `.dcode/config.toml` do projeto em que abria;
- **uma TUI anexada** a um `dcode serve` que subiu em outro projeto rodava com a
  configuração daquele outro — o modo, a política, o modelo, o prazo dos
  critérios.

Na `main`, com um projeto pedindo `read-only` e prazo de 3 minutos:

```
the session in <outro> runs "workspace-write", from the workspace the daemon
started in, not the read-only its own project asks for
criteria in <outro> are timed at 10m0s, not the 3m its project sets
a session opened in a workspace whose configuration cannot be read
```

## O que mudou

- **O daemon resolve a cadeia no workspace de cada sessão** — padrão, arquivo do
  usuário, arquivo do projeto, ambiente, travado — com o mesmo `FromEnv` que a
  linha de comando usa. Vale para os quatro lugares que copiavam o `Base`: a
  sessão, a qualificação, a medição dos critérios e a lista de specs.
- **Configuração de projeto que não se lê recusa a sessão**, com
  `workspace_invalid` e o que está errado, em vez de abri-la com a configuração
  do daemon. Uma sessão que ignorasse em silêncio o arquivo do projeto rodaria sob
  regras que ninguém naquele projeto escolheu.
- **A lista de specs** passa a dizer por que não respondeu, quando o workspace
  existe mas não se consegue ler a configuração dele ou confinar os critérios. Um
  caminho que não é workspace continua respondendo "nada", como antes.
- **Valor travado** que alguém tentou sobrescrever é dito no log do daemon, com o
  workspace, como a linha de comando já dizia.

O `Base` continua sendo a cadeia de onde o daemon subiu: é dele que saem o que é
do daemon e não da sessão — onde gravar o registro, os avisos de boot. Opções
montadas à mão, sem ambiente de onde resolver, são usadas como vieram: é um teste
montando as suas, e não há cadeia a ler.

## A invariante

As duas linhas novas em §7 são reivindicadas por
`TestEachSessionReadsTheConfigurationOfItsOwnWorkspace` e
`TestAProjectConfigurationThatCannotBeReadRefusesTheSession`, pela montagem do
próprio daemon. `TestDaemonBuildResolvesAConfiguredProfileAsOneBundle` passa a
configurar o perfil pelo `models.toml`, como uma pessoa configura: um perfil
posto à mão nas opções de boot não chega mais a uma sessão, e não deveria.

Correção de comportamento, sem campo novo no protocolo: PATCH.
