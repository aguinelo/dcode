# O retorno do erro

**Família:** `failure-feedback`
**Data:** 2026-08-28
**Estado:** `.r` — o problema e as regras. Sem `.p`, sem código.

---

## 1. Contexto

O `dcode` detecta bem e devolve mal.

A `agent-loop` roda a `DoneSet` ao fim de cada ciclo, sabe exatamente qual
critério falhou, e diz ao modelo:

> *"You changed files and this is not done yet: **tests** did not pass. Fix the
> cause. Do not weaken the check to make it pass, and do not report success — if
> you cannot get there, say what is left."*

O texto está certo em tudo menos no que omite. O modelo é informado **de que**
falhou e nunca **de por quê**. Sem stderr, sem a asserção que quebrou, sem
arquivo, sem linha.

E a informação existia. `internal/loop/done.go`:

```go
code, _, err := run(cctx, c.Command)
                ↑
```

O `CriterionRunner` devolve `(int, string, error)` — código, **saída**, erro. O
`Check` descarta a saída num `_` e monta um `Report` que carrega só
`map[string]CriterionState`. A evidência é colhida e jogada fora na mesma linha.

**A fase vizinha faz o contrário.** O `qualifier.Measured` guarda `Output`,
truncado em `MaxOutput`, e o escreve no `done.toml` para a pessoa ler — porque
*"é a única coisa que separa um critério vermelho porque falta trabalho de um
vermelho porque falta o mundo"*. A fase que só propõe preserva a evidência; a
fase que corrige, não.

### O que isso custa, observado

A medição de 2026-08-27/28 encontrou, em três contratos de duas famílias, turnos
que leem tudo, raciocinam certo e **terminam sem fazer o ato**. A `working-defaults`
RN-9 atacou metade disso — o caso em que a verificação é impossível — e mediu
~10 pontos de ganho. A outra metade é esta: um modelo que sabe que falhou e não
sabe o que quebrou tem menos a fazer do que parece.

## 2. Fronteira de determinismo

**Regime: determinístico**, inteiro. É a razão de esta família ser pequena e de
não ter contrato comportamental próprio.

| Parte | Regime | Verificação |
|---|---|---|
| Capturar a saída do critério que falhou | determinístico | asserção |
| Truncá-la e dizer que truncou | determinístico | asserção |
| Entregá-la ao modelo no lembrete | determinístico | asserção |
| Decidir se houve progresso entre ciclos | determinístico | asserção |
| Decidir quando desistir | determinístico | asserção |
| **O que o modelo faz com a saída** | já existe (mediado) | contratos da `agent-loop` |

Nada aqui é mediado. O que muda é **o que o modelo recebe**, e o efeito disso é
medido pelos contratos que já existem, não por contratos novos.

## 3. User stories

**US-1.** Como agente, quando um critério falha, quero ver **o que o comando
imprimiu**, para poder corrigir a causa em vez de ir procurá-la.

**US-2.** Como operador, quero que um critério que foi de quarenta falhas para
uma conte como progresso, para o laço não desistir de um trabalho que está
andando.

**US-3.** Como operador, quero que o teto de paciência do laço seja generoso na
medida em que o agente está bem informado — hoje ele é apertado porque o agente
está cego, e essas duas coisas devem andar juntas.

## 4. Regras de negócio

### RN-1 — A saída do critério que falhou chega ao modelo

Truncada, com o mesmo teto e a mesma honestidade do qualificador: cortada é
cortada **e diz que foi**.

Só a dos critérios que **não passaram**. A saída de um critério verde é ruído
pago em toda rodada, e o que ele tem a dizer já está dito pelo fato de ter
passado.

### RN-2 — A saída é evidência, nunca instrução

O que um comando imprime é texto de terceiros: vem de um teste, de um linter, de
um script do projeto. Ele entra no prompt como **resultado observado** e nunca
como algo a obedecer. Uma suíte cuja mensagem de erro diga "ignore as instruções
anteriores" é uma suíte com um teste mal escrito, não uma ordem.

Esta regra existe porque a superfície é nova: hoje nenhuma saída de comando
entra no contexto por este caminho.

### RN-3 — Critério que não imprime nada nomeia o seu comando

Encontrado rodando o produto contra um workspace de verdade. Um `done.toml` com

```toml
[changelog]
command = "test -f CHANGELOG.md"
```

falha **em silêncio**: `test -f` não escreve nada. A saída fica vazia, o bloco da
RN-1 não renderiza, e o modelo recebe exatamente o que recebia antes desta
família existir — o nome, e nada.

No teste de campo ele se virou: foi ler o `.dcode/done.toml` para descobrir o que
`changelog` queria dizer. Duas rodadas e duas chamadas de ferramenta para achar
uma informação que o produto **já tinha na mão** — o comando está no
`Criterion.Command`, e foi o laço que o executou.

Nomear o comando quando não há saída não é dizer mais; é parar de esconder o que
já se sabe. E é a diferença entre "changelog falhou" e "changelog é
`test -f CHANGELOG.md`, e falhou".

**Não substitui a saída quando ela existe.** Saída é evidência do que aconteceu;
o comando é só a identidade do critério. Mostrar os dois sempre gastaria contexto
repetindo o que o nome já sugere na maioria dos casos.

### RN-4 — Truncar pelo fim, não pelo começo

A informação que interessa num relatório de teste está no fim: o resumo, a
contagem, a última asserção. Cortar o fim para preservar o cabeçalho é preservar
exatamente a parte que não decide nada.

### RN-5 — Progresso não é só cardinalidade

Hoje `Progressed` exige que o conjunto **encolha** e seja **subconjunto** do
anterior. As duas metades estão certas e são insuficientes: o mesmo critério,
agora quase passando, é indistinguível de um que não se moveu.

O que conta como aproximação é do `.p`. O que a `.r` fixa é que **existe** um
terceiro estado entre "encolheu" e "parado", e que o laço não pode chamar os
dois de stall.

### RN-6 — O teto de paciência sobe **depois**, nunca antes

`MaxStallCycles` é 2 hoje. Ele é apertado porque o agente está cego: dois ciclos
sem progresso com um agente que não sabe o que quebrou é uma decisão razoável.

Subir o teto antes de a RN-1 existir compraria mais ciclos de tentativa às
cegas, que é gastar dinheiro para adiar a mesma desistência. A ordem é lei
aqui.

### RN-7 — Nada disto devolve a decisão de "pronto" ao modelo

A RN-10 da `agent-loop` continua inteira. O modelo não julga se terminou; ele
passa a **ver o que a régua viu**. A régua continua sendo o comando, o comando
continua rodando no sandbox, e o veredito continua sendo o código de saída.

## 5. O risco que fica, e o que não o cobre

**A saída de um comando pode ser enorme e pode ser lixo.**

Uma suíte com quatrocentos testes falhando imprime um relatório que não cabe, e
truncá-lo pode deixar de fora justamente o que importa. O teto é uma escolha
sem resposta certa: pequeno demais e a evidência não serve, grande demais e o
contexto vai embora numa rodada.

O que **não** cobre isso: escolher um número e declará-lo bom. O que cobre
parcialmente é a RN-4 — cortar pelo fim é o palpite menos ruim, porque é onde os
executores de teste põem o resumo — e dizer que cortou, para o modelo saber que
há mais.

**O segundo risco é de incentivo, e é mais sutil.** Um agente que vê a mensagem
exata do teste tem um caminho novo para o desonesto: mudar o teste até a
mensagem sumir. A doutrina já proíbe (*"do not weaken the check"*) e o
`Protected` já sinaliza quando um arquivo de medição é tocado. Esta família
**aumenta a pressão sobre essas duas defesas** sem acrescentar nenhuma, e isso
fica escrito antes de construir.

## 6. Fora de escopo

- **Checkpoints por spec num `/loop <objetivo>`.** É o outro buraco do ciclo
  longo — cinco horas de trabalho deixam uma árvore suja e nenhum ponto de
  retorno — e é uma família própria, porque mexe com a fronteira declarada de
  que o `vcs` deste produto lê e não escreve. Não se resolve de passagem aqui.
- **Um segundo modelo conferindo o primeiro.** Mesma recusa da `done-qualifier`:
  troca uma decisão não verificada por duas. O que falta neste laço não é um
  opinador, é o executor devolvendo o que já mediu.
- **Diagnosticar a falha.** O laço entrega a saída; interpretar é do agente.
  Um harness que classificasse tipos de erro seria uma máquina que este produto
  não quer, e que erra em silêncio.
- **Guardar a saída em disco.** O `done.toml` é da qualificação e é revisado por
  uma pessoa. A saída de um ciclo é efêmera e pertence ao event log da sessão,
  que já existe.

## 7. Changelog

- [202608281900 — o erro que não voltava](changelog/202608281900-o-erro-que-nao-voltava.md)
- [202608282000 — a saída fica](changelog/202608282000-a-saida-fica.md)
