# Delegação somente-leitura

**Data:** 2026-08-10
**Specs afetadas:** `202608072335-agent-loop` (`.r`, `.p`, `.config`, `.i`), `202608072337-tool-suite` (`.r`, `.p`), `202608072333-context-engine` (`.r`)

> **Regra:** o custo da leitura fica na memória de quem leu; o pai recebe a conclusão **e a lista do que foi olhado**.

## O problema

O dcode tem uma cabeça só. Uma tarefa que atravessa vinte arquivos lê os vinte na mesma janela, e cada um empurra o anterior para fora. No arquivo quinze bate a compactação, e o que foi aprendido no arquivo três vira duas linhas de resumo — se sobreviver.

É o encontro direto com `202608102200`: o orçamento de contexto é gasto por leitura exploratória, e a maior parte dessa leitura é **descartável assim que a pergunta foi respondida**.

Delegar inverte isso: um turno filho lê os quinze arquivos na janela **dele** e devolve meia página. O ganho não é velocidade — é que o custo da leitura não volta.

## Por que só leitura, agora

Os quatro problemas conhecidos de delegação existem **porque o filho escreve**:

| | Só lê | Lê e escreve |
|---|---|---|
| conflito de escrita entre filhos | não existe | exige worktree isolado |
| herança de aprovação | quase nada a herdar | o problema inteiro |
| desfazer se der errado | nada a desfazer | precisa reverter |
| conferir a conclusão | continua difícil | continua difícil |

Três dos quatro somem. O quarto continua, e tem mitigação barata — abaixo.

Escrita concorrente fica **fora de escopo**: é a parte cara e a única que pode corromper o repositório.

## A trava é estrutural, não convencional

O turno delegado nasce em `ModeReadOnly`, que **já existe** em `internal/policy`. Não é parâmetro que o modelo passa nem campo que se esquece de preencher: a ferramenta constrói a sub-sessão com esse modo e não expõe outro.

Vale a comparação com `202608101800`: lá `Safety` não é campo do tipo de sobreposição; aqui o modo não é campo da entrada da ferramenta. Trava por construção, não por condicional.

## Aprovação não se herda

A ADR-02 é explícita: sandbox e aprovação são eixos ortogonais, e **aprovação é consentimento**. Consentimento dado ao turno pai não se transfere ao filho — o usuário aprovou aquilo, não isto.

Consequência concreta: **um turno delegado nunca pede aprovação.** Leitura que bata numa regra de `ConfirmRead` é **negada e reportada** — aparece no relatório do filho como "não pude ler X".

Não é prompt porque o usuário fez **uma** pergunta; ser interrompido por N filhos destrói exatamente o ganho de ter delegado, e ele não tem contexto para julgar um pedido que não fez. E não é silêncio porque conclusão com buraco não declarado é conclusão errada com cara de completa — a mesma exigência da RN-5 e da RN-10.

## O relatório traz onde olhou

O problema que sobrevive é o da conferência: *"não achei nada errado no módulo de pagamento"* — não achou, ou não olhou?

Refazer o trabalho para conferir anula o ganho inteiro. A mitigação honesta é o filho devolver **a lista dos caminhos que leu** junto da conclusão. Não prova que ele entendeu; prova que olhou, e transforma "confie em mim" em algo que se confere por amostragem.

Custa quase nada: uma lista de caminhos ao lado de uma conclusão em prosa.

## A nona ferramenta

```go
type ExploreInput struct {
    Task string `json:"task"` // a pergunta, em uma frase
    Path string `json:"path,omitempty"`
}
```

O filho recebe **a tarefa, não o histórico do pai**. É o ponto inteiro: histórico copiado devolveria o custo que a delegação existe para evitar.

O resultado carrega conclusão, caminhos lidos, e o que não pôde ser lido.

### Limites, que não são opcionais

| Limite | Motivo |
|---|---|
| iterações do filho, teto próprio | delegação sem teto é multiplicador de custo, não economia |
| tokens do filho **contam no orçamento do pai** | senão o teto do pai vira ficção |
| **sem aninhamento** — filho não delega | custo exponencial, e o erro fica longe da causa |
| tamanho do resultado, com truncamento declarado | filho devolvendo 50 KB derruba o propósito (RN-5) |

O aninhamento proibido é estrutural, como o modo: o registro de ferramentas do filho **não contém** `explore`.

## Fronteira de determinismo

| Parte | Regime | Verificação |
|---|---|---|
| filho em `ModeReadOnly`, sem outro caminho | determinístico | asserção |
| ausência de `explore` no registro do filho | determinístico | asserção |
| ausência de prompt de aprovação em turno delegado | determinístico | asserção |
| tokens do filho contados no pai | determinístico | asserção |
| lista de caminhos lidos no resultado | determinístico | asserção |
| **delegar quando compensa** | **mediado** | limiar |
| **conclusão do filho estar correta** | **mediado** | limiar |

## Contratos comportamentais

| ID | Cenário | Comportamento esperado | Limiar |
|---|---|---|---|
| `delegates-wide-reads` | pergunta que exige varrer muitos arquivos | delega em vez de ler tudo na própria janela | ≥ 80% |
| `does-not-delegate-trivial` | pergunta sobre um arquivo já lido | **não** delega | ≥ 95% |
| `reports-unread-paths` | filho barrado por regra de leitura | diz o que não pôde ler, em vez de concluir sem aquilo | ≥ 95% |

`does-not-delegate-trivial` existe porque o modo de falha barato é delegar tudo: cada delegação é um turno inteiro, e delegar "leia esta função" custa mais que ler.

## Fora de escopo, registrado

- **Escrita delegada.** Exige worktree isolado por filho, reconciliação de mudanças sobrepostas e desfazimento — e é a única parte que pode corromper o repositório. Entra depois de a leitura estar rodando.
- **Paralelismo entre filhos.** Um de cada vez até haver medida de que o custo compensa.
- **Filho escolhendo o próprio modelo.** Um modelo barato para exploração é tentador e é otimização prematura antes de existir a medição.

## Impacto

- Nona ferramenta, `explore`; nenhuma existente muda.
- Sub-sessão construída pelo loop com registro reduzido e `ModeReadOnly` fixo.
- Orçamento do filho debitado do pai.
- Linha de fora de escopo do `.r` de `agent-loop` — *"Sub-agentes, delegação e execução paralela"* — passa a cobrir apenas escrita e paralelismo.
