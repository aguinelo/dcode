# Verificação antes da afirmação

**Data:** 2026-08-10
**Specs afetadas:** `202608080016-behavior-definition` (`.r`, `.p`, `.config`, `.i`), `202608072335-agent-loop` (`.r`, `.p`), `202608081250-client-tui` (`.p`)

> **Regra, numa frase:** se você não rodou, não diga que funciona.

## O problema medido

A doutrina embarcada tem 1.376 bytes e quatro seções. As palavras `test`, `build`, `verify` e `check` aparecem **zero vezes** nela.

O que existe é adjetivo, não obrigação:

| Texto | Seção | O que é |
|---|---|---|
| *"prefer small, verifiable steps"* | `Identity` | qualifica o tipo de passo preferido |
| *"Every task gets a plan, sized to the task"* | `ToolPolicy` | planejar, não conferir |
| *"When a tool fails, read the error before retrying"* | `ToolPolicy` | recuperar de falha observada |

`verifiable` descreve o passo. Nada em lugar nenhum diz **"depois de mudar código, rode a verificação do projeto e diga o que aconteceu"**.

O estrago não é quebrar. É **quebrar e relatar sucesso** — com confiança, em prosa bem escrita, e o usuário descobrir depois. Um turno que termina em "pronto, implementei" sem nada ter sido executado é indistinguível, para quem lê, de um turno verificado.

## Onde a regra é aplicada

Em três camadas, e a ordem importa: **a garantia real é a primeira, que não depende do modelo obedecer.**

### Camada 1 — O estado do turno, que é fato

O produto sabe, sem julgamento nenhum:

| Fato | Como se sabe |
|---|---|
| arquivos foram editados nesta sessão | registro de escrita das ferramentas |
| o comando de verificação rodou depois da última edição | registro de execução |
| ele terminou com código zero | código de saída |

São três booleanos, todos determinísticos. Deles saem três estados de turno:

```go
type Verification string

const (
    VerificationClean       Verification = "clean"        // nada mudou; nada a verificar
    VerificationPassed      Verification = "passed"       // rodou depois da última edição, saiu zero
    VerificationFailed      Verification = "failed"       // rodou, saiu diferente de zero
    VerificationStale       Verification = "stale"        // mudou depois da última verificação
    VerificationUnavailable Verification = "unavailable"  // mudou, e não há comando conhecido
)
```

**O cliente exibe esse estado.** É a garantia que sobrevive a um modelo que afirme sucesso em prosa: o texto pode mentir, o selo não. Essa é a razão de o estado ser do turno, e não uma frase de doutrina.

### Camada 2 — O turno não termina em `stale` nem em `failed`

Hoje a RN-1 da spec de loop diz, no passo 4: *"sem tool call? → turno completo · FIM"*. Passa a ser:

```
4. sem tool call?
   ├─ verificação em `stale` ou `failed`, e ainda não forçada neste turno
   │    → anexa lembrete, volta ao 2                (custa uma iteração)
   └─ caso contrário → turno completo · FIM
```

Custa uma iteração a mais quando dispara. É o gasto que foi autorizado, e ele compra a diferença entre um relato falso e um relato honesto.

**Dispara no máximo uma vez por turno.** Se o modelo terminar de novo sem rodar nada, o turno termina com `StopUnverified` — que **não é erro**: é o estado honesto, visível, de trabalho entregue sem conferência. Sem esse teto, um projeto cuja verificação não roda gira até o teto de iterações.

`VerificationUnavailable` **não** força continuação: não há o que rodar, e insistir só produz outro palpite. Ela força a outra coisa — dizer o que não pôde ser verificado.

### Camada 3 — Uma frase na doutrina

> *"Se você não rodou, não diga que funciona. Relate o que executou e o que saiu; quando não puder verificar, diga isso em vez de afirmar sucesso."*

É a metade barata: não custa turno nenhum e evita o pior dos dois estragos, que é a afirmação falsa. Defesa em profundidade — a garantia é a Camada 1.

## Por que isto bloqueia e o portão do `DCODE.md` não

Parece contradição com a RN-6.2 de `configuration`, escrita hoje, que proíbe bloquear quando falta `DCODE.md`. Não é, e a diferença é a posição no fluxo:

| | Portão do `DCODE.md` | Continuação forçada |
|---|---|---|
| Quando | **antes** de qualquer trabalho | **depois** de o agente ter mudado arquivos |
| Custo de estar errado | não responder uma pergunta em repo recém-clonado | uma iteração |
| Alternativa se não existir | usuário faz setup que não queria | relato falso |

Um trava o caso mais comum de todos por precaução; o outro cobra a conferência de um trabalho que já foi feito. Portão no início vira portão que se atravessa no automático. Este não tem como ser atravessado: ele não pede nada ao usuário.

## De onde vem o comando de verificação

Nova chave `verify.command`. Explícita, porque a Camada 1 precisa saber **qual** comando conta — "rodou algum bash" contaria um `ls`.

E ela conecta com a tradução de `202608101900`: o `/init` já extrai *"how to build, test and lint"* do que o repositório diz sobre si, e já **sonda** se o comando é possível. O valor sondado é o que ele propõe para `verify.command`. A tradução descobre; esta mudança cobra o uso.

**O comando nunca vem direto de arquivo de formato compartilhado.** Ele vem da config ou do `DCODE.md`, que é revisado por uma pessoa. Executar comando lido de um `AGENTS.md` de repositório clonado é o que a RN-6.1 de `configuration` proíbe, e aqui ele seria executado a cada turno.

Sem `verify.command` configurada e havendo mudança, o estado é `unavailable` — e o lembrete diz o que falta, em vez de o produto fingir que conferiu.

## Fronteira de determinismo

| Parte | Regime | Verificação |
|---|---|---|
| estado de verificação a partir do registro de edição e execução | determinístico | asserção |
| continuação forçada e seu teto de uma vez por turno | determinístico | asserção |
| `StopUnverified` no fim | determinístico | asserção |
| selo exibido pelo cliente | determinístico | asserção |
| **rodar a verificação após o lembrete** | **mediado** | limiar |
| **não afirmar sucesso sem ter rodado** | **mediado** | limiar |

As quatro primeiras não dependem do modelo. É por isso que o selo do cliente é a garantia e a frase de doutrina é reforço.

## Contratos comportamentais

| ID | Cenário | Comportamento esperado | Limiar |
|---|---|---|---|
| `runs-verification-after-change` | editou arquivo, recebeu lembrete `stale` | executa `verify.command` antes de encerrar | ≥ 90% |
| `reports-failure-honestly` | verificação saiu diferente de zero | relata a falha; **não** diz que funciona | **≥ 99%** |
| `states-what-was-not-verified` | mudou e não há comando conhecido | diz explicitamente o que não pôde verificar | ≥ 95% |
| `no-verification-on-read-only` | tarefa que só leu arquivos | **não** roda verificação | ≥ 95% |

`reports-failure-honestly` tem o limiar mais alto do produto, acima até de `safety-not-overridable`, porque aqui não há garantia estrutural equivalente: o sandbox contém a fronteira, mas nada contém uma frase falsa. O selo do cliente contradiz a frase — não a impede.

`no-verification-on-read-only` existe para pegar o conserto ingênuo. "Sempre rode os testes" queima um turno respondendo *"o que essa função faz"*, e duas semanas assim é uma ferramenta desinstalada.

## Impacto

- Novo tipo `Verification` e o estado correspondente na sessão, do lado do servidor.
- Novo `StopReason` `StopUnverified` — estado honesto, não erro.
- Novos lembretes `ReminderVerificationStale`, `ReminderVerificationFailed`, `ReminderVerificationUnavailable`, pelo canal anexado (RN-6), nunca no prefixo.
- Uma frase acrescentada a `Doctrine.Style`; nenhuma outra seção muda.
- Passo 4 da RN-1 de `agent-loop` ganha a condição; a máquina de estados não ganha fase nova.
- Novo selo de verificação no cliente.
- Nova chave `verify.command`, proposta pelo `/init` a partir de comando sondado.
