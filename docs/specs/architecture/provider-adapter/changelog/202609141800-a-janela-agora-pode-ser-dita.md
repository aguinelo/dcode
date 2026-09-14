# A janela agora pode ser dita

**Data:** 2026-09-14
**Specs afetadas:** `202608072334-provider-adapter` (`.config`, seção 1)
**Fonte:** pedido do usuário — apontar o dcode para um servidor local
(LM Studio, API compatível com OpenAI, `qwen3.5-9b`, ~32k de contexto) usando
`--family generic`.

## O que mudou

Nova variável, `DCODE_WINDOW` / `model.window`, inteira, vazia por padrão.
Quando presente e maior que zero, sobrescreve o que a família devolveria em
`Window(model)`. `internal/app/app.go`, no único lugar que já chamava
`p.Window(opts.Model)` para montar `ctxCfg.Window`.

## Por que

`Generic.Window` (`internal/provider/family_generic.go`) devolve **128.000**
para qualquer modelo, sempre — é um palpite conservador, e o próprio `.r` desta
spec já registrava que "a janela e as imagens também são chute" para essa
família. Contra um endpoint desconhecido o chute é a única opção. Contra um
endpoint que o usuário **conhece** — ele digitou a URL, ele sabe o contexto do
modelo que está rodando — insistir no chute joga fora uma informação que já
está na mão de quem está configurando.

O erro concreto: um modelo local de ~32k tratado como se tivesse 128k. O
medidor de orçamento de contexto (`context_budget`, `internal/behavior`) dispara
avisos de compactação em 60/80/92% de uma janela que não existe — quando os
avisos chegariam, o provedor já teria recusado a requisição por estourar o
contexto real, várias dezenas de milhares de tokens antes. A dcode não tinha
como saber que estava errado; agora tem como ser corrigida.

## Por que não uma família nova

Uma família nova (`qwen3.5-9b`, digamos) declararia `Window` correto — mas
também herdaria a mesma ressalva que `generic` já carrega: nenhum contrato
comportamental foi medido contra ela. Criar família por endpoint local é
prometer uma garantia (limiares medidos) que não existe, só para resolver um
número. `DCODE_WINDOW` resolve exatamente o número, sem prometer o resto.

## O que não mudou

`Family.Window(model string) (int, error)` continua exatamente como está —
nenhuma família precisou mudar, e a interface declarada em `.p` (seção 2) não
foi tocada. O override vive na composição, em `internal/app`, o mesmo andar
onde `DCODE_BASE_URL` já troca o endpoint sem trocar a família (RN-1). Zero ou
ausente não desliga nada: a família continua respondendo por padrão.
