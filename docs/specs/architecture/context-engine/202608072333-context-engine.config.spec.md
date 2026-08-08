# Config: Motor de Contexto

> Nenhuma variável de ambiente nova no código sem estar aqui.
> Precedência: flag > variável de ambiente > arquivo de config > default.

## 1. Compactação

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_COMPACT_AT` | fração | `0.80` | Fração da janela do modelo que dispara a compactação. Expresso em fração, não em contagem absoluta, porque o número certo muda por modelo e a fração não. Abaixo de `0.50` a compactação vira frequente e o cache deixa de render; acima de `0.95` não sobra folga para o turno em curso. |
| `DCODE_COMPACT_KEEP_TURNS` | inteiro | `4` | Turnos recentes preservados na íntegra além da regra obrigatória de RN-6. Reduzir aumenta a agressividade do corte; `0` mantém apenas o mínimo exigido pela regra de negócio. |
| `DCODE_COMPACT_ENABLED` | booleano | `true` | Desligar faz a sessão falhar ao atingir a janela em vez de compactar. Só use em depuração, quando compactar atrapalharia a reprodução de um bug. |

> `DCODE_COMPACT_AT` é fração da janela **do modelo em uso**, resolvida pelo adaptador de provider. O motor de contexto não conhece modelos.

## 2. Estimativa

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_TOKEN_ESTIMATE_RATIO` | float | `3.5` | Caracteres por token na heurística de `Estimate`. Valor conservador para texto misto de código e português. Aumentar subestima e arrisca estourar a janela; diminuir compacta cedo demais. |
| `DCODE_TOKEN_ESTIMATE_MARGIN` | fração | `0.10` | Margem de segurança somada à estimativa, absorvendo o erro da heurística. |

## 3. Constantes não configuráveis

Fixas em código, documentadas porque afetam comportamento observável.

| Constante | Valor | Motivo |
|---|---|---|
| Ordem das seções do prefixo | tabela 4 do `.p` | Alterar invalida cache de toda sessão viva; é mudança de contrato, não de config. |
| Fronteira de corte da compactação | turno completo | Corte no meio de um par assistant/tool produz histórico inválido para o provedor. |
| Preservação da última `RoleUser` | sempre | RN-6 é regra de negócio, não ajuste. |

## 4. Changelog

_Sem alterações desde a criação._
