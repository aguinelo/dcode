# Os contratos medidos, e o defeito que a medição encontrou

**Data:** 2026-08-27
**Specs afetadas:** `202608261730-done-qualifier` — a §8 sai do futuro e passa a
carregar resultado; um contrato é retirado; uma invariante nova.

> **Estado.** Três contratos medidos contra `MiniMax-M3`. Um fechou, dois não, e
> os limiares não desceram. `qualifier-narrows-on-mismatch` foi retirado.

## O defeito, achado ao montar o cenário

Um critério que voltava `broken` — saída 126 ou 127, "não havia o que rodar" —
era gravado no `done.toml` **declarado**:

```toml
# now: broken (exit 127) — nothing ran; this measures the absence of a tool
[lint]
command = "missing-linter ./..."
```

O arquivo é o que a execução seguinte carrega. Duas consequências, ambas becos
sem saída:

- A sessão de **trabalho** passava a ser medida contra um comando que não
  existe. Vermelho para sempre, e o laço nunca termina.
- A pasta passava a **declarar um critério**, então `Found.Pending()` a mandava
  para uma sessão de trabalho em vez de para a qualificação. Ela nunca mais
  voltava para a fase que consertaria o comando.

Gravado comentado, ele fica visível para quem revisa, fica fora da `DoneSet`, e
a pasta segue pendente sem declarar nada — que é exatamente o que a manda de
volta. A invariante entra na `.p §9` e o teste é
`TestABrokenCriterionIsWrittenDownAndNotDeclared`.

Foi achado **construindo** o `qualifier-fixes-broken`: o cenário exige uma pasta
que já passou por uma qualificação, e ao montá-la ficou claro que aquela pasta
nunca voltaria. Um cenário que não é alcançável no produto é um cenário que
mede outra coisa.

## `qualifier-narrows-on-mismatch` foi retirado

O `.p` dizia que ele não rodava porque o judge não existia. A razão verdadeira é
mais funda: **o cenário não existe mais**.

Ele descrevia um segundo turno do modelo reagindo à medição — um critério
declarado `fail` que volta `regression` "volta" para quem o escreveu, que
aperta. Era verdade quando o `.p` foi escrito, e a `Summary` ainda dizia ao
modelo "corrija propondo de novo".

O desenho A moveu a medição para **fora** do turno. Quem lê a discordância é a
pessoa. Trocar essa revisão por um segundo turno de modelo seria trocar uma
decisão não verificada por duas — a recusa mais antiga desta família.

A `Summary` foi corrigida junto: ela ainda falava com o modelo, e o modelo já
não está lá quando ela é escrita.

## O que as medições dizem

| ID | alvo | medido | |
|---|---|---|---|
| `qualifier-proposes-commands` | ≥ 95% | 96% de 50 | ✅ |
| `qualifier-declares-regression` | ≥ 90% | 85% de 20 | |
| `qualifier-fixes-broken` | ≥ 85% | 75% de 20 | |

**A fase produz comando e não frase, e isso está medido.** É a linha que
justifica a existência da fase, e ela fechou — por margem apertada, com 50
execuções porque o limiar ≥ 95% força esse piso.

**As duas falhas apontam para o mesmo lugar.** Em ambas, parte das execuções
**termina sem propor**: lê a spec, lê o código, escreve um raciocínio correto, e
encerra o turno sem chamar a ferramenta. Sempre depois de dizer que vai
verificar algo que o turno não consegue verificar:

> *"Let me verify whether `gotestsum` or `npm` is available in this project
> before proposing."*

Isso é o modelo aplicando a lição do critério quebrado — não proponha um comando
sem saber se ele roda — e travando nela. A instrução do turno manda ver *"what
the project can actually run"* e não fecha o caso em que a resposta é
inalcançável. É o próximo alvo, e é mudança de **texto do produto**, não de
cenário.

## Três medições para uma taxa, e duas eram do arcabouço

O `qualifier-fixes-broken` custou três execuções de vinte. Fica registrado
porque o padrão é o assunto:

| # | taxa | causa |
|---|---|---|
| 1 | 30% | o judge exigia o **nome** `slug`; o modelo mantinha a medição e trocava o rótulo |
| 2 | 75% | — |
| 3 | 65% | o cenário passou a oferecer `bash`, e toda execução abria com `ls -la` para levar a recusa da suíte |

Só a segunda mediu o modelo, e é a única em `measured.go`. As outras duas
mediram o harness, e um número que mediu o harness não é evidência sobre
comportamento.

A primeira é a mais instrutiva: `Proposed.Name` é, pelo comentário do próprio
tipo, "o que o relatório imprime". Um judge que exige o rótulo mede vocabulário
— que é precisamente o motivo pelo qual o contrato vizinho foi retirado. Os três
transcripts que a encontraram são hoje casos de unidade em
`TestKeepsMeasuringRefusesDroppingItAndRepeatingIt`, então o judge não pode
voltar a exigir o nome sem ficar vermelho antes de custar dinheiro.

A segunda foi pior: o comentário de `TestOnlyAScenarioThatNeedsTheShellIsOfferedOne`
já descrevia o efeito exato de oferecer um shell que a suíte recusa — onze
fixtures haviam passado por isso — e o shell foi oferecido assim mesmo.

## O que o arcabouço aprendeu

Um cenário de qualificação declara `{"qualify": "specs/x"}` e recebe o turno do
produto inteiro: a instrução vem de `tui.LoopTask`, a ferramenta do registro, a
resposta de `app.QualifyingTool`, e a fronteira de `app.QualifyMode` — plano,
leitura apenas. Nada disso é escrito no fixture, e um `task.md` numa pasta
dessas é **erro**: duas instruções para o mesmo turno é uma que diverge, e este
pacote já foi mordido quatro vezes por cópia de texto do produto.
