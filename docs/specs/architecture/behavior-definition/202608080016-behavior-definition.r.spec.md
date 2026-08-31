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
| **Profundidade de planejamento proporcional** | **mediado por modelo** | limiar |

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
| US-7 | usuário | ver onde estamos numa tarefa longa | acompanhar sem reler o histórico |
| US-8 | usuário | que tarefa pequena não vire cerimônia | ferramenta burocrática é ferramenta abandonada |
| US-9 | usuário | nunca receber "pronto" sobre trabalho que não foi conferido | relato falso custa mais que turno a mais |

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

**Corpo que entra no turno é anunciado.** O índice é auditável — está no prefixo, e `--dump-prompt` o imprime. O corpo não era: ele era anexado ao histórico como lembrete sem nada ser emitido, então um bloco de texto entrava no turno, gastava contexto e mudava o comportamento do modelo, sem a pessoa ter como saber que aconteceu nem qual skill foi.

Trinta linhas acima da injeção, o teto do índice já se recusa a descartar em silêncio. O anúncio é essa mesma regra aplicada ao único fato observável que não estava viajando pelo log. Turno que não carrega skill nenhuma não anuncia nada: uma linha por turno cujo único conteúdo é que a funcionalidade existe é linha gasta.

**O gatilho bate no que distingue a skill, não no que ela tem em comum.** Sem `triggers` explícito, a linha de quando-usar é casada pelas próprias palavras significativas, e duas condições valem juntas: dois acertos distintos, e pelo menos um numa palavra que é daquela skill e de nenhuma outra do índice.

A primeira condição sozinha não bastava. Duas skills que dizem "projeto" e "versão" não se distinguem por essas palavras, então uma tarefa que citasse as duas carregava os dois corpos. Exigir um acerto que discrimina também mantém vizinhas de um mesmo domínio alcançáveis: elas compartilham a palavra do domínio e cada uma continua tendo a sua.

A lista de palavras vazias cobre **as duas línguas em que este produto é escrito**. Ela era só de inglês, e por isso `quando`, `projeto` e `estiver` contavam como significativas enquanto `when` e `that` não — a mesma frase era filtrada numa língua e não na outra. Uma lista por língua cobre só as línguas que estão nela; o que cobre o resto é `triggers`, casado como frase, que não passa por aqui.

### RN-8 — A formulação pertence à família; a regra, não
Duas famílias de modelo não respondem igual à mesma frase — uma prefere estrutura marcada, outra prosa direta.

A **regra** é única e vive nesta spec. A **formulação** é da família (ADR-05). Uma família que precise mudar a *regra*, e não só o fraseado, é sinal de que a regra está errada ou de que aquele modelo não é suportável.

### RN-9 — Planejamento é intrínseco, com profundidade proporcional
Toda tarefa tem plano. Não é opcional nem acionado por comando: é comportamento do produto.

O que **não** é fixo é a cerimônia. "Corrige esse typo" passando por descoberta, refinamento, plano e aprovação é absurdo, e é assim que uma ferramenta ganha fama de burocrática — a maioria das tarefas reais é pequena.

| Tamanho | Plano | Aprovação |
|---|---|---|
| trivial | 1 item | automática |
| médio | 3 a 5 itens | leve |
| complexo | descoberta e refinamento completos | explícita |

Quem decide a profundidade é o modelo, o que joga esta regra no regime mediado — daí o contrato comportamental correspondente.

O plano é **dado estruturado**, produzido pela ferramenta `plan`, nunca prosa no meio da resposta. É o que permite ao cliente exibi-lo sem interpretar texto.

**Plano vivo:** itens são adicionados, divididos e marcados como bloqueados durante a execução. Plano imutável força o agente a marcar como feito o que não fez, ou a travar. Realidade divergir de plano é o normal.

### RN-10 — Instrução do usuário nunca sobrescreve segurança
Instrução de projeto ajusta estilo, convenção e preferência. **Não** desliga a fronteira do sandbox, não remove a exigência de aprovação, não altera a invariante de leitura prévia.

Instrução que tente isso é ignorada, e o fato é registrado — não silenciosamente descartado.

### RN-11 — A pergunta é quem pode mudar, não o que pode mudar
A doutrina base tem quatro seções, e apenas uma delas — `Safety` — tem motivo próprio para ser imutável. Identidade e estilo não são superfície de ataque: ninguém é atacado pela reescrita do próprio tom de saída.

O que separa corretamente as seções é a **origem**. Configuração da raiz do usuário é a voz de quem é dono da máquina. Arquivo dentro do workspace veio junto com código que pode ter sido clonado de qualquer lugar, e **não é o usuário** — ainda que tenha o mesmo nome de arquivo.

Por isso a sobreposição de doutrina vem de **uma única raiz**, a do usuário. Não é preferência de desenho: um `.dcode/doctrine/identity.md` num repositório clonado redefiniria quem o agente pensa que é antes de qualquer instrução ser lida, que é o vetor da RN-10 por outra porta.

### RN-12 — Substituir e acrescentar são permissões distintas, decididas por seção
Cada seção declara **como** pode mudar, e isso não é escolha de quem configura.

| Seção | Substituir | Acrescentar | Motivo |
|---|---|---|---|
| `Identity` | sim | sim | preferência do dono da máquina |
| `Style` | sim | sim | idem |
| `ToolPolicy` | **não** | sim | descreve máquina que existe; declarar ferramenta inexistente faz o modelo chamar o que não há |
| `Safety` | **nunca** | **nunca** | RN-10 |

`Safety` não aceita nem apêndice porque acrescentar ao fim é funcionalmente substituir: o texto acrescentado pode dizer "ignore o acima". Seção que aceita apêndice não tem trava — tem trava que se contorna escrevendo mais um parágrafo.

A trava é **estrutural, não condicional**: `Safety` não é campo do tipo de sobreposição. Pela RN-2, regra que vira invariante de código sai do texto e passa a ser verificável por asserção; trava por convenção quebra no primeiro refactor, trava por tipo não compila.

Sobreposição aplicada é **sempre visível** — a auditoria do prompt marca a origem de cada seção. Substituição invisível seria pior que a imutabilidade que ela substitui, porque removeria a única forma de saber o que foi ao modelo.
### RN-13 — Se não rodou, não diga que funciona
Trabalho que mudou arquivo e não foi verificado **não é relatado como funcionando**. Essa é a regra; o resto é como aplicá-la.

Nada nesta doutrina exigia conferência: `verifiable` em `Identity` qualifica o tipo de passo preferido, não obriga a nada. O estrago não é quebrar — é **quebrar e relatar sucesso**, porque um turno que termina em "pronto" sem nada executado é indistinguível, para quem lê, de um turno verificado.

Aplicada em três camadas, e a garantia é a primeira:

1. **Estado de verificação do turno**, derivado de fato — houve edição, o comando de verificação rodou depois dela, e com que código de saída. **O cliente exibe esse estado.** É o que sobrevive a um modelo que afirme sucesso em prosa: o texto pode mentir, o selo não.
2. **O turno não termina** com mudança não verificada ou verificação falha: o lembrete é anexado e o ciclo continua, uma vez. Persistindo, termina em estado honesto de não verificado — que não é erro.
3. **Uma frase na doutrina.** Metade barata, custo zero de turno, e evita o pior estrago, que é a afirmação falsa.

Duas coisas que a regra **não** é:

- Não é "sempre rode os testes". Verificação se liga a **ter mudado algo**; disparar em tarefa de leitura queima turno e a ferramenta é desinstalada (US-8, RN-9).
- Não é sobre o comando existir. Não havendo comando conhecido, a obrigação vira **dizer o que não pôde ser verificado** — nunca fingir que conferiu.

O comando de verificação vem de configuração ou de arquivo específico revisado por uma pessoa, **nunca de formato compartilhado de terceiro** — ele passaria a ser executado a cada turno, que é a RN-6.1 de `202608081203-configuration` violada em laço.

## 5. Fora de escopo

- Caminhos concretos dos arquivos de instrução e formato da configuração — `202608081203-configuration`.
- Comandos do cliente e como injetam instrução — `202608081203-configuration`, seção 6.
- Exibição do plano — `202608081250-client-tui`.
- Prompt de sub-agentes; não há sub-agentes no MVP.
- Memória semântica entre sessões.

## 6. Changelog

- [202608081250 — Ferramenta `plan`](../tool-suite/changelog/202608081250-ferramenta-plan.md)
- [202608101800 — Doutrina editável por camada](changelog/202608101800-doutrina-editavel-por-camada.md)
- [202608102000 — Verificação antes da afirmação](changelog/202608102000-verificacao-antes-da-afirmacao.md)
