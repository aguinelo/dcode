# Config: Motor de Contexto

> Nenhuma variável de ambiente nova no código sem estar aqui.
> Precedência: flag > variável de ambiente > arquivo de config > default.

## 1. Compactação

| Variável | Tipo | Default | Uso |
|---|---|---|---|

> `DCODE_COMPACT_AT` é fração da janela **do modelo em uso**, resolvida pelo adaptador de provider. O motor de contexto não conhece modelos.

## 2. Estimativa

| Variável | Tipo | Default | Uso |
|---|---|---|---|

## 2.1 Orçamento realimentado (RN-6.1)

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_BUDGET_NOTICE` | booleano | `true` | Liga o aviso de ocupação da janela ao modelo. Desligar devolve o comportamento anterior: o único sinal passa a ser o aviso **pós**-compactação. |

> **Correção.** Esta linha dizia "frações da janela" e, ao mesmo tempo, que todas deviam ser menores que `DCODE_COMPACT_AT`. As duas coisas não cabem juntas: `DCODE_COMPACT_AT` é `0.80` por default, então `0.80` e `0.92` da janela são inalcançáveis — o contexto é cortado antes de chegar lá.
>
> Lidas como fração do **orçamento**, as duas afirmações valem e o sentido melhora. "Você está em 92%" passa a dizer 92% do que se tem antes de a memória ser cortada, que é exatamente aquilo sobre o que o modelo consegue agir. Contra a janela, seria um número sobre um limite que nunca chega.
>
> As faixas acompanham o gatilho: baixar `DCODE_COMPACT_AT` para `0.5` encolhe o orçamento e move as três junto. Uma faixa presa à janela dispararia depois do corte.

> **Não há chave para emitir por nível.** Repetir o aviso enquanto a fração estiver acima do limiar custa em todo turno e produz habituação: aviso que aparece sempre deixa de ser lido. A emissão é por borda, e rearma na compactação.

## 3. Constantes não configuráveis

Fixas em código, documentadas porque afetam comportamento observável.

| Constante | Valor | Motivo |
|---|---|---|
| Ordem das seções do prefixo | tabela 4 do `.p` | Alterar invalida cache de toda sessão viva; é mudança de contrato, não de config. |
| Fronteira de corte da compactação | turno completo | Corte no meio de um par assistant/tool produz histórico inválido para o provedor. |
| Preservação da última `RoleUser` | sempre | RN-6 é regra de negócio, não ajuste. |
| Fração e faixa fora do prefixo | sempre | RN-2; número volátil no prefixo invalida o cache em todo turno. |
| Aviso de orçamento por borda | sempre | RN-6.1; por nível custa sempre e produz habituação. |
| Limiar mais alto abaixo de `CompactAt` | sempre | RN-6.1; aviso simultâneo ao corte é aviso inútil. |

## 4. Changelog

- [202608102200 — Orçamento de contexto realimentado](changelog/202608102200-orcamento-de-contexto-realimentado.md)
