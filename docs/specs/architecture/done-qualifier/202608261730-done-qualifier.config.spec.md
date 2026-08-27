# Config: Qualificação da definição de pronto

> Nenhuma variável de ambiente nova no código sem estar aqui.
> Precedência: flag > variável de ambiente > arquivo de config > default.

## 1. Chaves previstas — **nenhuma declarada ainda**

Este arquivo não declara chave nenhuma. As que a família vai precisar estão
descritas abaixo em prosa, e entram como **linha de tabela** no PR da etapa que
as lê.

A distinção não é estilística. Neste repositório, uma linha de tabela num
`.config.spec.md` **é** a declaração: `TestEveryKeyTheSpecsDeclareIsReadSomewhere`
lê as linhas e reprova a chave que nenhum código consome. A regra existe porque
superfície de configuração declarada e não implementada é pior que ausente — ela
promete um controle que não está lá, e o valor é lido, resolvido, exibido por
`dcode config` e ignorado. Em 2026-08-11 havia 64 chaves nesse estado.

É a mesma forma das "Invariantes previstas" do `.p §9`, pelo mesmo motivo: o que
ainda não existe não é reivindicado como existente. Descrever em prosa é
descrever; declarar em tabela é prometer.

> **Não fazer:** mover a chave para a segunda coluna da tabela para o padrão não
> casar. Isso é derrotar a guarda por formatação, que é exatamente o defeito que
> a revisão da `loop-command` encontrou num teste que comparava uma constante
> consigo mesma. A guarda pede uma coisa e ela deve receber a coisa, ou não
> receber nada.

### 1.1 Ligar a fase — etapa 2

**`DCODE_QUALIFIER_ENABLED`**, bool, default `false`. Liga a fase de
qualificação. Desligada, o produto se comporta exatamente como hoje.

Nasce **desligada**, e é o único lançamento honesto: a fase gasta um turno de
modelo e bloqueia esperando um humano. Ligada por default, ela transformaria
todo turno de todo mundo numa pergunta.

**`DCODE_QUALIFIER_MEASURE_EXISTING`**, bool, default `true`. Mede em t=0 os
critérios das origens que já existem (`done.toml`, `tasks.md`, `verifyCommand`).
Não pede assinatura para elas — quem escreveu já assinou escrevendo (RN-9 da
`.r`).

Nasce **ligada** porque é barata, não pede nada a ninguém, e é o que responde a
pergunta da §10 do `.p` com dados de uso real.

### 1.2 Tetos da fase — etapas 1 e 2

**`DCODE_QUALIFIER_MAX_PROPOSALS`**, inteiro, default `3` (etapa 3). Quantas
vezes o modelo pode chamar `done_propose` no mesmo turno. Esgotado, a última
proposta medida vai ao operador com a contagem à vista.

**`DCODE_QUALIFIER_MAX_SIGN_ROUNDS`**, inteiro, default `3` (etapa 2). Quantas
vezes a proposta volta ao operador quando a edição dele muda a classe de algum
critério. Esgotado é **recusa**.

**`DCODE_QUALIFIER_SIGN_TIMEOUT`**, duração, default `30m` (etapa 2). Prazo da
assinatura. Esgotado é **recusa**. Zero desliga o prazo, e é a única forma de a
fase esperar indefinidamente.

**`DCODE_QUALIFIER_OUTPUT_LIMIT`**, bytes, default `2000` (etapa 1). Quanto da
saída de cada critério chega ao operador. Truncado diz que foi truncado.

## 2. O que os tetos recusam, e o que nunca aprovam

**Os dois tetos e o prazo recusam ao estourar, e isso não é configurável.**

Um prazo que aprova é a forma mais silenciosa de a RN-6 da `.r` ser quebrada: o
turno começa contra uma régua que ninguém leu, e o relatório do fim diz pronto.
O mesmo vale para um teto de rodadas que, esgotado, aceita o último estado.

O custo de recusar é o operador digitar de novo. O custo de aprovar sozinho é um
agente trabalhando contra uma régua que ninguém viu.

## 3. O que **não** é chave nova

O teto de um critério continua sendo `DCODE_DONE_TIMEOUT`, declarada em
`202608072335-agent-loop.config.spec.md` §1.1. Medir um critério em t=0 e
verificá-lo no fim do turno são o mesmo ato contra o mesmo comando; dois tempos
limite para isso seriam dois comportamentos para o mesmo conceito.

Pelo mesmo motivo, os limites do turno de qualificação são os da `agent-loop`.
A fase não tem orçamento próprio.

## 4. Medição de contratos comportamentais

**Os três contratos desta família foram medidos** contra `MiniMax-M3` em
2026-08-27, com `DCODE_EVAL_MODEL`, `DCODE_EVAL_RUNS` e `DCODE_EVAL_ENABLED` —
declaradas em `202608072334-provider-adapter.config.spec.md` §4, não
redeclaradas aqui. Os resultados estão na `.p §8`, com modelo e data em
`internal/evals/measured.go`.

Um deles fechou; dois não. Os limiares ficaram onde estavam.

`qualifier-narrows-on-mismatch` **foi retirado**, e o motivo mudou entre a
escrita desta seção e a medição. Não é mais que o judge seria difícil: é que o
cenário dele não existe no produto. A `.p §8` explica.

**O contrato que precisou de três medições diz mais que os outros dois.** Duas
delas mediram o arcabouço — um judge que exigia o nome do critério, e um
cenário que oferecia um shell que a suíte recusa — e só a terceira mediu o
modelo. Isso é o regime funcionando: uma taxa baixa é tão frequentemente uma
afirmação sobre o cenário quanto sobre o modelo, e só a evidência guardada
separa as duas. Nenhuma das duas primeiras entrou em `measured.go`.

## 5. Constantes não configuráveis

| Constante | Valor | Motivo |
|---|---|---|
| Saídas que produzem `ClassBroken` | 126 e 127, e falha ao iniciar | São a resposta do shell para "não havia o que rodar". É o que faz um critério quebrado se disfarçar de vermelho. |
| `ClassRegression` compara com `ExitCode` | sempre | `ExitCode` é o que conta como cumprido. Comparar com zero classificaria como aceitação um critério já verde declarado com `exit: 1`. |
| Proposta vazia é erro | sempre | `DoneSet` vazia é "sem definição de pronto", que o ciclo relata como pronto. Ausência de definição não é definição permissiva. |
| `NoAcceptance` é aviso, nunca recusa | sempre | Refatoração legítima não tem critério de aceitação. O harness não sabe distinguir isso de proposta vazia de conteúdo, então nomeia e o operador assina sabendo. |
| Critério editado é medido de novo | sempre | Sem isso a edição do operador escapa da regra que a família existe para aplicar. |
| Toda proposta fica no `Record` | sempre | Apertar depois de ver passar é certo; trocar por trivial é errado; só a sequência distingue. |
| `done_propose` só no turno de qualificação | sempre | Uma ferramenta que redefine o pronto ao alcance de um turno de trabalho é a saída curta da RN-7. |
| `SourceQualified` fora do `SourceAuto` | sempre | Qualificar é interativo e custa um turno. Cair nela por omissão surpreende quem só queria o comando legado. |
| A `DoneSet` assinada é imutável no turno | sempre | O relatório do fim tem que ser sobre a régua que o operador viu no começo. |

## 6. Changelog

- [202608261730 — a definição de pronto passa a ter uma fase que a levanta](changelog/202608261730-qualificacao-antes-do-laco.md)
- [202608261900 — o contrato técnico da qualificação](changelog/202608261900-contrato-da-qualificacao.md)
- [202608271200 — os contratos medidos](changelog/202608271200-os-contratos-medidos.md)
