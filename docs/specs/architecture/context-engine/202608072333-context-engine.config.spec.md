# Config: Motor de Contexto

> Nenhuma variável de ambiente nova no código sem estar aqui.
> Precedência: flag > variável de ambiente > arquivo de config > default.

## 1. Compactação

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_COMPACTION_ENABLED` | booleano | `true` | Liga o gatilho automático (80% do orçamento). Desligado, nada dispara sozinho — mas `/compact` e a recuperação forçada (provedor recusando por contexto grande) continuam funcionando: a pessoa pedindo direto, ou o provedor recusando, nunca foi o que este interruptor decide. `Engine.Compact` nem lê este campo. |

> **Não existe `DCODE_COMPACT_AT`.** O gatilho é a constante `CompactAt = 0.80` de `DefaultConfig` (`internal/contextengine/types.go`), fração da janela **do modelo em uso** — a janela vem do adaptador de provider, ou de `DCODE_WINDOW`; o motor de contexto não conhece modelos. Ver a seção 3. Mudar o gatilho é mudança de código, com o teste `TestEveryBandLandsBelowTheCompactionTrigger` junto.

## 2. Estimativa

Nenhuma chave. A heurística — caracteres por token, margem, custo de página de PDF — é constante (seção 3): a estimativa precisa ser a mesma para a mesma entrada em qualquer máquina, ou o golden test da compactação fica instável.

## 2.1 Orçamento realimentado (RN-6.1)

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_BUDGET_NOTICE` | booleano | `true` | Liga o aviso de ocupação da janela ao modelo. Desligar devolve o comportamento anterior: o único sinal passa a ser o aviso **pós**-compactação. |

> **Correção.** Esta linha dizia "frações da janela" e, ao mesmo tempo, que todas deviam ser menores que o gatilho de compactação. As duas coisas não cabem juntas: `CompactAt` é `0.80`, então `0.80` e `0.92` da janela são inalcançáveis — o contexto é cortado antes de chegar lá.
>
> Lidas como fração do **orçamento**, as duas afirmações valem e o sentido melhora. "Você está em 92%" passa a dizer 92% do que se tem antes de a memória ser cortada, que é exatamente aquilo sobre o que o modelo consegue agir. Contra a janela, seria um número sobre um limite que nunca chega.
>
> As faixas acompanham o gatilho: baixar `CompactAt` para `0.5` encolhe o orçamento e move as três junto. Uma faixa presa à janela dispararia depois do corte.

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
| `CompactAt` | `0.80` da janela | Gatilho automático. Constante, e não chave: as três faixas do aviso de orçamento são fração dele, e um valor que se move por configuração move junto o que o modelo é avisado. |
| `KeepTurns` | `4` | Turnos recentes que sobrevivem à compactação além da tarefa corrente (RN-6). Um dos dois pisos da cauda. |
| `KeepFraction` | `0.30` da janela | O outro piso, em tokens: quatro turnos curtos protegem quase nada, quatro longos deixam quase nada a compactar. Vence o piso que proteger mais. |
| `CharsPerToken` | `3.5` | Heurística da estimativa; sem tokenizador de rede. |
| `Margin` | `0.10` | Somada a toda estimativa. Zero é tratado como ausente: sem margem, o erro da heurística vira estouro da janela. |
| `charsPerPDFPage` | `12000` | Custo de uma página de PDF na estimativa — o topo da faixa de 1.500 a 3.000 tokens por página, a 4 caracteres por token. Estimar longo compacta cedo; estimar curto perde o turno. |
| Páginas de PDF | `PDFPages`, mínimo `1` | PDF que esconde os objetos de página conta como uma página, nunca como grátis. |
| Marcador de documento de texto | `.p`, seção 3.1 | `<document name="…" media_type="…" boundary="B">` … `</document boundary="B">`, atributos entre aspas e escapados em HTML, `B` = 24 hex do SHA-256 do conteúdo. Fronteira derivada do conteúdo: o arquivo fica intacto e nada nele fecha o bloco. |
| Resumo que falha não compacta | sempre | Erro, vazio ou corte no teto de saída deixam o histórico inteiro e emitem `session.error` `compaction_failed` (`agent-loop`, `202610161200`). |

## 4. Changelog

- [202608102200 — Orçamento de contexto realimentado](changelog/202608102200-orcamento-de-contexto-realimentado.md)
- [202608110900 — Chaves declaradas e não lidas saem da spec](changelog/202608110900-chaves-declaradas-e-nao-lidas.md)
- [202608250000 — O contexto avisa antes de cortar](changelog/202608250000-o-contexto-avisa-antes-de-cortar.md)
- [202608250100 — A cauda tem dois pisos](changelog/202608250100-a-cauda-tem-dois-pisos.md)
- [202609151700 — Compactar por pedido](changelog/202609151700-compactar-por-pedido.md)
- [202610180100 — A spec segue o código](changelog/202610180100-a-spec-segue-o-codigo.md)
