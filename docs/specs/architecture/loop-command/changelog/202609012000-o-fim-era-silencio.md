# O fim era silêncio

**2026-09-01** — RN-11: o fim da corrida é dito, com o estado dos critérios.

## O que havia

```go
func (p *program) nextSpec() tea.Cmd {
	if len(p.loopQueue) == 0 {
		p.loopGoal = ""
		return nil
	}
```

Nada. A corrida acabava e a tela não mudava.

De fora, um laço que trabalhou quatro specs e parou é **indistinguível** de um
que travou na quarta. "Acabou" lê exatamente como "está pensando", e quem
assiste não tem como separar os dois — a não ser esperando mais um pouco, que é
a experiência que este produto passa o tempo tentando não entregar.

## O que o aviso diz

Que acabou, quanto foi trabalhado, e onde os critérios ficaram:

```
o laço terminou — 4 specs trabalhada(s) · pronto: 2 de 4 critérios cumpridos
por cumprir       coverage
não conferido     integration
```

Contagem responde **quanto**. Nome responde **o que fazer agora** — e o fim da
corrida é exatamente quando essa é a pergunta.

## De onde vem o estado

Do evento de turno concluído, e não relido das entradas da tela.

A completude é declarada uma vez, por quem sabe, e viaja no protocolo
precisamente porque é *"a garantia que sobrevive a um modelo afirmando sucesso em
prosa"*. Reconstruí-la a partir do texto desenhado seria reintroduzir a prosa no
caminho do fato, e quebraria no dia em que o texto mudasse.

## Corrida que não correu não fala

A fila esvazia em toda conclusão de proposta, inclusive nas em que fila nunca
houve — `/loop <objetivo>` sem pasta nenhuma passa por ali. Anunciar o fim de uma
corrida que não começou seria linha sobre a funcionalidade existir, não sobre a
sessão. A contagem de specs trabalhadas é o que separa os dois casos.

## Invariantes

- `TestTheLoopSaysItFinishedAndWhereDoneStands`
- `TestARunThatWorkedNothingSaysNothing`
- `TestAFinishedRunWithEverythingMetStillReportsIt` — "4 de 4" é resposta, não
  omissão.
