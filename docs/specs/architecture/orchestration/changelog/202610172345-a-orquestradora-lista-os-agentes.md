# A orquestradora lista os agentes

**Data:** 2026-10-17
**Specs afetadas:** `202610132000-orchestration` (`.i`, passo 13), `202608081250-client-tui`
(`.p`, §2.2 e §10)

Primeira parte do passo 13 da `.i`, adiantada com o aval do dono para que a TUI e o desktop
(passo 14) andem em paralelo com o núcleo. A RN-9 pede a lista viva também na TUI, como texto;
agora ela existe, dobrada só do fio, sem daemon nenhum nos testes.

## O que entrou

- **A lista é uma dobra do fio.** `internal/tui/agents.go` dobra `child.started`, `child.state`,
  `child.completed`, `review.verdict`, `gate.measured`, `question.asked`, `question.answered`,
  `merge.completed`, `merge.refused` e `main.health` em `Model.Agents`. `child.*` traz o filho
  inteiro, então a linha muda sem reler `/children`. `Model.loadAgents` toma as respostas de
  `/children` e `/questions` como a lista.
- **A coluna lateral troca o diff pela lista** na sessão com `role: orchestrator`. A orquestradora
  roda em `plan` e nunca escreve (decisão 3): o diff dela ficaria vazio para sempre, e a coluna
  não abriria, porque abria só com arquivo tocado, janela ou aprovação. Agora abre também pelo
  papel. A metade de baixo continua a de sessão.
- **Tokens, e não dólares** (decisão 1): entrada, saída e cache, como um trio cuja ordem é dita
  uma vez sob o título — a mesma regra das barras do diff, cuja escala é dita.
- **Retenção é do daemon.** Fechado e merged saem quando a resposta do daemon não os traz mais;
  o cliente não tem relógio de retenção (`.p` §11). `failed` que o cliente já conhece fica mesmo
  quando a resposta o deixa de fora: falhou espera alguém e não sai sozinho.
- **Nada some em silêncio.** Evento que não se lê vira linha de erro no fluxo, com o tipo; evento
  de filho que a lista não conhece abre a linha dele, com o estado "ainda não dito"; estado que o
  cliente não conhece é desenhado como veio; filho que não cabe é contado.

## O que ficou de fora, e por quê

- `dcode orchestrate`, `^Q`, `^R` e a leitura de `/children` ao abrir. São o resto do passo 13 e
  dependem de rotas que o daemon ainda não responde (`POST /v1/projects`, `/children`,
  `/questions`); o cliente (`pkg/client/orchestrate.go`) já as tem. Sem tecla nova, as tabelas de
  teclas e comandos da `client-tui` não mudam.
- As constantes dos oito estados ficam na TUI, não no `protocol`: o passo 4 está em voo no núcleo
  e é dele dizer onde a máquina de estados mora. Quando o `protocol` ganhar as constantes, a TUI
  passa a usá-las.
- Nenhuma linha da `.p` §13 da `orchestration` é reivindicada aqui. "Fechado sai da lista depois
  da retenção" é invariante do daemon; o que este PR testa é o lado do cliente, e está na
  `client-tui`. O fechamento das invariantes previstas continua com o primeiro passo do núcleo que
  testar uma delas.
