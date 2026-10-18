# A pessoa decide na conversa da orquestradora

**Data:** 2026-10-18
**Specs afetadas:** `202610132000-orchestration` (`.r` RN-4; `.p` §5.1, §5.2, §5.3, §5.5, §9.1, §10,
§13); `behavior-definition` `.p` §2.1, pela seção do papel da orquestradora.

## O que aconteceu

No teste ao vivo de 2026-10-18 três filhos ficaram prontos juntos, e a pessoa respondeu os merges um
por um, agente por agente, numa fila separada da conversa. A pergunta de merge dizia só "Merge X,
medido verde em abc?" — sem o que mudou, e com a evidência só no corpo do PR. O dono decidiu, no
mesmo dia, que a decisão mora na conversa com a orquestradora, num cartão como o de pergunta do
Claude Code.

## As quatro decisões do dono

1. **Um cartão.** Cada item tem uma etiqueta curta, o texto, as opções com rótulo e descrição, a
   recomendada marcada, e uma resposta em texto livre ("Outro"); vários itens num cartão só.
2. **Lote.** Perguntas do mesmo tipo pendentes juntas — três merges prontos — são um cartão, cada
   uma com a recomendação já selecionada; a pessoa ajusta e confirma tudo de uma vez, ou aplica a
   recomendação a todas.
3. **Texto livre vira proposta.** Se a pessoa escreve "merge em todas" ou "devolve o listas pedindo
   mais testes", a orquestradora monta um cartão **pré-preenchido** que a pessoa confirma com um
   clique. A orquestradora nunca aplica resposta.
4. **Cada merge mostra** o que mudou (2–3 linhas da orquestradora, na língua da pessoa) e a
   evidência medida: o veredito do revisor e o resultado, o comando e o commit do gate, do log de
   decisões — o que o harness mediu, nunca o que o filho disse.

Depois o dono mostrou a referência exata: um item por vez, "1 de N" com setas, a pergunta como
título, as opções como rádio com descrição, "(Recomendado)", "Outro", "Pular" e "Próximo ⌘↵", que no
último confirma todas, e o selo "Precisa da sua contribuição". A `.p` §5.5 descreve isso para o
desktop e a TUI, que desenham o cartão no PR seguinte; este traz o núcleo e o fio.

## O que mudou

- **Protocolo.** `Question` ganha `header`, `allow_other`, `summary` e `evidence`
  (`QuestionEvidence`, lida das linhas do log com o `seq` delas); `QuestionOption` ganha
  `description`. Tipos novos: `QuestionCard`, `CardItem` (a `Question` plana com `state`,
  `selected`, `selected_note`, `other`), `AnswerQuestionsRequest`/`Response`, `AnswerItem`,
  `AnswerResult`; evento `question.card`; rota `POST /v1/sessions/{id}/questions/answers`; e `GET
  /questions` traz `cards`. Tudo aditivo: a TUI (`^Q`) e a fila do desktop continuam lendo a mesma
  `Question`.
- **Núcleo.** `cards.go` agrupa: a pergunta entra no cartão aberto do tipo dela, ou num novo; o
  cartão fecha quando todo item tem resposta. `answers.go` responde o lote item por item, a primeira
  resposta levando dentro do lote também, e grava o pulo (`question.skipped`). `decide.go` tem
  `Summarize` e `ProposeAnswers`. `evidence.go` lê a evidência do log.
- **Ferramentas.** `summarize` e `propose_answers`; `ask` aceita `header` e `description`. A seção
  do papel da orquestradora diz quando usar cada uma, e que ela nunca responde pela pessoa.
- **Log.** Quatro `kind` novos: `question.summarized`, `question.skipped`, `question.relayed`,
  `answers.proposed`.

## O que o código decide onde o dono deixou a forma em aberto

- **O agrupamento é do harness**, não de uma ferramenta: determinístico, segue `question.asked`, e
  não precisa de linha própria. A orquestradora não compõe cartão; só resume e propõe.
- **"Outro" tem dois sentidos, e a regra é quem age.** Numa pergunta de desenho, a resposta só vai
  à orquestradora, então as palavras são a resposta (`answer: "other"`). Num merge ou no teto o
  harness executa a opção, e palavras não se executam: vão à orquestradora (`question.relayed`) e a
  pergunta continua esperando, até a pessoa confirmar o cartão que ela propuser. É a decisão 3
  aplicada ao próprio cartão.
- **A evidência é lida do log, não do estado.** O `Child.Gate` vem da mesma linha, mas a decisão 4
  diz "do log de decisões": a pergunta carrega o `seq` de `review.verdict` e de `gate.measured`, e
  um log sem as duas não pergunta (o filho para, dito). O `claim` do filho mora na linha do gate e
  não entra na evidência.
- **A proposta escolhe só entre as opções.** "Outro" é da pessoa; uma proposta com ele é recusada.
- **Pular exige o cartão**: pular é "neste cartão, agora não", e o item pulado continua pendente.
- **"Aplicar a recomendação a todos" não tem rota própria**: é o mesmo lote, com a `recommended` de
  cada item.

## Por que nenhuma ferramenta responde

O que garante "a orquestradora nunca responde" não é o prompt: é que nenhum método de `Conductor`
responde pergunta, e `TestNoToolOfTheOrchestratorReachesAnAnswer` lê a interface por reflexão. A
proposta é um cartão; `TestProposingAnswersMakesAProposedCardAndAnswersNothing` confere que nem
`question.answered` nem `merge.queued` saem dela.
