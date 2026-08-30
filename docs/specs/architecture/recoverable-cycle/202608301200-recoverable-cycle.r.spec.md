# Um ciclo que sabe voltar

**Família:** `recoverable-cycle`
**Data:** 2026-08-30
**Estado:** `.r` — o problema e as regras. Sem `.p`, sem código.

---

## 1. Contexto

O laço é **fechado na detecção e aberto na recuperação**.

Ele roda a `DoneSet` a cada ciclo, sabe qual critério falhou, e desde a
`failure-feedback` sabe dizer o que quebrou. O que ele não sabe fazer é
**voltar**. Cada ciclo escreve por cima do anterior, e a única direção é para
frente.

Isso tem duas consequências, e a segunda é a que dói.

**Um ciclo que piora não tem desfecho.** O `Progressed` distingue "encolheu e é
subconjunto" de todo o resto — e "todo o resto" junta empatar, regredir, e
trocar uma falha por outra. Os três viram `stall++`. O laço **sabe** que piorou
e a informação morre ali:

```go
if Progressed(*unmet, now) || *unmet == nil {
    *stall = 0
} else {
    *stall++
}
```

### A metade que já existe, e que esta spec quase reinventou

**O instantâneo já existe, e o desfazer também.** A primeira versão desta seção
dizia que o ponto de retorno não existia. Estava errada, e o erro foi ler
`Progressed` e `State.Written()` sem ler o que os vizinhos fazem:

| peça | onde |
|---|---|
| `Snapshot(path)`, antes de cada escrita, só o primeiro toque | `tools/undo.go` |
| `BeginTurn()`, zerando o conjunto a cada turno | `turn.go:285` |
| `Undo()`, restaurando conteúdo e removendo o que foi criado | `tools/undo.go` |
| recusa de arquivo que mudou no disco depois do turno | `Undo()` |
| por arquivo, nunca tudo-ou-nada | `Undo()` |
| `/undo` para a pessoa | `tui`, `server` |

O comentário no `file.go`, escrito muito antes desta família, já dizia o porquê:
*"How it stood before, so the turn can be undone."*

E não é commit, não toca em git, não move branch — a fronteira de que o `vcs` lê
e não escreve **já estava respeitada** sem que ninguém precisasse descobrir isso
de novo.

### O que falta, então

Três coisas, e as três são pequenas porque a máquina está pronta:

1. **O laço nunca desfaz por decisão própria.** `Undo()` tem três chamadores —
   o servidor, a sessão e o motor — e todos servem ao `/undo` que **a pessoa**
   digita. Nada no ciclo o aciona.
2. **Não há sinal para acionar.** `Progressed` devolve um booleano onde cabem
   três respostas, então "regrediu" chega ao ciclo indistinguível de "empatou".
3. **O escopo é o turno, não o ciclo.** `BeginTurn` zera o conjunto a cada
   turno; um laço que roda vários ciclos dentro de um turno tem um ponto de
   retorno só, e é o do começo do turno.

Um `/loop <objetivo>` de horas continua deixando uma árvore suja: o desfazer
alcança o último turno, e não a décima sétima spec.

## 2. Fronteira de determinismo

**Regime: determinístico**, inteiro — e isso é deliberado, não conveniente.

| Parte | Regime | Verificação |
|---|---|---|
| Registrar o conteúdo anterior no primeiro toque de um caminho | determinístico | asserção |
| Tirar o instantâneo entre ciclos | determinístico | asserção |
| Classificar o ciclo em avançou / empatou / regrediu | determinístico | asserção |
| Decidir voltar | determinístico | asserção |
| Restaurar | determinístico | asserção |
| Dizer que voltou | determinístico | asserção |
| **O que o agente faz depois de voltar** | já existe (mediado) | contratos da `agent-loop` |

Nada aqui é mediado, e a RN-3 diz por que isso não é opcional.

## 3. User stories

**US-1.** Como operador, quero que um `/loop` longo deixe pontos de retorno, para
eu poder ver o que cada ciclo mudou e desfazer um ciclo sem desfazer o dia.

**US-2.** Como operador, quero que um ciclo que **piorou** seja desfeito em vez
de servir de base para o próximo, para o laço não empilhar dano.

**US-3.** Como agente, quero saber que fui revertido e por quê, para não repetir
a mesma tentativa achando que ela nunca aconteceu.

## 4. Regras de negócio

### RN-1 — Todo **ciclo** tem um ponto de retorno, e hoje só o turno tem

O instantâneo existe e é tirado antes de cada escrita, que é a metade certa. O
que falta é o recorte: `BeginTurn` zera por turno, e um turno pode conter muitos
ciclos de verificação.

Antes, nunca depois — e essa parte o produto já faz.

### RN-2 — O ponto de retorno continua sendo conteúdo, não commit

Já é assim, e a regra existe para que continue sendo. Nada é commitado,
indexado, nem faz branch andar.

### RN-3 — Voltar é decisão do laço ou da pessoa, nunca do modelo

**Já é verdade e não pode deixar de ser.** `Undo` não está no registro de
ferramentas; os três chamadores servem ao `/undo` que a pessoa digita.

Um agente que pode reverter o próprio trabalho pode reverter a **evidência** —
apagar o que ficou vermelho é a forma mais limpa de sair de um laço que só
termina quando o vermelho acaba. É a mesma razão pela qual `done_propose` não
existe num turno de trabalho, e pela qual o modelo não julga se terminou.

O que esta família acrescenta é um **segundo** decisor não-modelo: o laço,
quando a medição diz que regrediu.

### RN-4 — Só se volta de **regressão nomeada**, nunca de "não progrediu"

Empatar é comum e legítimo: um ciclo que leu, entendeu e não fechou critério
nenhum não estragou nada. Reverter esse ciclo joga fora trabalho bom.

O que justifica voltar é **regressão** — um critério que passava e passou a não
passar. Isso é um estado que o `Report` já sabe apurar e que hoje ninguém apura,
porque `Progressed` devolve um booleano onde cabem três respostas.

### RN-5 — Nada é restaurado em silêncio

Quem escreveu tem que saber que foi desfeito, e quem está olhando também. Uma
reversão silenciosa é a única coisa pior que não ter reversão: o agente repete a
tentativa achando que ela nunca aconteceu, e a pessoa vê arquivos mudarem
sozinhos.

### RN-6 — Instantâneo que não pôde ser tirado é **dito**, não presumido

Arquivo grande demais, permissão negada, disco cheio. O ciclo continua — recusar
o trabalho porque o seguro falhou seria o seguro segurando o trabalho como
refém — mas o laço não pode acreditar que tem um ponto de retorno que não tem.

É a mesma distinção que o `vcs` faz entre "não há repositório" e "não olhei", e
que a `done-qualifier` faz entre vermelho e quebrado.

## 5. O risco que fica, e o que não o cobre

**O instantâneo cobre o que as ferramentas escreveram, e o `bash` escreve fora
dele.**

`State.Written()` sabe de `write` e `edit`. Um comando de shell que gera código,
roda um formatador, ou apaga um diretório não passa por ali — e o ponto de
retorno não vai ter aquilo. Restaurar sobre uma árvore que mudou por fora pode
deixar um estado que nunca existiu: metade do ciclo desfeita, metade não.

Isto **não é contornável** sem vigiar o filesystem inteiro, que é uma máquina
que este produto não quer e que erraria em silêncio. O que dá para fazer é
**declarar a cobertura** — o ponto de retorno é sobre o que o agente escreveu
com as ferramentas — e não prometer mais do que isso.

**O segundo risco é oscilação.** Um laço que reverte, tenta de novo, reverte de
novo gasta ciclos sem sair do lugar, e cada volta apaga a evidência da tentativa
anterior. O teto de ciclos parados já limita isso por cima, mas mal: ele conta
ciclos, e uma oscilação é justamente uma sequência de ciclos que parecem
diferentes.

## 6. Fora de escopo

- **Commit, branch, stash ou qualquer escrita em git.** RN-2.
- **Desfazer o que um comando de shell fez.** §5, e é limite declarado.
- **Desfazer entre sessões.** O ponto de retorno vive enquanto a sessão vive.
  Recuperação depois do fim é o event log, e é outro assunto.
- **Desfazer edição feita por uma pessoa durante a sessão.** O arquivo é dela; o
  laço restaura o que o laço escreveu, e o que não reconhece ele não toca.
- **Escolher qual ciclo restaurar.** O laço volta **um** — o último. Uma máquina
  de escolher ponto no tempo é uma interface, e esta família é um seguro.

## 7. Changelog

- [202608301200 — o laço não sabia voltar](changelog/202608301200-o-laco-nao-sabia-voltar.md)
