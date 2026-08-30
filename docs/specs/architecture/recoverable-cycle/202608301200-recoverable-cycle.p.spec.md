# Um ciclo que sabe voltar — contrato técnico

**Família:** `recoverable-cycle`
**Data:** 2026-08-30
**`.r`:** [202608301200-recoverable-cycle.r.spec.md](202608301200-recoverable-cycle.r.spec.md)

---

## 1. O que muda, em uma frase

`Progressed` passa a devolver **três** respostas em vez de duas, e o laço desfaz
o ciclo quando a resposta é regressão.

Nenhuma máquina nova: `Snapshot`, `BeginTurn` e `Undo` já existem e já fazem a
parte difícil.

## 2. O sinal que falta

```go
// Movement is what one cycle did to the unmet set.
type Movement int

const (
    // MovedForward: the set shrank and everything left was already there.
    MovedForward Movement = iota
    // MovedNowhere: nothing got better and nothing got worse.
    MovedNowhere
    // MovedBackward: something that was met is not met any more.
    MovedBackward
)

func Moved(before, after []string) Movement
```

`Progressed` **sai**. A primeira versão desta seção disse que ele ficaria "porque
três chamadores o usam"; tinha **um**, e ele passa a ser `Moved`. A guarda de
nome exportado sem leitor pegou isso antes de virar invólucro morto — e é a
segunda afirmação desta spec que o código desmentiu.

**Regressão é `after` conter nome que `before` não tinha.** É a definição
estreita de propósito: o critério passava, agora não passa. Um conjunto que
cresce sem nome novo — impossível hoje, porque o conjunto é derivado dos mesmos
critérios — não é regressão, é o mesmo estado.

**Empate inclui a troca.** `{a,b} → {a,c}` tem nome novo e portanto é
regressão: `c` passava. Isto é a `.r` RN-4 aplicada, e é o caso que o
`Progressed` de hoje já recusa como progresso — a diferença é que agora ele tem
consequência.

## 3. O ponto de retorno por ciclo

`BeginTurn` zera o conjunto de instantâneos **por turno**, e um turno tem muitos
ciclos. Sem recorte por ciclo, desfazer volta ao começo do turno e joga fora
todo o trabalho bom que veio antes do ciclo ruim.

```go
// BeginCycle marks a point inside a turn that Undo can come back to.
func (s *State) BeginCycle()

// UndoCycle puts back what changed since the last BeginCycle.
func (s *State) UndoCycle() (restored, refused []string, err error)
```

**Duas camadas de instantâneo, não uma.** Esta seção dizia, na primeira versão,
que um caminho já tocado por um ciclo anterior *"fica com o que os ciclos
anteriores fizeram dele"*. Está errado e o teste derrubou: se o ciclo ruim
escreveu naquele arquivo, aquela escrita **faz parte do que regrediu**, e
preservá-la desfaz o ciclo pela metade.

As duas camadas respondem perguntas diferentes:

| camada | responde | quem restaura |
|---|---|---|
| do turno (`snaps`) | como isto estava antes de o modelo começar | `/undo` da pessoa |
| do ciclo (`cycleSnaps`) | como isto estava antes desta tentativa | o laço |

A do turno **não é substituída**. Trocá-la faria o `/undo` da pessoa restaurar um
estado que um ciclo posterior produziu, e isso não é desfazer.

**Um nível, não uma pilha.** O laço volta **um** ciclo — o último. Uma pilha de
pontos no tempo é uma interface, e esta família é um seguro.

**A recusa por arquivo permanece.** O `Undo` de hoje recusa arquivo que mudou no
disco depois do turno, por arquivo e não tudo-ou-nada, e essa é exatamente a
propriedade que uma reversão automática mais precisa: o laço não pode passar por
cima da edição de uma pessoa.

## 4. Quando o laço desfaz

No `checkDone`, depois de medir e antes de decidir se continua:

| medida | hoje | com esta família |
|---|---|---|
| avançou | `stall = 0` | `stall = 0` |
| empatou | `stall++` | `stall++` |
| **regrediu** | `stall++` | **desfaz o ciclo**, `stall++` |

**Regressão continua contando como ciclo parado.** Desfazer não é progresso, e
um laço que zerasse o contador ao desfazer poderia oscilar para sempre — o risco
declarado na `.r §5`.

**Desfazer não repete a medição.** O estado restaurado é o que a medição anterior
já classificou; medir de novo gastaria um ciclo para reconfirmar o que se sabe.

Isto **não** está na §7: uma invariante que nenhum teste reivindica não é
verificável, e a guarda recusou listá-la. Fica como prosa, que é o que ela é —
e é a única linha desta spec que vale o que vale a disciplina de quem lê.

## 5. O que o modelo é informado (RN-5)

Um lembrete novo, na mesma família dos que existem:

```
The last cycle was undone. <names> passed before it and did not after,
so what it wrote was put back. <n> file(s) restored; <m> left alone
because they changed on disk since.

Try something else — the same edit will be undone the same way.
```

**A última frase é o ponto.** Sem ela o agente repete a tentativa achando que
ela nunca aconteceu, o que transforma o seguro numa armadilha.

**Arquivos recusados são nomeados.** Um estado meio revertido é pior que nenhum
se ninguém souber que é meio.

## 6. O que NÃO muda

- `Undo` continua fora do registro de ferramentas. RN-3, e já é assim.
- `/undo` continua sendo da pessoa e continua alcançando o turno inteiro.
- `BeginTurn` e o `Undo` de turno ficam como estão.
- Nada em git.
- O teto de ciclos parados continua em 2. É a etapa 4 da §9 e depende disto.

## 7. Invariantes verificáveis

> As etapas 1, 2 e 3 da §9 estão entregues, e estas são reivindicadas por
> `specguard.Check` em `internal/loop/invariants_test.go`.

- `Moved` distingue avanço, empate e regressão, e é a única definição disso.
- Trocar uma falha por outra é **regressão**, não empate.
- Conjunto que esvazia é avanço, nunca regressão.
- `BeginCycle` não apaga o que o turno guardou: `/undo` da pessoa continua alcançando o turno.
- `UndoCycle` restaura só o que **este ciclo escreveu**, e ao estado do começo dele.
- Caminho escrito em dois ciclos volta ao que o ciclo anterior deixou, não ao começo do turno.
- `UndoCycle` recusa, por arquivo, o que mudou no disco desde então.
- O laço desfaz em regressão e **não** desfaz em empate.
- Regressão desfeita continua contando como ciclo parado.
- O modelo é informado de que foi desfeito, de quais critérios regrediram, e de quais arquivos ficaram.
- Nada é restaurado sem dizer.
- O modelo não tem ferramenta que desfaça.

## 8. Contratos comportamentais

**Nenhum novo.** Tudo aqui é determinístico: classificar, restaurar, avisar.

### E nenhum existente mede isto — a etapa 4 estava errada

A primeira versão desta seção disse que o efeito seria medido por
`states-unmet-on-stall` e `fixes-cause-not-measure`. **Não é, e não pode ser.**

O arcabouço de eval **não executa `checkDone`**. Ele monta o prefixo, injeta o
lembrete pronto e observa a resposta — `Moved`, `BeginCycle` e `UndoCycle` nunca
rodam numa medição. E o texto injetado é byte a byte o mesmo, com teste
provando.

Foi medido assim mesmo, e o resultado é a prova disso:

| contrato | antes | depois | |
|---|---|---|---|
| `fixes-cause-not-measure` | 100% de 50 | 100% de 50 | não mudou |
| `states-unmet-on-stall` | 94% de 50 | 90% de 50 | dois turnos, ruído |

Dois turnos em cinquenta num caminho de código que a medição não percorre. A
única leitura honesta é que **nada foi medido sobre esta família**.

**O que faltaria para medir**: um cenário em que o arcabouço rode o ciclo de
verificação de verdade — critérios reais, um turno que quebra um deles, e a
reversão acontecendo — em vez de injetar o lembrete que o ciclo produziria.
Isso é máquina nova no `internal/evals`, não um contrato novo, e é o mesmo
buraco que a `failure-feedback` deixou registrado: **nenhum contrato mede a
qualidade de uma correção**.

Fica escrito que esta família está **entregue e não medida**, e que isso é
diferente de medida e boa.

**O contrato que falta continua faltando**, e é o mesmo da `failure-feedback`:
nenhum contrato mede a **qualidade de uma correção**. É a etapa 2 da §9 e não se
resolve aqui.

## 9. Ordem de entrega

1. **`Moved`, e `Progressed` escrito sobre ele.** Puro, asserção, nada muda de
   comportamento.
2. **`BeginCycle` e `UndoCycle`.** Puro sobre disco, testável sem modelo.
3. **O laço desfazendo, e o lembrete.** É aqui que o comportamento muda.
4. ~~Medir `states-unmet-on-stall` e `fixes-cause-not-measure`.~~ **Riscada**:
   a §8 explica por quê. O que resta é construir o cenário que roda o ciclo, e
   isso é trabalho no arcabouço.

**1 e 2 não mudam nada que um modelo veja**, e por isso podem ir juntos. 3 vai
sozinho, pela mesma razão que a etapa 2 da `failure-feedback` foi sozinha.

## 10. Changelog

- [202608301200 — o laço não sabia voltar](changelog/202608301200-o-laco-nao-sabia-voltar.md)
