# dcode

🇬🇧 [English version](README.md)

**Dreibox Code** — um harness de codificação agêntico para o terminal, escrito em Go.

> **Status: fase de especificação.** Ainda não há implementação — este repositório contém
> hoje decisões de arquitetura e specs. Nada aqui é instalável ou executável.
> Dar uma estrela significa interesse no rumo do projeto, não que ele já faça algo.

---

## Por que mais um

Quatro agentes de codificação de terminal já fazem isso bem, e cada um é genuinamente bom
em algo diferente:

| | Linguagem | Licença | Mais forte em | Mais fraco em |
|---|---|---|---|---|
| [Claude Code](https://github.com/anthropics/claude-code) | TS + Rust | source-available | engenharia de contexto, design de ferramentas | cold start, provider único |
| [Codex CLI](https://github.com/openai/codex) | Rust | Apache-2.0 | sandbox aplicado pelo SO, governança | amarração a um provider |
| [opencode](https://github.com/anomalyco/opencode) | TypeScript | MIT | 75+ providers, extensibilidade | peso do runtime |
| [jcode](https://github.com/1jehuang/jcode) | Rust | MIT | latência de inicialização, RAM por sessão | nenhum sandbox |

A lacuna é a interseção que nenhum deles ocupa: **densidade de sessão _e_ um sandbox real
aplicado pelo sistema operacional _e_ neutralidade de provider.** jcode tem a performance
e nenhum sandbox. Codex tem o sandbox e é preso a um provider. opencode tem a neutralidade
e o runtime mais pesado dos quatro.

Essa interseção é a razão deste projeto existir.

---

## Decisões de arquitetura

Cinco decisões, registradas antes de qualquer código. Cada uma é restrição estrutural
sobre tudo que vem depois.

### Go no núcleo

Escolhido sobre Rust e TypeScript. Go entrega cerca de 90% do perfil de performance do
Rust com o melhor ferramental de CLI e TUI de qualquer linguagem, um modelo de
concorrência que cai diretamente sobre o problema (N sessões, streaming SSE,
multiplexação de PTY) e o pool de contribuidores mais denso neste domínio específico.

A versão honesta: **Go e Rust ficaram dentro do ruído um do outro.** A matriz ponderada
que produziu esta decisão separou os dois por um dígito numa escala de 115 pontos, o que
não é resolução suficiente para chamar um de correto. O custo aceito do Go é pressão de
GC sob muitas sessões concorrentes — que ataca exatamente a tese acima, então uma meta de
memória por sessão entra como teste de regressão desde o primeiro dia.

### Sandbox e aprovação são preocupações separadas

Copiado inteiro do Codex, porque é o modelo mais bem desenhado dos quatro.

- **Sandbox** é fronteira técnica aplicada pelo kernel — `read-only`, `workspace-write`,
  `full-access`. Apple Seatbelt no macOS, Landlock e bubblewrap no Linux, Windows Sandbox
  no Windows.
- **Política de aprovação** é decisão de autorização, ortogonal à fronteira —
  `untrusted`, `on-request`, `never`.

Manter os dois separados é o que reduz fadiga de aprovação. Harnesses que misturam as duas
coisas perguntam demais, o usuário desliga o prompt inteiro, e o modelo de segurança vira
decoração.

### Contexto append-only

A decisão de performance de maior alavancagem. **O prefixo do contexto nunca é mutado
entre turnos.** Editar qualquer coisa no início da conversa invalida o cache KV inteiro e
recobra o prompt completo, em latência e em dinheiro.

Consequências que a maioria dos harnesses erra:

- Schemas de ferramenta MCP são anunciados no startup a partir de cache. Um servidor que
  conecta no turno 5 e injeta definições invalida o cache da sessão inteira.
- Nada de timestamp, contador de tokens ou estado volátil no system prompt.
- Compactação é rara e em bloco, nunca incremental.

### Client-server desde o primeiro commit

O núcleo roda como daemon local; o TUI é apenas um cliente. É a decisão mais barata agora
e mais cara de retrofitar — aplicativo desktop, extensão de IDE, sessão compartilhada e
execução remota caem toda dela, e nenhuma cabe num monolito de TUI.

### Agnóstico de provider, com camada de adaptação real

Não é só troca de endpoint. Harnesses agnósticos perdem para os afinados rodando o mesmo
modelo, porque system prompt, schema de ferramenta e estratégia de edição precisam ser
adaptados por *família* de modelo. Essa camada é trabalho orçado, não configuração.

---

## Como isto é construído

Desenvolvimento guiado por especificação, usando o **protocolo RPI** — quatro arquivos
`.spec.md` que compartilham um prefixo de timestamp, com ordem estrita de precedência:

| Arquivo | Papel | Regra |
|---|---|---|
| `.r.spec.md` | Research — contexto, user stories, regras de negócio | Verdade absoluta. Se o código contradiz, o código está errado. |
| `.p.spec.md` | Planning — schemas, contratos, tipos | Use exatamente os nomes e tipos definidos. |
| `.config.spec.md` | Config — env vars, flags, constantes de infra | Nenhuma env var nova no código sem entrada aqui. |
| `.i.spec.md` | Implementing — checklist ordenado de execução | Siga a ordem. |

Precedência: `.r` > `.p`/`.config` > `.i`.

### A parte interessante: spec para comportamento não determinístico

Um harness tem um problema que uma aplicação CRUD não tem — seu comportamento mais
importante é mediado por um modelo de linguagem. Isto não se escreve como schema:

> quando uma edição falha por match ambíguo, o agente relê o arquivo em vez de tentar de
> novo às cegas

Então todo `.r.spec.md` aqui declara em qual regime seu escopo opera — determinístico,
mediado por modelo, ou misto — e essa declaração decide como ele é verificado: asserção em
`go test`, ou limiar medido sobre fixtures.

O corolário é objetivo de arquitetura, não acidente: **empurre o máximo de comportamento
possível para o lado determinístico da linha.** Se a montagem de contexto for função pura
do estado da sessão, ela é exatamente golden-testável — e o contexto append-only torna
isso natural, porque o prefixo é função pura do histórico.

Detalhes em [`docs/conventions/SDD-HARNESS.pt-BR.md`](docs/conventions/SDD-HARNESS.pt-BR.md).

---

## Estrutura do repositório

```
docs/
  conventions/            bilíngue — X.md é inglês, X.pt-BR.md é português
    LANGUAGE.md           a própria política bilíngue
    SDD-HARNESS.md        como aplicar SDD a um harness
    TESTING.md            TDD, regra de reprodução de bug, gate de cobertura
    GO-CODE-REVIEW.md     checklist de revisão de Go, com checks do produto
  specs/                  só português — ver LANGUAGE.md, seção 3
    architecture/         specs transversais (protocolo, contexto, sandbox)
    domains/              specs de funcionalidade
```

A implementação vai viver em `internal/` (tudo que não é contrato público) e `pkg/` (tudo
que é).

---

## Status atual

| Área | Estado |
|---|---|
| Decisões de arquitetura | registradas |
| Spec do protocolo client-server | escrita, `experimental` |
| Spec do motor de contexto | escrita, `experimental` |
| Spec do adaptador de provider | escrita, `experimental` |
| Spec do loop do agente | escrita, `experimental` |
| Spec de sandbox e política | escrita, `experimental` |
| Spec do conjunto de ferramentas | escrita, `experimental` |
| Spec de distribuição | escrita, contrato de artefato `stable` |
| Implementação | **não iniciada** |

O primeiro marco de implementação é o vocabulário de tipos do protocolo — sem servidor,
sem I/O, só os tipos compartilhados e seus testes de ida e volta.

**Modelos.** MiniMax M3 é o modelo principal e é construído e medido primeiro; Claude vem
depois, e é o que prova que o split transporte/família é de fato ortogonal. *Transporte* é
formato de fio (`openai`, `anthropic`) e é reusável; *família* é a camada de adaptação e é
quem carrega os limiares medidos e os limites de turno por modelo. "OpenAI-compatible"
descreve formato de fio, nunca comportamento — modelo desconhecido atrás desse endpoint
herda o fio, jamais os limiares validados de outro modelo.

**Marco de auto-hospedagem.** Um pull request ao dcode escrito de ponta a ponta pelo
dcode, aprovado na revisão e passando o gate de 90%, sem edição manual. É a melhor eval
que o projeto tem: a própria suíte de testes e o checklist de revisão viram função de
aptidão.

---

## Contribuindo

Cedo demais para contribuição de código — as specs ainda estão em movimento.

O que é útil agora é argumento. Se alguma decisão acima parecer errada, abra uma issue e
diga por quê. O raciocínio está escrito justamente para poder ser atacado: toda decisão
tem custo declarado, e a escolha Go contra Rust em particular ficou perto o bastante para
que informação nova a inverta.

### Workflow

**GitHub Flow.** `main` está sempre pronta para deploy. O trabalho acontece em branches de
vida curta cortadas de `main` e volta por pull request.

```
main ──┬─────────────────────────┬──▶
       └── feat/event-log ── PR ─┘
```

Nome de branch segue o tipo da mudança: `feat/`, `fix/`, `docs/`, `chore/`, `refactor/`.
Para trabalho guiado por spec, use o slug da spec sem o timestamp —
`feat/client-server-protocol`.

**[Conventional Commits](https://www.conventionalcommits.org/).** Em toda mensagem de
commit e todo título de PR. O prefixo de tipo é o que alimenta geração de changelog e
comunica o raio de impacto de relance.

```
feat:     capacidade nova
fix:      correção de bug
docs:     só documentação
refactor: mudança que preserva comportamento
perf:     performance, sem mudança de comportamento
test:     só testes
chore:    build, ferramental, dependências
```

Mudança quebrando contrato leva `!` antes dos dois-pontos (`feat!:`) e explica a quebra no
corpo. Para qualquer coisa marcada como `stable` num `.p.spec.md`, quebra também exige
entrada em `changelog/` e incremento de major.

Commit que muda comportamento técnico precisa manter a spec correspondente sincronizada —
spec nunca pode ficar obsoleta em relação ao código.

**Autoria.** Commit tem autor único e nenhum trailer de coautoria. Todo commit é
atribuível a uma pessoa; ferramenta que auxiliou não recebe crédito.

### Testes

**TDD.** Teste primeiro, veja falhar, depois escreva o código. Teste que nunca foi visto
vermelho não é rede de segurança.

**Todo bug ganha teste de reprodução — antes da correção.** Reproduza o bug num teste que
falha, confirme que ele falha pelo sintoma relatado, então corrija, e o mesmo teste passa
sem ser alterado. PR de `fix:` sem teste novo é bloqueado. Teste de regressão é permanente
e nunca é simplificado.

**Gate de cobertura de 90%**, com denominador explícito: código determinístico em
`internal/` e `pkg/`. Código gerado, wiring de `main` e caminhos mediados por modelo ficam
de fora — o último porque não é verificável por asserção de forma alguma, só por limiar
medido sobre fixtures.

Essa exclusão é pressão deliberada na direção certa: comportamento do lado determinístico
da linha conta para a cobertura e é exatamente verificável, então o incentivo é continuar
movendo comportamento para lá.

O gate é piso, não meta. Ele pega arquivo sem teste nenhum; não prova correção — e teste
que exercita uma linha sem afirmar nada é achado de revisão mesmo com a cobertura verde.

Regras completas em [`docs/conventions/TESTING.pt-BR.md`](docs/conventions/TESTING.pt-BR.md).

### Idioma

Este projeto é bilíngue. Inglês é canônico e fica com o nome sem sufixo; português é a
tradução e leva o sufixo `.pt-BR`. O README e tudo em `docs/conventions/` existem nos dois,
com link cruzado no topo.

Duas exceções deliberadas:

**Specs são só em português.** O RPI define o `.r.spec.md` como verdade absoluta — se o
código contradiz, o código está errado. Essa regra precisa de exatamente uma fonte da
verdade. Duas cópias de uma spec vão divergir, e quando discordarem não há como saber qual
delas o código deveria satisfazer. Spec divergente é pior que spec ausente, porque parece
autoritativa.

**Commits e comentários de código são só em inglês.** Mensagem de commit alimenta
ferramenta de changelog que assume idioma único, e corpo bilíngue dobra o ruído no
`git log` sem ajudar quem já está lendo código em inglês.

PR que altera documento bilíngue deve atualizar as duas versões no mesmo PR. Política
completa em [`docs/conventions/LANGUAGE.pt-BR.md`](docs/conventions/LANGUAGE.pt-BR.md).

---

## Licença

[MIT](LICENSE) — a mesma licença do opencode e do jcode.
