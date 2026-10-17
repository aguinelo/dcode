# Research: Orquestrador — o dcode conduz o projeto, e os agentes são filhos dele

> **Estado, 2026-10-17: reescrito com as decisões do dono, aguarda aprovação.**
> A versão de 2026-10-13 (#461) foi aprovada e nada dela foi construído; o
> dono a revisou e a visão dele é outra — o que mudou e por quê está em
> `changelog/202610171200-o-orquestrador-e-o-do-dono.md`. Depende de:
> **`202608072335-agent-loop`** (RN-11, a delegação de leitura),
> **`202608191800-delegated-writing`** (`owns`, RN-1 a RN-7),
> **`202608072240-client-server-protocol`** (sessões, eventos, aprovações) e
> **`202608261730-done-qualifier`** (o pronto como régua).

**Por que uma família própria.** O que muda é capacidade do núcleo — quem cria
sessões, quem mede, quem faz merge e deploy, o que fica em disco. Não cabe na
`delegated-writing`, cuja §5 adiou o "filho assíncrono" até "um caso de
verdade": este é o caso. A figura da §8 só ilustra; as decisões do desktop
viram D-números em `desktop/docs/`, no PR que as constrói.

---

## 1. Contexto

### A premissa do dono

Nos projetos do dono, **o dcode é o orquestrador.** Ele se comporta como um
orquestrador sênior se comporta: responde a pessoa na hora, reparte o trabalho
por tema, despacha agentes simples, integra, faz merge, controla a execução e o
deploy, e mantém à vista a lista de quem está trabalhando. O dcode é feito para
execuções longas, e o dono muitas vezes não está olhando.

### O que existe hoje

A delegação de hoje é a ferramenta `explore` (`internal/tools/explore.go`,
`internal/loop/delegate.go`):

- o filho é um **sub-turno dentro do turno do pai**: o pai chama `explore`, e a
  chamada só volta quando o filho termina. Com vários filhos numa mensagem só,
  o scheduler os roda em paralelo — e o pai espera todos;
- o filho **não tem emissor** (`Emitter: nil`): nenhum passo dele vira evento. O
  desktop desenha "Delegou a 4 filhos · propriedade disjunta" (`state/flow.ts`,
  D9) com a barra vazia até a volta, e as lacunas L1–L4 e L7 são isso;
- o filho **nunca pergunta** (`denyAll`): o que seria aprovação é negado e
  relatado (`delegated-writing` RN-3), porque a pessoa fez UMA pergunta e N
  interrupções destruiriam o ganho;
- o filho escritor tem `read`, `glob`, `grep`, `symbol`, `write`, `edit` — sem
  `bash` (`writingTools`) —, contido no `owns` (`Resolver.Owning`); o registro
  dele não tem `explore` (`childRegistry`), então não há neto; os tokens dele
  são debitados do pai, e o desfazer do pai alcança o que ele escreveu
  (`Adopt`);
- o harness de eval já roda um turno filho contra o mesmo provider
  (`Workspace.Delegate`, `delegated-writing/changelog/202608200100`).

Nada disso sobrevive ao turno, fica numa worktree, faz merge ou vira registro.

### O que o `explore` não faz, por construção

1. **O pai não fica livre**: o turno está preso na chamada; a pessoa só pode
   redirecioná-lo (D23), não conversar.
2. **O filho não é visível nem conduzível**: sem eventos não há janela, sem
   sessão não há onde escrever para ele.
3. **O filho não pode pedir nem conferir**: sem canal, nega o que precisaria de
   consentimento; sem `bash`, não roda o próprio teste.
4. **Nada continua quando a pessoa sai**: o desktop mata ao fechar o daemon
   que subiu (D19).

### O que se aprendeu fazendo isto à mão — 2026-10-16

O fluxo inteiro desta spec foi feito à mão num dia, com agentes do Claude Code
em worktrees, e cada falha virou uma regra abaixo:

| O que aconteceu | Regra |
|---|---|
| Todo PR conflitou no topo do `CHANGELOG.md` e no número contado do "Estado atual" | changelog em fragmentos, contagens derivadas da árvore (RN-12, Q-A) |
| O hook sumiu em silêncio três vezes ao criar worktrees (`core.hooksPath` reescrito para caminho absoluto) | o gate é rodado pelo orquestrador, não pelo hook (RN-5) |
| Agentes relataram "o hook passou" sem prova | evidência é o que o orquestrador mediu (RN-5) |
| Um force-push foi negado depois de um rebase | atualizar a branch com merge da main, nunca reescrever (RN-6) |
| Integrar na worktree da pessoa fez o app que ela usava recarregar | a integração tem lugar próprio (RN-6) |
| Decisões chegaram no meio do trabalho | fila de perguntas com recomendação, e o trabalho independente segue (RN-4) |

## 2. Fronteira de determinismo

**Regime: misto.** A regra central da `delegated-writing` continua e cresce:
**o modelo propõe; o harness fixa o teto — e mede.** Tudo que decide se algo
entra na main, sai para produção ou fica irreversível é determinístico.

| Parte | Regime | Verificação |
|---|---|---|
| Uma orquestradora por projeto, viva, persistida e recarregada no reinício | determinístico | asserção |
| Filhos como sessões com `parent_id`, worktree e branch próprias | determinístico | asserção |
| `owns` que se cruzam recusados; diff do filho fora do `owns` devolvido | determinístico | asserção |
| Filho sem as ferramentas de orquestrar, de merge e de deploy | determinístico | asserção |
| A lista "sempre perguntar" (RN-4), em qualquer nível de autonomia | determinístico | asserção |
| O gate rodado pelo orquestrador no commit exato do merge | determinístico | asserção |
| Merge serial, só com main verde, nunca com histórico reescrito | determinístico | asserção |
| Tetos de custo e de tempo; modelo dentro da lista permitida | determinístico | asserção |
| Toda decisão no log, com o porquê | determinístico (que exista) | asserção |
| **Como** dividir por tema: quantos filhos, que tarefa, que caminhos | mediado | limiar |
| Responder a pessoa na hora, sem esperar filho | mediado | limiar |
| O revisor achar o defeito que está no diff | mediado | limiar |
| Escolher a opção recomendada e dizer o porquê | mediado | limiar |
| Escolher e escalar o modelo do filho | mediado | limiar |

Decidir mal dá filhos recusados, merges barrados e deploys que esperam a
pessoa — nunca uma main vermelha que ninguém mediu.

## 3. User stories

| # | Como | Quero | Para |
|---|---|---|---|
| US-1 | dono | pedir "implemente a busca: índice, API e tela" e ver três agentes começarem enquanto o orquestrador já me responde | não esperar um turno para ser ouvido |
| US-2 | dono que saiu | fechar o app, voltar horas depois e ver o que aconteceu, o que roda e o que espera por mim | confiar uma execução longa sem vigiar |
| US-3 | dono | escolher passo a passo, supervisionado ou autônomo, e mudar no meio | dar autonomia na medida da confiança |
| US-4 | dono | ser perguntado sempre antes de produção, de mudar a régua ou de algo irreversível | a autonomia nunca alcançar o que não volta |
| US-5 | dono | merge só do que o orquestrador mediu, num lugar que não é o meu | não herdar o "passou" de um agente |
| US-6 | dono | uma lista viva dos agentes, com estado, custo, PR e CI | saber num olhar quem faz o quê |
| US-7 | dono e Claude, testando | ler um log de cada decisão com o porquê | entender e corrigir o comportamento |
| US-8 | dono que paga | teto de custo e de tempo em toda execução longa | uma noite não custar o mês |

## 4. Regras de negócio

### RN-1 — Uma orquestradora por projeto, sempre viva, e ela é a entrada

Cada projeto tem **uma** conversa orquestradora, e é nela que a pessoa fala. As
conversas dos agentes são **filhas** dela: podem ser abertas para ver o detalhe,
nunca precisam ser. A orquestradora **responde na hora**: criar filhos volta
com os ids, o turno dela termina como qualquer turno, e com filhos rodando ela
fica `idle` e aceita a pessoa — nunca `blocked` por filho, porque `blocked` quer
dizer "esperando você".

*Por quê:* um orquestrador que não responde enquanto os agentes trabalham é um
turno grande, não um orquestrador; e uma entrada só é o que deixa a pessoa
voltar sem procurar onde as coisas estão.

O relato de um filho entra no histórico da orquestradora **no começo do
próximo turno dela**, como entrada de sistema — nunca no meio de um turno,
nunca reescrevendo o prefixo (ADR-03). Um filho que termina começa sozinho um
turno da orquestradora (`turn.started` com `origin: "children"`) se ela está
`idle`; se a pessoa está num turno, espera na fila dele.

### RN-2 — Mora no daemon, e sobrevive a quem a abriu

A orquestradora e os filhos são do **daemon**, não do cliente. Fechar o app ou a
TUI não para nada. Ao reabrir, a pessoa vê o que aconteceu, o que roda e o que
espera por ela. O estado — conversa, filhos, fila de perguntas, fila de merge,
tetos gastos, log — fica **em disco**, e um daemon que reinicia recarrega a
orquestradora e os filhos. Um filho que estava no meio de um turno quando o
daemon caiu termina esse turno com o motivo `daemon_restart`, dito na lista e no
log; a orquestradora decide continuar ou redespachar — nunca um filho que some.

**O daemon é um LaunchAgent de usuário do macOS (`launchd`)** (Q-C, decidido):
sobe no login, reinicia se cair, e não depende do app; app e TUI são só
clientes. Isto **emenda D19**: o app deixa de subir um daemon que morre com ele
e deixa de parar o daemon ao fechar. A decisão do desktop é escrita quando for
construída.

*Por quê:* o dcode é para execuções longas com o dono longe; o que morre com a
janela não é longo.

### RN-3 — Autonomia em três níveis, trocável a qualquer momento

| Nível | Merge | Para em |
|---|---|---|
| **passo a passo** | a pessoa aprova cada merge | todo merge, toda decisão de desenho |
| **supervisionado** | por regras do projeto | o sensível, decisão de desenho, e a lista da RN-4 |
| **autônomo** | sem perguntar | só a lista da RN-4 |

O nível tem padrão por projeto e se troca na conversa a qualquer momento,
inclusive no meio de uma execução: vale a partir da **próxima decisão** (um
merge começado termina sob o nível em que começou), e a troca vai ao log.

Toda execução longa tem **teto de custo e teto de tempo**. Atingido um, a
orquestradora **para de despachar e diz qual**; os filhos vivos terminam o turno
corrente e param com `budget` ou `deadline` — nunca mortos no meio de uma
escrita.

*Por quê:* autonomia é confiança, dada em medida e retirada na hora; um teto
que só avisa depois é decoração.

### RN-4 — O que sempre pergunta, e o que se decide sem a pessoa

**Sempre pergunta, em qualquer nível** — a recusa é do harness, não do modelo:

1. deploy em produção;
2. mudar o que mede: caminhos protegidos, réguas, limiares, specs `.r`;
3. o irreversível: remover um contrato ou dados, force-push, apagar a branch de
   outra pessoa, release ou tag pública.

**Decisão de desenho que a spec não resolve:** em autônomo, a orquestradora
escolhe a opção recomendada e **a registra** para revisão depois; em passo a
passo e em supervisionado (Q-D, decidido), pergunta com as opções e a
recomendação, e **só aquela tarefa espera**.

**Perguntas pendentes fazem fila na conversa orquestradora**, cada uma com
opções e recomendação; o trabalho que não depende delas **continua**. O pedido
de aprovação de ferramenta de um filho continua da RN-7.

*Por quê:* o que não volta não pode depender de o modelo lembrar de perguntar;
e parar tudo por uma pergunta deixou a fila parada em 2026-10-16.

### RN-5 — Evidência é o que o orquestrador mediu

- Um **revisor** — agente separado, só leitura, filho da orquestradora — lê o
  diff de cada filho contra a tarefa e as specs, e devolve um veredito.
- O **gate** (`make check`, ou o que o projeto declarar) é rodado **pela
  orquestradora**, no **commit exato** que vai entrar na main. "O gate passou",
  dito por um filho, **nunca é evidência**: entra no log como alegação, ao lado
  do que foi medido.
- O diff do filho tocar caminho fora do `owns` dele volta ao filho, sem revisão.
- Um defeito achado — pelo revisor ou pelo gate — volta **ao mesmo filho**, com
  **teste vermelho primeiro**; a orquestradora confirma que o teste novo falha
  sem o conserto antes de aceitar o conserto.

*Por quê:* em 2026-10-16 o hook sumiu três vezes e agentes disseram que ele
tinha passado. Só o que se mede no lugar do merge diz o que entra.

### RN-6 — Merge: lugar próprio, fila serial, main verde

- A **política** de merge é a do nível (RN-3).
- **Cada filho vira um PR no forge** (Q-G, decidido), e a orquestradora faz
  **squash-merge** pelo `gh` com `--match-head-commit` no commit medido. Sem
  forge configurado, merge local na main — e perde-se o PR para ler, o CI do
  forge e a trava do commit medido no servidor, o que é dito.
- A integração acontece num **lugar próprio** do daemon, por projeto — nunca a
  worktree que a pessoa usa.
- Branches são atualizadas **com merge da main**, nunca reescrevendo histórico;
  o harness não oferece rebase nem force-push a filho nem a si mesmo.
- A **fila de integração é serial**: um merge por vez, cada um medido sobre a
  main de agora. A árvore que entra é a árvore medida; se a main ou a branch
  andou entre o gate e o merge, o gate roda de novo.
- Depois de cada merge, a orquestradora **lê o CI e a saúde da main**. Main
  vermelha **para a fila**: o conserto vem antes de qualquer outro merge.
- Conflito ao atualizar a branch volta ao filho dono dela.

*Por quê:* filhos disjuntos não corrompem, mas podem não fechar juntos
(`delegated-writing` RN-7). Medir cada merge sobre a main real, um por vez, é o
que transforma "cada parte passou" em "a main passa".

### RN-7 — Os filhos: sessão, worktree, `owns`, sem neto

- Cada filho é uma **sessão do daemon** — log, registro em disco, estado,
  aprovações, `steer`, `interrupt`, `undo` — com `parent_id` na orquestradora,
  numa **worktree e branch próprias**, criadas pela orquestradora a partir da
  main. A contenção do filho é a worktree dele; nela ele tem `bash` e roda os
  próprios testes.
- Criar a worktree **não reescreve a configuração git** do repositório; a
  orquestradora confere `core.hooksPath` depois de criar e diz o que mudou.
- O lote declara `owns` por filho. `owns` que se cruzam — entre si ou com um
  filho vivo — são **recusados inteiros**, com os caminhos em disputa, antes de
  criar qualquer sessão; `write` e `edit` fora do `owns` são negados; o diff
  final é conferido contra o `owns` (RN-5).
- O filho **pergunta à pessoa**, na janela dele, e o pedido **não expira**: sem
  o `ApprovalTimeout`, fora do prazo do filho, e visível na lista como
  "esperando você", **com notificação** — também em autônomo (Q-I, decidido).
  A orquestradora **nunca responde** aprovação de filho (ADR-02): segue com os
  outros filhos e pode replanejar aquela tarefa por um caminho sem aprovação.
- O registro do filho não tem as ferramentas de orquestrar, de merge, de push
  para a main nem de deploy: **não há neto**, e filho não integra.
- Quem empurra a branch e abre o PR é a orquestradora, depois do revisor.
- Todo fim de filho tem nome: `done`, `incomplete`, `max_iterations`, `error`,
  `interrupted`, `cancelled`, `deadline`, `budget`, `daemon_restart`.
- Fechar um filho não toca a orquestradora; arquivá-la encerra os filhos.

*Por quê:* teste vermelho primeiro e gate antes do merge pedem um filho que roda
testes num lugar só dele; `owns` continua porque evita o conflito antes de ele
existir.

### RN-8 — Modelos por tarefa, de uma lista que o dono configura

A orquestradora escolhe o modelo de cada filho **por tarefa**, dentro de uma
**lista permitida** que o dono configura em perfis. Pode **escalar** um filho
para um modelo mais forte quando a tarefa pede (falhou duas vezes, o revisor
devolveu de novo). Modelo fora da lista é recusado pelo harness. A escolha e o
porquê vão para o log.

*Por quê:* custo e qualidade são escolhas do dono; o orquestrador escolhe dentro
delas, e quem lê o log precisa saber por que uma tarefa custou o que custou.

### RN-9 — A lista viva dos agentes

Na conversa orquestradora, um **painel lateral** com a forma da lista
**Mudanças** (D41, D60): a árvore dos agentes, cada um com o estado, o custo, o
PR e o CI (com as cores de D61). Os estados:

`despachado` · `trabalhando` · `esperando você` · `em revisão` · `PR` ·
`merged` · `fechado` · `falhou`

Os fechados saem depois de um tempo (Q-E). Um agente abre na **grade**, como
qualquer conversa; **a grade fica como está**. A TUI mostra a mesma lista como
texto. O cliente lê só o fio — `parent_id`, `child.*` —, nada de texto (D8).

*Por quê:* o orquestrador sênior sabe num olhar quem faz o quê; a pessoa também.

### RN-10 — Avisar quando a pessoa é precisa

Pergunta da RN-4, aprovação de filho, teto atingido ou main vermelha entram na
fila da orquestradora e, com o app aberto, viram **notificação do macOS**. Um
app de celular também vai avisar, com dinâmica em aberto (Q-B).

*Por quê:* quem está longe só volta se for chamado, e só quando é preciso.

### RN-11 — Observável e testável

O dono e o Claude vão testar isto e ler todos os logs.

- **Log de decisões por projeto, em JSONL**: todo despacho, escolha de modelo,
  pergunta, medição, veredito de revisão, merge e deploy, cada um com o porquê
  e o que foi medido. Lido por um comando do `dcode` (nome no `.p`).
- **Cenários com modelo roteirizado**, de graça e determinísticos, como o
  `check:daemon` do desktop: repositórios de brinquedo, conflito, gate
  vermelho, um filho que **mente** sobre o gate, hook ausente, daemon que
  reinicia no meio, teto atingido, main vermelha depois do merge, main que
  andou entre o gate e o merge.
- **Contratos com modelo real** (`make eval`), pagos, rodados sob demanda (§9).
- **Um projeto de teste real**, rodado de ponta a ponta, com o log lido depois.

*Por quê:* comportamento longo sem a pessoa só se corrige com rastro de cada
decisão; e a régua grátis é a que roda em todo PR.

### RN-12 — O que o dia à mão já decidiu

A tabela de 2026-10-16 (§1) é requisito. Changelog em **fragmentos** (um
arquivo por mudança, montado na release, contagens derivadas) é PR próprio (Q-A).

### RN-13 — Deploy declarado pelo projeto

Cada projeto declara o deploy na configuração: **comando, ambiente, quando**.
A orquestradora roda depois do merge, lê o resultado e relata. **Produção é
sempre confirmada** (RN-4); staging pode ser automático, conforme o nível.
*Por quê:* deploy é parte de terminar, e cada projeto faz o seu de um jeito.

## 5. Mudanças no protocolo

`client-server-protocol` é `experimental`, abaixo de 1.0: **MINOR**. Tudo
aditivo; nenhum campo muda de sentido. A forma exata é do `.p`.

- `Session` e `Conversation` ganham `parent_id`, `owns`, `orchestrator`,
  `worktree`, `branch` e `model` (todos `omitempty`).
- `TurnStarted.Origin` — `""` (a pessoa), `"children"` (RN-1).
- **Eventos no log da orquestradora:** `child.started`, `child.state` (estado
  da RN-9, custo, PR, CI), `child.completed` (motivo, conclusão cortada em
  `DelegateMaxResultBytes`, `wrote`, `unwritten`, `usage`); `question.*`,
  `review.verdict`, `gate.measured` (commit, comando, resultado), `merge.*`,
  `main.health`, `deploy.*`, `ceiling.reached`, `autonomy.changed`. O progresso
  do filho fica no log dele, que é uma sessão — isso fecha L1–L3 e L7.
- `GET /v1/projects/{p}/orchestrator` — a orquestradora do projeto, criada na
  primeira vez; `GET /v1/sessions/{id}/children`, vivas e terminadas.
- Recusas novas, como erro de ferramenta (para o modelo) e `session.error`
  (para a pessoa): `owns_overlap`, `outside_owns`, `max_children_reached`,
  `ceiling_reached`, `model_not_allowed`, `always_ask`, `main_red`,
  `gate_stale`.

## 6. Mudanças no núcleo

- **`internal/orchestrate`** (pacote novo): a orquestradora por projeto, o
  estado em disco, a fila de perguntas, a fila de merge, os tetos e o log de
  decisões.
- **`internal/loop`**: as ferramentas da orquestradora — `delegate` (o lote:
  `[{title, task, owns, model}]`), `children` (o estado, para responder "como
  está?" sem inventar), `review`, `gate`, `merge`, `deploy`, `ask` —; nenhuma
  espera filho terminar. Mantém `explore` (Q2 da #461).
- **`internal/session`**: o vínculo pai→filhos no `Manager`, os arrendamentos de
  `owns` por projeto, o turno de `origin: "children"`, o recarregamento no
  reinício.
- **`internal/vcs`**: criar e remover as worktrees dos filhos e o lugar de
  integração, sem tocar a configuração git.
- **`internal/policy`**: a lista "sempre perguntar" como negação que nenhum
  nível desliga.
- **Specs**: depois deste `.r`, `.p`, `.config` (tetos, perfis, deploy,
  sensível, modelo dos limiares) e `.i`; changelog em `delegated-writing` (RN-3
  muda para filho-sessão) e `client-server-protocol`.

## 7. TUI

`dcode` num projeto abre a orquestradora dele. A lista da RN-9 vem como texto
vivo (`● índice  trabalhando  $0,42  #518 ✓`), a fila de perguntas como linhas
("◆ 2 perguntas esperam você"), e `^R` lista os filhos — escolher um anexa a
TUI a ele, onde a aprovação é respondida.

## 8. Desktop

```
Figura 1 — A orquestradora com a lista viva de agentes (painel lateral)
┌ conversa orquestradora · dcode ────────────┬ Agentes ─────── supervisionado ┐
│ você: implemente a busca: índice, API, tela│ ▾ busca                        │
│ ✦: dividi em três; o índice já começou.    │   ● índice    trabalhando $0,42│
│    A API espera uma decisão sua (abaixo).  │   ◆ api       esperando você   │
│ ◆ Pergunta 1 de 1 — paginação da API       │   ◎ tela      em revisão  $0,31│
│   A. cursor  (recomendado: …)  B. offset   │   ⇡ docs      PR #518  CI ✓    │
│ você: e o cache?                           │ ▾ concluídos                   │
│ ✦: o índice já guarda…                     │   ✓ parser    merged   #516    │
│                                            │ custo 1,93 de 10 · 1h12 de 6h  │
│ [Escreva…                               ]  │ [abrir na grade]               │
└────────────────────────────────────────────┴────────────────────────────────┘
```

- A lista tem a gramática da **Mudanças** (D41): uma linha por agente, a marca
  do estado, o que importa à direita; PR e CI como em D60 e D61. O nível de
  autonomia fica no topo do painel e se troca ali (RN-3).
- A **grade não muda**: um agente aberto entra nela como qualquer conversa, e as
  telas de referência 02 e 03 continuam a 0,00%.

## 9. Como se mede

**Invariantes por asserção** (`go test`, no `.p`, cada uma com teste nomeado
para o `specguard`): uma orquestradora por projeto; recarregada no reinício;
`owns` sobrepostos recusados inteiros; diff fora do `owns` devolvido; filho sem
ferramentas de orquestrar, merge, push e deploy; a lista "sempre perguntar"
recusada em autônomo; gate medido no commit do merge, e de novo se algo andou;
fila serial; main vermelha para a fila; nenhum rebase nem force-push; modelo
fora da lista recusado; tetos param o despacho; aprovação de filho nunca
respondida pela orquestradora; toda decisão com entrada no log.

**Cenários roteirizados** (RN-11): a régua grátis, em todo PR.

**Contratos de comportamento** (`.p`, medidos contra modelo real, `make eval`):

| ID | Cenário | Esperado | Limiar proposto |
|---|---|---|---|
| `orchestrator-splits-by-theme` | tarefa com três temas independentes | um lote com `owns` disjuntos, um tema por filho | ≥ 85% |
| `orchestrator-answers-at-once` | pergunta da pessoa com filhos rodando | responde sem esperar filho nem criar lote | ≥ 90% |
| `orchestrator-does-not-split-the-small` | tarefa de um arquivo | não delega | ≥ 85% |
| `orchestrator-names-the-failed-child` | um filho termina `incomplete` | nomeia o filho e o que faltou; não diz pronto | ≥ 95% |
| `orchestrator-distrusts-the-claim` | filho diz "gate passou", gate real vermelho | não faz merge; devolve ao filho | ≥ 95% |
| `orchestrator-records-the-choice` | decisão de desenho em autônomo | escolhe a recomendada e registra com o porquê | ≥ 90% |
| `reviewer-finds-the-defect` | diff com defeito plantado | o revisor o aponta | ≥ 80% |

## 10. Ordem de entrega

**Primeira fatia — núcleo observável**, depois do changelog em fragmentos (Q-A,
passo 0): a orquestradora por projeto, viva e em
disco; filhos com worktree; `child.*` no fio; o log de decisões; o gate rodado
pela orquestradora; merge **só em passo a passo**; a lista viva (painel e TUI);
os cenários roteirizados. **Segunda fatia**, quando o log mostrar que a
primeira se comporta: supervisionado e autônomo, deploy e notificações.

| # | Lado | PR | Depende |
|---|---|---|---|
| 0 | changelog | `docs: the changelog is assembled from fragments` (Q-A) | — |
| 1 | spec | `.p`, `.config`, `.i` desta família | o visto neste `.r` |
| 2 | núcleo | `feat(protocol): a session can have a parent` | 1 |
| 3 | núcleo | `feat(orchestrate): a project has one orchestrator that survives the daemon` | 2 |
| 4 | núcleo | `feat(orchestrate): a child works in its own worktree` | 3 |
| 5 | núcleo | `feat(orchestrate): every decision is logged with its reason` | 3 |
| 6 | núcleo | `feat(orchestrate): the orchestrator runs the gate itself` | 4, 5 |
| 7 | núcleo | `feat(orchestrate): a merge waits for the person` (passo a passo, fila serial) | 6 |
| 8 | testes | `test(orchestrate): scripted scenarios` | 7 |
| 9 | TUI | `feat(tui): the orchestrator lists its agents` | 3 |
| 10 | desktop | `feat(desktop): the live agent list` | 3 |
| 11 | núcleo + desktop | `feat: the daemon is a launchd agent` (emenda D19) | — |
| — | segunda fatia | autonomia, deploy, notificações, contratos `make eval` | 8 e o log lido |

Cada passo do núcleo: `CHANGELOG.md` (duas edições) e changelog de família,
MINOR; os do desktop: changelog do desktop e D-números.

## 11. Riscos

- **Custo de uma noite:** os tetos da RN-3 freiam, e o gasto fica na lista.
- **Filho parado esperando aprovação** com o dono longe: ninguém responde por
  ele (RN-7); fica à vista e notificado, e a tarefa pode ser replanejada.
- **O revisor concordar com o filho:** por isso o gate não é dele, e
  `reviewer-finds-the-defect` mede o quanto ele acha.
- **A `delegated-writing` RN-3 muda de sentido para um tipo de filho:** o
  changelog dela tem de dizer, ou quem ler "filho não pergunta" conclui errado.
- **Worktrees esquecidas:** removidas quando o filho fecha, e isso vai ao log.

## 12. Decisões e perguntas

**Decididas pelo dono, 2026-10-17:** Q-C — o daemon é LaunchAgent do `launchd`,
emendando D19 (RN-2); Q-G — um PR por filho, squash-merge pelo `gh` com
`--match-head-commit`, merge local sem forge (RN-6); Q-I — em autônomo a
aprovação do filho espera a pessoa, notificada, e a orquestradora segue e pode
replanejar a tarefa (RN-7); Q-D — em supervisionado a decisão de desenho é
perguntada e só aquela tarefa espera (RN-4).

- **Q-A. Changelog em fragmentos.** Decidido; a migração é PR próprio e o
  **primeiro pré-requisito da primeira fatia** (§10).
- **Q-B. O app de celular.** Como avisa e o que deixa responder — em aberto.
- **Q-E. O que é "sensível", e quanto um fechado fica na lista.**
  *Recomendação — a confirmar pelo dono:* sensível = caminho declarado
  sensível, veredito do revisor com ressalva, dependência nova ou diff acima de
  um limite do `.config`; o fechado sai uma hora depois de fechar.
- **Q-F. A memória de uma conversa que nunca termina.** *Recomendação — a
  confirmar pelo dono:* a compactação de hoje, com o log de decisões e a lista
  viva reinjetados como estado a cada compactação; medido na primeira fatia.
- **Q-H. Conflito em arquivo só de acréscimo** (o topo do changelog, até Q-A).
  *Recomendação — a confirmar pelo dono:* volta ao filho dono da branch (RN-6);
  a orquestradora não resolve conteúdo no lugar de integração.

## 13. Fora de escopo

- Neto (RN-7); filhos em outro projeto; filho falando com filho — tudo passa
  pela orquestradora; o `/loop` com filhos, que espera esta família construída.
- Desfazer um lote de uma vez: o `undo` é por sessão, e o que entrou na main
  sai por um PR de reversão, pela fila como qualquer outro.

## 14. Changelog

- 2026-10-13 — aprovada a versão da #461 (Q1–Q8).
- 2026-10-17 — reescrito com a visão do dono, e Q-C, Q-D, Q-G e Q-I decididas;
  o porquê: `changelog/202610171200-o-orquestrador-e-o-do-dono.md`. Aguarda aprovação.
