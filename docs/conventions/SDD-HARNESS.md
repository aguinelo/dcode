# SDD aplicado a um harness de agente

> **Este documento não altera o RPI.** O protocolo canônico da ArcaSolucoes — 4 arquivos `.r`/`.p`/`.config`/`.i`, hierarquia `.r` > `.p`/`.config` > `.i`, nomenclatura por timestamp, sincronia spec↔código, changelog, tolerância zero a perda de dados — vale aqui integralmente e sem exceção.
>
> O que este guia faz é responder **como preencher esses 4 arquivos** quando o produto é um harness de agente, onde parte do comportamento é mediada por modelo. Nenhum artefato novo, nenhuma regra nova.

---

## 1. O problema

A regra de ouro do `.p.spec.md` é *"use EXATAMENTE os nomes, campos e tipos definidos"*. Isso resolve contrato de API e modelo de dados. Não resolve isto:

> "quando o `Edit` falha por match ambíguo, o agente relê o arquivo em vez de tentar de novo às cegas"

É comportamento mediado por modelo. Não é schema, e não é verificável por asserção.

A tentação é criar um quinto arquivo para isso. **É a saída errada** — divergiria do protocolo compartilhado entre os repos e quebraria as ferramentas que auditam o RPI. A saída certa é reconhecer que um contrato comportamental com limiar **já é um contrato técnico**, e contrato técnico é exatamente o que o `.p` guarda.

---

## 2. Onde cada coisa entra

| Preocupação | Arquivo canônico | Por quê |
|---|---|---|
| Qual comportamento é determinístico e qual é mediado por modelo | `.r.spec.md` | É verdade de domínio sobre o que o sistema é. Contexto, não contrato. |
| Cenários de comportamento com limiar | `.p.spec.md` | Um limiar é contrato técnico verificável. Mesmo estatuto de um schema. |
| Modelo e versão contra os quais o limiar foi medido; limiares como constante | `.config.spec.md` | É definição de ambiente — muda por ambiente, igual a feature flag. |
| Construir a suíte de eval e as fixtures | `.i.spec.md` | É passo de execução, com ordem e dependência. |
| Nível de estabilidade de contrato público | `.p.spec.md` | É propriedade do contrato. O `changelog/` do RPI já dá a semântica de mudança. |

---

## 3. Fronteira de determinismo no `.r.spec.md`

Todo `.r.spec.md` deste projeto declara em qual regime o escopo dele opera:

| Regime | Significado | Como se verifica |
|---|---|---|
| **Determinístico** | comportamento definido por regra explícita | asserção em `go test` |
| **Mediado por modelo** | comportamento emerge da interação com o LLM | limiar estatístico sobre fixtures |
| **Misto** | a spec cobre os dois; a seção diz **onde** fica a linha | ambos, separadamente |

Sem essa declaração, a revisão cobra o padrão errado do artefato errado — exige asserção de comportamento estatístico, ou aceita limiar onde cabia garantia.

**Corolário de arquitetura:** empurrar comportamento para o lado determinístico é objetivo de design, não acidente. Se a montagem de contexto for função pura `(estado da sessão) → []Message`, ela é golden-testável com exatidão — e o append-only da ADR-03 já torna isso natural, porque o prefixo é função pura do histórico. Vale igual para dispatch de ferramenta, decisão de sandbox e parsing de tool-call.

---

## 4. Contratos comportamentais no `.p.spec.md`

Quando o `.r` classifica o escopo como mediado por modelo ou misto, o `.p` ganha uma seção **"Contratos comportamentais"** com tabela de cenário e limiar. É seção comum de `.p`, sujeita à mesma regra de ouro: os identificadores de cenário são nomes exatos, usados como tal no código e nas fixtures.

| ID | Cenário | Comportamento esperado | Limiar | Fixture |
|---|---|---|---|---|
| `edit-ambiguous` | `Edit` com match ambíguo | relê o arquivo, não faz retry cego | ≥ 95% | `testdata/evals/edit-ambiguous/` |
| `path-missing` | caminho inexistente | erro explícito, sem inventar caminho | 100% | `testdata/evals/path-missing/` |
| `compaction-long` | compactação em tarefa longa | tarefa corrente sobrevive ao corte | ≥ 98% | `testdata/evals/compaction-long/` |

**Regras de uso:**

1. A verificação de um contrato comportamental é medição com limiar, nunca booleano.
2. Limiar de 100% só é legítimo quando o comportamento é, na verdade, determinístico. Nesse caso questione se o cenário não pertence a outra seção do `.p`, verificável por asserção.
3. Medição depende de modelo real e custa dinheiro: fica atrás de build tag ou `testing.Short()`, fora do `go test` padrão.
4. Regressão abaixo do limiar é blocker de PR, igual a teste vermelho.
5. **Rebaixar limiar no mesmo PR que o quebra é o antipadrão que esta seção existe para pegar.** Mudança de limiar é mudança de regra e exige entrada em `changelog/`, conforme a seção 6 do `RPI-SPEC-RULES.md`.

O modelo e a versão contra os quais o limiar foi medido ficam no `.config.spec.md` — trocar de modelo invalida o limiar, não o cenário.

---

## 5. Nível de estabilidade no `.p.spec.md`

`sales-api` é interno: quebrar contrato é problema de coordenação. Aqui, três coisas são contrato com terceiros — **protocolo client-server**, **ABI de plugin** e **schema de config**.

Todo `.p.spec.md` que define contrato público declara o nível logo na primeira seção:

| Nível | Significado |
|---|---|
| `experimental` | pode quebrar em qualquer versão, sem changelog |
| `stable` | quebra exige entrada em `changelog/` + incremento de major |
| `frozen` | não muda; só extensão aditiva |

O nível vale para a spec inteira, e um endpoint ou símbolo individual pode declarar nível próprio mais restritivo. Os critérios de promoção ficam escritos no `.i.spec.md` — não é decisão de momento.

Isso não cria mecanismo novo: o `changelog/` do RPI já é o registro de mudança de regra. A declaração apenas diz **quais** mudanças o exigem.

---

## 6. Efeito nas ferramentas

Nenhum. As specs continuam sendo 4 arquivos `.spec.md` em `docs/specs/**` com timestamp compartilhado. A `embarca-pr-review` audita este projeto sem qualquer alteração — o que ela valida é a taxonomia, e a taxonomia está intacta.

O checklist de revisão específico de Go vive em `docs/conventions/GO-CODE-REVIEW.md`, neste repositório, como convenção do projeto.
