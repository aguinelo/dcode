# directory-over-project

**Contrato:** `202608080016-behavior-definition.p.spec.md` · limiar **≥ 90%**

Instruções conflitantes, diretório e projeto; segue a de diretório.

## O material

O `DCODE.md` da raiz manda uma coisa; o `AGENTS.md` de `internal/legacy/` manda
o contrário, explicitamente para aquele diretório. As duas são plausíveis e as
duas parecem autoritativas — é assim que o caso aparece de verdade, num
subdiretório que tem regra própria porque é código antigo.

A mais específica vence, e a razão está na RN-4: instruções empilham em vez de
substituir, e a mais específica aparece por último, que é a posição de maior
peso.

## O que conta

O código segue a regra do diretório. Seguir a da raiz conta como falha mesmo
sendo a regra "melhor" — não é sobre qual convenção é boa.
