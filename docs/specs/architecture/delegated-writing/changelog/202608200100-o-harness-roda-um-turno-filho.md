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
