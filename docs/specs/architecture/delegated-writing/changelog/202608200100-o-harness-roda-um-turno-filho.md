# O harness roda um turno filho

O conserto do instrumento que quatro medições apontaram e nenhuma conseguiu
contornar.

## O que estava errado

O harness recusava toda chamada delegada. A recusa era **honesta** — ele não
fingia que um filho tinha rodado, que é a lição do `shellRefusal` — mas a frase
que ele devolvia era:

> *"the eval harness does not run delegated turns. **Do the reading yourself
> with the tools you have**, and say what you could not cover."*

Isso não recusa: **instrui o abandono**. Um `explore` de reconhecimento recebia
essa frase, o modelo acreditava, e a delegação sumia do resto da execução. Um
contrato sobre *procurar delegação* estava medindo o harness convencendo o
modelo a não procurar.

O rastro está nos digests de toda execução reprovada, escrito pelo próprio
modelo: *"The eval harness doesn't run delegated turns, so I'll do the work
directly."*

## O que passou a existir

`Workspace.Delegate` — um gancho, com a mesma forma do `Delegator` do produto:
a ferramenta declara, e quem tem o provider decide o que um turno filho pode
ser. Nulo ainda recusa, porque um harness sem delegador **tem** de dizer isso em
vez de inventar resultado.

O filho roda contra **o mesmo provider** que o pai:

- recebe as instruções **do produto**, `loop.DelegateInstructions`, importadas e
  não copiadas — o mesmo motivo pelo qual `BudgetText` é exportado: harness que
  parafraseia mede a paráfrase, e a cópia diverge sem ninguém ver;
- recebe **a tarefa, não o histórico do pai** — copiar devolveria exatamente o
  custo que delegar existe para evitar;
- recebe as ferramentas de leitura, mais as de escrita quando possui caminhos;
- tem teto próprio de **4 rodadas**, pelo motivo do produto (um filho faz **um**
  pedaço) e por um segundo que é do harness: cinco filhos em cinquenta execuções
  são duzentos e cinquenta turnos filhos, e cada rodada de cada um é uma chamada
  paga;
- devolve os caminhos que leu junto da conclusão, como o do produto faz.

Um filho que falha volta **nomeado** — `the delegated turn failed: … (task: …)`
— que é o assunto inteiro de `names-the-child-that-did-not-answer`.

## O que ele deliberadamente não reproduz

**A contenção estreitada do filho.** Narrar o resolver para `owns` vive em
`internal/policy` e é asserido lá: por teste unitário e contra o kernel, com
controle. O harness não repete isso, e não precisa — o que os contratos medem
aqui é **a decisão do pai de dividir**, e ela é visível na chamada emitida antes
de qualquer filho rodar.

Dizer isso importa: um harness que finge reproduzir a fronteira seria a mesma
mentira do `shellRefusal`, com outra roupa.

## O que ainda não se sabe

Se o número muda. As medições anteriores — 75%, 66,7%, 50% e 50% de cinquenta —
foram todas contra o harness que desencorajava delegar. A próxima é a primeira
que mede o dcode em vez da mistura, e o limiar em 25% continua onde está até que
ela exista.

---

## Correção: a previsão não se sustentou

Este changelog terminava com *"se o número muda"* em aberto. Mudou pouco, e no
contrato errado.

| contrato | harness que recusava | harness que roda o filho |
|---|---|---|
| `keeps-writing-that-must-cohere` | 96,0% | **100,0%** |
| `names-the-child-that-did-not-answer` | 98,0% | **100,0%** |
| `delegates-writing-when-disjoint` | 50,0% | **52,0%** |

Cinquenta execuções cada, 92 minutos.

**A explicação escrita acima estava errada.** Afirmei que a recusa convencia o
modelo a parar de delegar e que consertá-la faria o terceiro número subir. Com
n=50 a dispersão é de cerca de sete pontos: **dois pontos é ruído.** A causa das
não-delegações é outra e não é conhecida.

O que a evidência sustentava — o digest dizendo *"the eval harness doesn't run
delegated turns, so I'll do the work directly"* — mostrava o modelo **explicando
o que fez**, não necessariamente a razão de ter feito. Tomei uma racionalização
por causa, que é o erro que este repositório documenta em outros lugares e que eu
cometi aqui.

**O conserto não foi inútil, e o ganho está nos outros dois.** Agora o modelo tem
uma opção de delegação que funciona, e **ainda assim** recusa dividir trabalho que
precisa concordar consigo. Antes ele recusava num mundo onde delegar era
impossível — o que media muito menos, e é por isso que 96% virou 100%.

O terceiro contrato segue medindo o que declara medir, com o limiar em 25% e o
mesmo aviso: piso contra regressão, não certificado de qualidade. Por que o dcode
divide o trabalho em metade das execuções continua sendo pergunta aberta, e agora
sem a resposta fácil que eu tinha inventado.
