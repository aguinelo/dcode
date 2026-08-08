# Research: Definição de Comportamento

> Fonte da verdade de negócio para **como o comportamento do agente é definido** — system prompt, descrições de ferramenta, instruções do usuário, skills e lembretes de runtime.
> Depende de: **ADR-03** (contexto append-only), **ADR-05** (adaptação por família).

## 1. Contexto

Comportamento de agente não é "um prompt". É uma **pilha de camadas** com custos e precisões diferentes, e a engenharia real está em decidir em qual camada cada regra vive.

O erro comum é empilhar tudo no system prompt: ele cresce, a atenção dilui, o prefixo fica caro em todo turno, e a regra acaba longe do momento em que importaria.

Esta spec define as camadas, a ordem de montagem, a precedência em caso de conflito, e — mais importante — **o critério para decidir onde uma regra nova deve morar**.

## 2. Fronteira de determinismo

**Regime: misto.** É a terceira spec mista do projeto, e a com a linha mais delicada.

| Parte | Regime | Verificação |
|---|---|---|
| Montagem do prompt e ordem das camadas | determinístico | golden file |
| Precedência entre instruções conflitantes | determinístico | asserção |
| Estabilidade de cache do prefixo | determinístico | asserção |
| Emissão de lembrete a partir do estado | determinístico | asserção |
| **Aderência do modelo à instrução** | **mediado por modelo** | limiar |
| **Escolha de ferramenta dedicada sobre shell** | **mediado por modelo** | limiar |

A montagem é totalmente determinística. O que não é: **se o modelo obedece.**

## 3. User stories

| # | Como | Quero | Para |
|---|---|---|---|
| US-1 | usuário | dar instruções permanentes ao agente sobre meu projeto | não repetir convenção a cada turno |
| US-2 | usuário | que instrução de subdiretório vença a do projeto | monorepo com convenções distintas por pacote |
| US-3 | administrador | travar instruções que o usuário não sobrescreve | política organizacional |
| US-4 | usuário | que o agente saiba que um arquivo mudou embaixo dele | não agir sobre estado obsoleto |
| US-5 | usuário | adicionar capacidade sem inflar o prompt de toda sessão | prompt grande custa em todo turno |
| US-6 | autor de família | ajustar formulação ao modelo | mesma regra, fraseado diferente, resultado melhor |

## 4. Regras de negócio

### RN-1 — Seis camadas, com custo e precisão declarados

| Camada | Entra quando | Custo | Precisão |
|---|---|---|---|
| Doutrina base | sempre, no prefixo | alto e permanente | baixa |
| Descrição de ferramenta | sempre, no prefixo | alto e permanente | **alta** — lida no momento da decisão |
| Instrução do usuário | sempre, no prefixo | alto e permanente | média |
| Índice de skill | sempre, no prefixo — só o índice | baixo | alta |
| Corpo de skill | quando o gatilho bate | baixo | alta |
| Mensagem de erro de ferramenta | quando a falha ocorre | **zero até ocorrer** | **máxima** |

### RN-2 — Restrição estrutural vence instrução em prompt

**Se uma regra pode ser aplicada por código, ela não pertence ao prompt.**

O prompt existe para o que **não se consegue** aplicar estruturalmente. Tudo que vira invariante de código sai do prompt e passa a ser verificável por asserção — o mesmo objetivo de arquitetura descrito em `docs/conventions/SDD-HARNESS.pt-BR.md`.

Exemplo canônico, "não edite arquivo que não leu":

| Abordagem | Qualidade |
|---|---|
| Parágrafo no system prompt | fraca — esquecida em conversa longa |
| Linha na descrição da ferramenta | melhor — perto do ponto de uso |
| **Invariante aplicada no código + erro que ensina a recuperação** | **correta** |

### RN-3 — Mensagem de erro de ferramenta é superfície de comportamento
Não é diagnóstico: é **onde o comportamento de recuperação é ensinado**, no único instante em que é relevante, a custo zero até acontecer.

É a camada mais eficiente da pilha e a mais esquecida. Erro genérico força o modelo a adivinhar; adivinhar é como um agente corrompe arquivo.

### RN-4 — Ordem de montagem é por estabilidade; precedência é por especificidade
São duas coisas diferentes e confundi-las é erro comum.

**Ordem de montagem** existe para o cache: do mais estável ao mais volátil, para maximizar o prefixo casável (ADR-03).

**Precedência** existe para conflito: a instrução mais específica vence — exceto o que é travado, que vence tudo.

### RN-5 — O prefixo é montado uma vez por sessão
Doutrina, ferramentas, instruções e índice de skill são resolvidos na criação da sessão e não mudam enquanto ela viver.

Instrução que aparecesse no meio da sessão invalidaria o prefixo inteiro — o mesmo motivo pelo qual definições de ferramenta são fixadas no início.

### RN-6 — Lembrete é um terceiro canal, sempre anexado
Nem system prompt, nem mensagem de usuário: **canal próprio**, injetado no histórico.

- É **sempre anexado**, nunca prefixado. É o que permite direcionar comportamento no meio da sessão **sem invalidar o cache**.
- É **função pura do estado da sessão** — mesmo estado, mesmo lembrete. Sem isso, o histórico deixa de ser reproduzível.
- Não polui a conversa do usuário: o cliente não o exibe como fala dele.
- A doutrina base explica ao modelo o que é esse canal, senão ele o confunde com instrução do usuário.

### RN-7 — Disclosure progressivo: índice no prefixo, corpo sob demanda
Skill entra no prefixo como **uma linha** descrevendo quando usá-la. O corpo é anexado quando o gatilho bate.

Carregar todo corpo de skill no prefixo é o caminho mais rápido para um prompt de dezenas de milhares de tokens pago em todo turno, com atenção diluída.

### RN-8 — A formulação pertence à família; a regra, não
Duas famílias de modelo não respondem igual à mesma frase — uma prefere estrutura marcada, outra prosa direta.

A **regra** é única e vive nesta spec. A **formulação** é da família (ADR-05). Uma família que precise mudar a *regra*, e não só o fraseado, é sinal de que a regra está errada ou de que aquele modelo não é suportável.

### RN-9 — Instrução do usuário nunca sobrescreve segurança
Instrução de projeto ajusta estilo, convenção e preferência. **Não** desliga a fronteira do sandbox, não remove a exigência de aprovação, não altera a invariante de leitura prévia.

Instrução que tente isso é ignorada, e o fato é registrado — não silenciosamente descartado.

## 5. Fora de escopo

- **Caminhos concretos dos arquivos de instrução na máquina do usuário** e formato da configuração — spec própria, ainda a escrever.
- **Comandos internos** do cliente e como injetam instrução — mesma spec futura.
- Prompt de sub-agentes; não há sub-agentes no MVP.
- Memória semântica entre sessões.

## 6. Changelog

_Sem alterações desde a criação._
