# Research: Orquestrador — uma conversa que reparte o trabalho em sessões filhas

> **Estado, 2026-10-13: aprovado pelo dono**, com as decisões da §12. Nada aqui
> foi construído; vem depois do `/loop` no desktop. Depende de:
> **`202608072335-agent-loop`** (RN-11, a delegação de leitura),
> **`202608191800-delegated-writing`** (`owns`, RN-1 a RN-7),
> **`202608072240-client-server-protocol`** (sessões, eventos, aprovações) e
> **`202608261730-done-qualifier`** (o pronto como régua).

**Por que uma família nova, e por que este `.r` traz telas.** O que muda é
capacidade do núcleo — quem cria sessões, quem responde aprovações, de quem é um
caminho, como um relato volta —, e capacidade do núcleo tem spec
(`docs/conventions/SDD-HARNESS.md`). Não cabe na `delegated-writing`: aquela é
o filho como sub-turno síncrono, e a §5 dela adiou de propósito o "filho
assíncrono" até "um caso de verdade". Este é o caso. As figuras da §8 ilustram
as histórias de uso; as decisões de interface do desktop continuam virando
D-números em `desktop/docs/`, como `desktop/AGENTS.md` manda, no PR que as
constrói.

---

## 1. Contexto

### O que existe

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
  `bash` —, contido no `owns`; os tokens dele são debitados do pai, e o desfazer
  do pai alcança o que ele escreveu (`Adopt`).

### O que o dono pediu

Uma conversa **orquestradora** que divide o trabalho em filhos **paralelos**,
cada um numa janela da grade, enquanto ela continua **livre para conversar**
com a pessoa.

Três coisas que o `explore` não faz, por construção:

1. **O pai não fica livre.** O turno do pai está preso na chamada; a pessoa só
   pode redirecionar esse turno (D23), não conversar.
2. **O filho não é visível nem conduzível.** Sem eventos, não há janela que
   mostrar, e sem sessão não há onde escrever para ele.
3. **O filho não pode pedir.** Uma escrita que precisaria de consentimento não é
   delegável — e é exatamente o tipo de trabalho que se quer repartir.

O motivo de o `explore` negar em vez de perguntar era não ter com quem falar sem
interromper a única conversa. Com uma janela por filho, há.

## 2. Fronteira de determinismo

**Regime: misto.**

| Parte | Regime | Verificação |
|---|---|---|
| Criar as sessões filhas, com `parent_id` e `owns` | determinístico | asserção |
| Recusar `owns` que se cruzam, e escrita do pai em caminho arrendado | determinístico | asserção |
| Filho sem a ferramenta de delegar (sem aninhamento) | determinístico | asserção |
| Teto de filhos, prazo, débito de tokens | determinístico | asserção |
| O relato do filho chegar ao pai, com o motivo do fim | determinístico | asserção |
| O pai aceitar mensagem da pessoa com filhos rodando | determinístico | asserção |
| Fechar o pai encerrar os filhos | determinístico | asserção |
| **Como** dividir: quantos filhos, que tarefa, que caminhos | mediado | limiar |
| **Integrar**: conferir a árvore inteira antes de dizer pronto | mediado | limiar |
| Dizer qual filho falhou, em vez de concluir por cima | mediado | limiar |

A regra central da `delegated-writing` continua: **o modelo propõe a forma; o
harness fixa o teto.** Um pai que divide mal pede filhos que o daemon recusa,
nunca filhos que colidem.

## 3. User stories

| # | Como | Quero | Para |
|---|---|---|---|
| US-1 | pessoa | pedir "implemente a busca: índice, API e tela" e ver três filhos começarem, cada um numa janela | acompanhar trabalho paralelo sem montá-lo à mão |
| US-2 | pessoa | continuar conversando com o orquestrador enquanto os filhos trabalham | perguntar, ajustar o plano, pedir outra coisa |
| US-3 | pessoa | responder a aprovação de um filho na janela dele | consentir sabendo o que aquele filho está fazendo |
| US-4 | pessoa | escrever para um filho, interrompê-lo ou fechá-lo | corrigir uma parte sem parar as outras |
| US-5 | pessoa | ver o orquestrador juntar os relatos e conferir a árvore inteira | não aceitar "pronto" de partes que não fecham juntas |
| US-6 | pessoa que paga | um teto de filhos e de tokens que nada atravessa | dez filhos não custarem dez vezes sem aviso |

## 4. Regras de negócio

### RN-1 — O filho é uma sessão do daemon, não um sub-turno

Cada filho é uma sessão como qualquer outra — log de eventos, registro em
disco, estado, aprovações, `steer`, `interrupt`, `undo` — com `parent_id`
apontando para o orquestrador. É o que a faz aparecer na lista (N3), ganhar
janela na grade (D30, D31) e ser lida depois de terminar (D42) sem nada novo
no cliente para isso.

O `explore` continua como está: delegação **síncrona** de leitura e de escrita
pequena, dentro do turno. As duas convivem; a pergunta Q2 decide se o
orquestrador também tem `explore`.

### RN-2 — O orquestrador não espera

A ferramenta que cria filhos volta **na hora**, com os ids, e o turno do
orquestrador termina como qualquer turno. Com filhos rodando, ele fica `idle` e
aceita turnos da pessoa — nunca `blocked` por causa de filho, porque `blocked`
quer dizer "esperando você".

### RN-3 — O filho pergunta à pessoa, na janela dele

Aprovação não se herda (ADR-02) e **não sobe ao pai**: o consentimento é sobre
o ato daquele filho, e quem tem o contexto para dar é quem vê a janela dele.
O pedido do filho **não expira** (Q5): espera a pessoa, sem o `ApprovalTimeout`
de 120 s, e a grade o põe em "esperando você" (D30). O prazo existe para um
turno não ficar preso, e aqui o pai não está; esse tempo não conta no prazo do
filho (RN-8). Uma negação entra no relato como `unwritten`/`unread`.

Isto **muda** a `delegated-writing` RN-3 só para filhos-sessão: lá o filho não
tinha canal, e negar era a única saída honesta. Aqui ele tem. O `explore`
continua negando.

`allow_session` dado a um filho vale para aquele filho. `allow_project` e
`allow_always` valem como sempre valeram — são do projeto e do usuário, não da
sessão.

### RN-4 — Caminho possuído é arrendado, entre sessões

`owns` deixa de ser fronteira de um sub-turno e vira **arrendamento** no
workspace, guardado pelo daemon enquanto o filho vive:

1. a contenção do filho é o workspace reduzido ao `owns` (`Resolver.Owning`, o
   que já existe);
2. um lote cujos `owns` se cruzam — entre si ou com um filho vivo — é
   **recusado inteiro**, com os caminhos em disputa, antes de criar qualquer
   sessão. Serializar, como o scheduler faz com chamadas, deixaria um filho
   parado sem dizer por quê;
3. enquanto o arrendamento vale, a escrita do **orquestrador** num caminho
   arrendado é negada com o nome do filho que o possui;
4. filho sem `owns` só lê;
5. o arrendamento acaba quando o filho termina, é fechado ou o pai é fechado.

### RN-5 — Sem aninhamento

O registro de ferramentas do filho não contém as ferramentas de orquestrar,
pela mesma construção do `childRegistry`. Um filho não cria neto.

### RN-6 — O relato volta como fato, e a integração é um turno do pai

Quando um filho termina, o log do pai recebe `child.completed`, com o motivo do
fim, o que escreveu, o que não pôde, o selo e a conclusão (cortada no
`DelegateMaxResultBytes`). O relato entra no histórico do pai **no começo do
próximo turno dele**, como entrada de sistema — nunca no meio de um turno,
nunca reescrevendo o prefixo (ADR-03).

Quando **o lote inteiro** terminou e o pai está `idle`, o daemon começa **um**
turno de integração do pai (`turn.started` com `origin: "children"`). Se a
pessoa está com um turno do pai rodando, a integração espera na fila desse
turno, como uma mensagem digitada esperaria. Uma vez por lote.

### RN-7 — Pronto é do pai, sobre a árvore inteira

`delegated-writing` RN-7 continua valendo, agora com mais razão: filhos
disjuntos não corrompem, mas podem não fechar juntos. A definição de pronto da
sessão-mãe roda no turno de integração, sobre a árvore toda. Filho não confere
a árvore (ela ainda muda); o filho pode ter a própria `DoneSet` só se o pai
declarar uma para ele (Q6).

### RN-8 — Todo fim de filho tem nome

`done`, `incomplete`, `unverified`, `max_iterations`, `error`, `interrupted`
(a pessoa interrompeu), `cancelled` (a pessoa ou o pai fechou), `timeout` (o
prazo do filho passou), `budget` (o teto do lote acabou). Nenhum filho
"some": um daemon que reinicia leva os filhos, como leva o pai, e o registro de
cada um diz até onde foi.

O prazo é por filho, com padrão na configuração (`orchestration.child_deadline`,
proposto `30m`), e o que vence é o relógio de parede do filho **fora** do tempo
esperando aprovação — esperar você não é trabalhar devagar.

### RN-9 — Teto da sessão-mãe, nunca do filho

- **Filhos vivos ao mesmo tempo:** `orchestration.max_children`, proposto `4`,
  máximo `8`. Contam no `MaxSessions` do daemon como qualquer sessão; um lote
  que não cabe é recusado inteiro, dizendo qual teto.
- **Tokens:** os do filho são debitados do orçamento do pai
  (`delegated-writing` RN-6). Passado o teto, o daemon recusa lotes novos e
  avisa o pai e a pessoa; os filhos vivos terminam o turno corrente e param com
  `budget`. Nunca matar no meio de uma escrita.
- **Iterações:** cada filho tem as próprias (`DelegateMaxIterations`), menores
  que as do pai.

### RN-10 — Quem fecha quem

- Fechar o pai (`DELETE /sessions/{id}`) encerra os filhos: nada sobrevive a
  quem o criou (D19, núcleo).
- Fechar ou interromper um filho não toca o pai: o pai recebe `child.completed`
  com `cancelled` ou `interrupted`.
- Desfazer é por sessão: o `undo` do filho desfaz o último turno do filho; o do
  pai, o do pai. O desfazer "do lote inteiro", que a `delegated-writing` RN-4
  dava de graça por estar num turno só, aqui é uma operação nova (Q7).

## 5. Mudanças no protocolo

`client-server-protocol` é `experimental`, abaixo de 1.0: **MINOR**. Tudo
aditivo; nenhum campo muda de sentido.

**Pedido e registro**

- `CreateSessionRequest.Orchestrate bool` — a sessão nasce orquestradora (Q1).
- `Session` e `Conversation` ganham `parent_id,omitempty`, `owns,omitempty` e
  `orchestrator,omitempty`. A lista (N3) já basta para a grade agrupar a
  família.
- `TurnStarted.Origin string,omitempty` — `""` (a pessoa), `"children"` (a
  integração da RN-6).

**Eventos, no log do pai**

```go
ChildStarted struct {
    BatchID string   `json:"batch_id"`
    ChildID string   `json:"child_id"`  // id da sessão filha
    Title   string   `json:"title"`     // nome curto dado pelo pai: fecha L4
    Task    string   `json:"task"`
    Owns    []string `json:"owns,omitempty"`
}

ChildCompleted struct {
    BatchID    string      `json:"batch_id"`
    ChildID    string      `json:"child_id"`
    Reason     string      `json:"reason"`    // RN-8
    Conclusion string      `json:"conclusion,omitempty"`
    Truncated  bool        `json:"truncated,omitempty"`
    Wrote      []string    `json:"wrote,omitempty"`
    Unwritten  []string    `json:"unwritten,omitempty"`
    Unread     []string    `json:"unread,omitempty"`
    Added      int         `json:"added,omitempty"`
    Removed    int         `json:"removed,omitempty"`
    Usage      *Usage      `json:"usage,omitempty"`
    Completion *Completion `json:"completion,omitempty"`
}
```

O progresso de cada filho **não** é copiado para o log do pai: está no log do
filho, que é uma sessão. Isso fecha L1, L2, L3 e L7 sem evento novo — a janela
do filho é o progresso dele.

**Rotas**

- Nenhuma rota nova para criar filho: quem cria é o daemon, a pedido do modelo
  do pai. A pessoa conduz um filho pelas rotas que já existem (`turns`,
  `steer`, `interrupt`, `approvals`, `DELETE`).
- `GET /v1/sessions/{id}/children` → `ListChildrenResponse{children: []Conversation}`,
  vivas e terminadas, para um cliente que não acompanha a lista inteira (a TUI).
- Recusas novas: `owns_overlap` (com os caminhos e o filho dono),
  `max_children_reached`, `orchestration_budget_exhausted`. Todas como erro da
  ferramenta, para o modelo ler, e como `session.error` no log do pai, para a
  pessoa.

## 6. Mudanças no núcleo

- **`internal/loop/orchestrate`** (pacote novo): a ferramenta `delegate`, que
  recebe um lote — `[{title, task, owns, path}]` —, confere tudo (RN-4, RN-9)
  e pede ao daemon as sessões; e a ferramenta `children`, que lê o estado do
  lote (para o pai responder "como está?" sem inventar). Nenhuma das duas
  espera filho terminar.
- **`internal/app`**: a sessão filha é montada pelo mesmo caminho que monta
  qualquer sessão (configuração do workspace, N2), com o modo do pai, a
  contenção reduzida, o registro sem as ferramentas de orquestrar, e as
  instruções do filho (`DelegateInstructions`, ajustadas para dizer que há uma
  pessoa que pode responder).
- **`internal/session`**: o registro de arrendamentos por workspace; o vínculo
  pai→filhos no `Manager` (fechar o pai fecha os filhos); a fila de relatos do
  pai; o turno de integração.
- **`internal/policy`**: a negação "arrendado por <filho>" para a escrita do
  pai.
- **Specs**: este `.r`; depois `.p` (tipos, eventos, recusas, invariantes com
  teste nomeado), `.config` (`max_children`, `child_deadline`, o orçamento, o
  modelo em que os limiares foram medidos) e `.i`. Changelog em
  `delegated-writing` (RN-3 muda para filho-sessão) e `client-server-protocol`.

## 7. TUI

A TUI continua uma conversa por vez, e isso basta:

- `dcode --orchestrate` e `/orchestrate` abrem uma sessão orquestradora.
- No pai, `child.*` é desenhado como a delegação já é, com título, dono, estado
  ao vivo (`GET /sessions/{id}/children` a cada mudança) e motivo do fim.
- Filho esperando aprovação vira uma linha no pai — "◆ busca-api espera
  aprovação" — e o `^R` lista os filhos sob o pai; escolher um anexa a TUI a
  ele, como já anexa a qualquer sessão viva. A aprovação é respondida lá.
- O resto (`steer`, `interrupt`, `undo`) é o de sempre, dentro do filho.

## 8. Desktop

### Grade

Os filhos entram na grade pelas regras de atenção (D30) — rodando, esperando
você, terminados desde que você olhou —, e duas coisas mudam:

- **A família fica junta**: dentro do mesmo grupo de prioridade, os filhos vêm
  logo depois do pai. Os lugares continuam não pulando (D30).
- **Os vínculos são ditos**: o topo do filho diz "↳ de <pai>" (clicar leva ao
  pai); o pai tem uma faixa com um ponto por filho (clicar leva ao filho).

```
Figura 1 — Grade 3×2: o orquestrador conversa enquanto quatro filhos trabalham
┌──────────────────────────┬──────────────────────────┬──────────────────────────┐
│ ✦ busca de ponta a ponta │ ↳ índice      ● rodando  │ ↳ api         ◆ espera   │
│ filhos ● ● ◆ ✓  3 de 4   │ owns internal/index/**   │ owns internal/api/**     │
│──────────────────────────│ escrevendo index.go      │ Pede rede · go get …     │
│ Delegou a 4 filhos       │ ⏱ 6m12s                  │ [1 permitir][2 sessão]   │
│ propriedade disjunta     │                          │ [3 negar]                │
│ você: e o cache?         │                          │                          │
│ ✦: o índice já guarda…   │                          │                          │
│ [Escreva…             ]  │ [Escreva para redirec…]  │ [Responda à aprovação ]  │
├──────────────────────────┼──────────────────────────┼──────────────────────────┤
│ ↳ tela        ● rodando  │ ↳ docs        ✓ pronto   │ outra conversa           │
│ owns web/search/**       │ escreveu 2 · +80 −3      │ (fora da família)        │
│ …                        │ ✓ verificado · 1 critério│                          │
│ [Escreva…             ]  │ [Escrever continua…   ]  │                          │
└──────────────────────────┴──────────────────────────┴──────────────────────────┘
```

- Fechar (×) um filho que roda pede confirmação e manda `DELETE`; o pai recebe
  `cancelled`.
- Quando o lote termina, a janela do pai mostra o turno de integração começando
  sozinho, com a marca "integrando os filhos" no lugar da pergunta.
- Maximizar o pai (D31) põe os filhos como pares no topo, na ordem do lote.

### Crew

```
Figura 2 — Crew: a família como árvore, e a aba Filhos do pai
┌ Painel ┬ Ativas ─────────────────┬ conversa ─────────────┬ Mudanças Pronto Filhos Sessão ┐
│ ▣      │ ✦ busca de ponta a ponta│ … fluxo do pai …      │ Lote 1 · 4 filhos · 3 de 4    │
│ ◷      │   ├ ● índice            │                       │  ● índice   internal/index/** │
│ ◎      │   ├ ◆ api   espera você │                       │  ◆ api      internal/api/**   │
│ ⚙      │   ├ ● tela              │                       │  ● tela     web/search/**     │
│        │   └ ✓ docs              │                       │  ✓ docs     docs/search.md    │
│        │ ● refactor do parser    │                       │ Tokens do lote  412k de 2M    │
│        │ dcode                   │                       │ Prazo por filho 30m           │
│        │   …                     │ [Escreva…           ] │ [Abrir todos na grade]        │
└────────┴─────────────────────────┴───────────────────────┴───────────────────────────────┘
```

A aba **Filhos** só existe numa orquestradora; a lista mostra a família como
árvore de um nível, porque não há neto (RN-5). O desktop lê só o fio:
`parent_id`, `owns`, `orchestrator` na lista, `child.*` no log do pai, e a
sessão do filho como qualquer conversa — nada parseado de texto (D8).

## 9. Como se mede

**Contratos de comportamento** (`.p`, com limiar medido contra modelo real,
`make eval`; fixtures em `internal/evals/testdata/evals/`):

| ID | Cenário | Esperado | Limiar proposto |
|---|---|---|---|
| `orchestrator-splits-disjoint` | tarefa com três partes independentes | um lote com `owns` disjuntos e títulos curtos | ≥ 85% |
| `orchestrator-keeps-talking` | pergunta da pessoa com filhos rodando | responde sem criar outro lote nem esperar | ≥ 90% |
| `orchestrator-integrates-before-done` | lote termina | roda a definição de pronto na árvore antes de dizer pronto | ≥ 90% |
| `orchestrator-names-the-failed-child` | um filho termina `incomplete` | nomeia o filho e o que faltou; não diz pronto | ≥ 95% |
| `orchestrator-does-not-split-the-small` | tarefa de um arquivo | não delega | ≥ 85% |

**Invariantes por asserção** (`go test`, no `.p`, cada uma com teste nomeado
para o `specguard`): sobreposição recusada inteira; pai negado em caminho
arrendado; filho sem as ferramentas de orquestrar; fechar o pai encerra os
filhos; débito de tokens; prazo não conta espera de aprovação; um relato por
filho; uma integração por lote; aprovação do filho nunca no log do pai.

**Réguas do desktop**: um cenário `orquestrador` no `check:daemon` — o modelo
roteirizado cria dois filhos, um pede aprovação, a régua responde na janela do
filho e confere no log do daemon que o pai continuou `idle` e recebeu os dois
`child.completed`. Atributos novos que a régua lê: `data-parent-id` na janela e
na linha da lista. `tests/daemon/**` é protegido: PR próprio. A régua visual
(telas 02 e 03) não tem família e deve continuar a 0,00%.

O defeito conhecido do harness de eval — não responder chamada delegada, o que
ensinou o modelo a desistir (`docs/ROADMAP.md`) — precisa de conserto **antes**
de medir estes contratos: um delegador de mentira que crie filhos de mentira e
devolva relatos plausíveis, sem fingir que rodou o que não rodou.

## 10. Entrega em passos (um tema por PR)

| # | Lado | PR | Depende |
|---|---|---|---|
| 1 | spec | `.p`, `.config`, `.i` desta família, com as respostas da §12 | o visto |
| 2 | núcleo | `feat(protocol): a session can have a parent` — `parent_id`, `owns`, `orchestrator`; `CreateSessionRequest.Orchestrate` | 1 |
| 3 | núcleo | `feat(session): owned paths are leased across sessions` (RN-4) | 2 |
| 4 | núcleo | `feat(loop): an orchestrator starts children without waiting` — `delegate`, `children`, `child.started` | 3 |
| 5 | núcleo | `feat(session): a child's report returns to its parent` — `child.completed`, fila de relatos, turno de integração | 4 |
| 6 | núcleo | `feat(session): children have a deadline and a budget` (RN-8, RN-9) | 5 |
| 7 | evals | `test(evals): a fake delegator answers delegated calls` | — |
| 8 | evals | `test(evals): the orchestrator contracts are measured` | 5, 7 |
| 9 | TUI | `feat(tui): an orchestrator shows its children` | 5 |
| 10 | desktop | `feat(desktop): a family stays together in the grid` | 2 |
| 11 | desktop | `feat(desktop): the orchestrator's children panel` (Crew) | 5 |
| 12 | desktop | `test(desktop): the daemon ruler runs an orchestrator` | 10 |
| 13 | núcleo | (Q4-B, se aprovado) `feat(session): a child can work in its own worktree` | 8 |

Passos 2–6: `CHANGELOG.md` (duas edições) e changelog de família, MINOR;
10–12: changelog do desktop e D-números.

## 11. Riscos

- **Custo.** Quatro filhos são quatro conversas pagas; o teto da RN-9 freia, e
  o contador do lote fica à vista no pai.
- **Atenção partida.** Quatro janelas pedindo aprovação ao mesmo tempo é o que a
  RN-3 do `explore` evitava. A grade ordena (D30) e nenhum pedido de filho
  expira (Q5): o custo é um filho parado, à vista, até a pessoa olhar.
- **Árvore incoerente** entre filhos disjuntos: é a RN-7, e o turno de
  integração é a resposta — o risco é o modelo pular a conferência, e o
  contrato `orchestrator-integrates-before-done` mede isso.
- **Filho sem `bash` não confere o próprio trabalho** (Q4). É o mesmo limite do
  `explore` escritor; aceitável para a primeira entrega.
- **Arrendamento esquecido** por um filho que travou: o prazo (RN-8) libera.
- **A `delegated-writing` RN-3 muda de sentido para um tipo de filho**: o
  changelog dela tem de dizer, ou quem ler "filho não pergunta" conclui errado.
- **Daemon do desktop (D19)**: fechar o app mata pai e filhos; a pergunta de
  fechar passa a contá-los.

## 12. Decisões — 2026-10-13, aprovado pelo dono

**Q1. Quem é orquestrador?**
- A. Toda sessão tem `delegate`.
- B. Opt-in na criação (`Orchestrate`), com prompt e ferramentas próprios.
- C. Um modo novo, ao lado de plan/assist/auto.
- **Decidido: B** (aprovado pelo dono). A muda o prefixo e o comportamento de toda sessão e os
  limiares medidos; C mistura "o que pode escrever" com "como trabalha", que o
  modo hoje não mistura.

**Q2. O orquestrador também tem `explore`?**
- A. Sim: lê de forma síncrona, delega de forma assíncrona.
- B. Não: só `delegate`, para uma forma só de dividir.
- **Decidido: A** (aprovado pelo dono). Reconhecer o terreno antes de dividir é leitura curta, e
  abrir uma sessão para isso é janela demais.

**Q3. Quando o pai integra?**
- A. Sozinho, uma vez, quando o lote inteiro termina.
- B. A cada filho que termina.
- C. Só quando a pessoa pedir.
- **Decidido: A** (aprovado pelo dono). B gasta um turno por filho e integra árvore pela metade; C
  deixa trabalho pronto parado sem ninguém dizer.

**Q4. Onde os filhos escrevem?**
- A. Na mesma árvore, `owns` disjuntos, sem `bash` (o que o `explore` escritor
  já faz).
- B. Uma `git worktree` por filho, com todas as ferramentas, e o pai junta.
- C. Na mesma árvore, com `bash` e o sandbox de escrita reduzido ao `owns`.
- **Decidido: A agora, B depois** (aprovado pelo dono) (passo 13). A reaproveita tudo que existe e a
  `delegated-writing` RN-7 já mostrou que worktree não resolve incoerência, só
  a adia para a junção. B vale quando os conjuntos não puderem ser disjuntos ou
  quando o filho precisar rodar testes; C promete uma contenção que o sandbox
  de cada plataforma teria de provar.

**Q5. Aprovação de filho que ninguém respondeu.**
- A. Mesmo prazo e mesma negação de toda sessão (120 s), relatada ao pai.
- B. Sem prazo para filho: espera a pessoa, e o prazo do filho (RN-8) não conta.
- C. Uma resposta nova, "permitir para o lote", dada uma vez.
- **Decidido: B** (aprovado pelo dono). O prazo existe para um turno não ficar preso; num filho, o
  pai não está preso, e negar sozinho perde trabalho que a pessoa só não viu
  ainda. C mexe no consentimento (ADR-02) e merece spec própria.

**Q6. O filho tem definição de pronto própria?**
- A. Não: só o pai confere, na árvore inteira.
- B. Sim, se o pai declarar critérios por filho no lote.
- **Decidido: A** (aprovado pelo dono) na primeira entrega. Um critério do filho roda numa árvore que
  outros filhos estão mudando; B só faz sentido com Q4-B.

**Q7. Desfazer o lote.**
- A. Só por sessão (o `undo` de cada um).
- B. Uma operação "desfazer o lote" no pai, que desfaz o último turno de cada
  filho do lote.
- **Decidido: A agora** (aprovado pelo dono), com B como proposta futura: desfazer N sessões de uma
  vez é atômico só no nome se uma delas recusar um arquivo.

**Q8. O nome do filho.**
- A. O pai dá um título curto no lote (`title`).
- B. O daemon deriva da tarefa, como deriva o título de uma conversa.
- **Decidido: A** (aprovado pelo dono), com B quando faltar: o pai sabe o que separou; fecha L4.

## 13. Fora de escopo

- Neto (RN-5).
- Filhos em outro workspace que não o do pai.
- Junção automática de worktrees (vem com Q4-B, se vier).
- Um filho mandar mensagem a outro filho. Tudo passa pelo pai.
- O `/loop` com filhos (cada pasta orquestrada): espera este e o `/loop` no
  desktop (`desktop/docs/proposals/loop.md`) estarem construídos.

## 14. Changelog

- 2026-10-13 — o dono aprovou: Q4 A (mesma árvore, `owns` disjuntos, sem
  `bash`; worktree depois), Q5 B (o pedido do filho espera a pessoa, fora do
  prazo dele; RN-3 e §11 reescritas) e as recomendações de Q1, Q2, Q3, Q6, Q7
  e Q8. Constrói-se depois do `/loop` no desktop.
