# A barra inferior

**Data:** 2026-08-12
**Specs afetadas:** `202608081250-client-tui` (`.r`, `.p`, `.i`)
**Origem:** design `dcode ADE TUI`, seção 02 — "Anatomia da barra inferior"

> **Regra:** a tela ganha uma região permanente na última linha: onde você está,
> o que mudou, e o que espera por você. Uma linha, sempre — nunca duas.

## Por que uma região nova

O que a barra carrega não cabe no status do topo, e não por falta de espaço.
São fatos de natureza diferente: o status do topo descreve **a sessão** — modelo,
modo de sandbox, selo de verificação. A barra descreve **onde o trabalho está
acontecendo**, e continua verdadeira independentemente do que o fluxo mostra.

Com um workspace isso é quase gratuito. O design que a originou tem quarenta
worktrees e cinco agentes por worktree, e ali "onde estou" deixa de ser
contexto e vira a informação mais cara da tela.

## Uma linha, sempre

A barra cede espaço **derrubando segmentos**, nunca quebrando em duas linhas.
Uma segunda linha tiraria uma linha do fluxo exatamente no momento em que o
terminal já está estreito demais — o pior momento possível para cobrar isso.

Ordem de queda, da direita para a esquerda: o que a pessoa consegue reconstruir
em outro lugar sai primeiro. O diff da vez está no próprio fluxo; **onde você
está** e **o que está travado esperando você** não estão em lugar nenhum.

Esgotados os opcionais, o nome do worktree é **cortado**, não removido: uma
resposta truncada para "onde estou" ainda responde.

## O que o âmbar passa a significar

Estrutura e foco, e nada mais: worktree ativo e pendência, ambos em fundo âmbar
sólido com texto quase preto. Dois segmentos sólidos lado a lado diriam que tudo
é estrutura, então são os únicos.

## Segmento sem dado não desenha

A regra vem do próprio design para a pendência — *"some por completo quando
zero, sem badge vazio"* — e vale para todos. Um badge que mostra zero é um badge
que as pessoas aprendem a não ler, e aí ele ainda está lá no dia em que há algo.

Por isso a barra hoje mostra três segmentos e não oito: worktree, diff acumulado
e pendência são os que o dcode sabe responder. Frota, cpu, quota por conta,
disco e conexão exigem um produto que ainda não existe — e um slot vazio
descreveria a barra, não a sessão.

## O que fica de fora, e por quê

O design `dcode ADE TUI` descreve um ambiente de desenvolvimento com frota de
agentes em worktrees isolados, CLIs de terceiros lado a lado, terminais
embutidos, abas, splits, fan-out com merge, busca unificada e SSH remoto.

Isso não é uma revisão visual do dcode: é outro produto, com outra arquitetura.
A barra inferior entra agora porque é a única peça que o design especifica por
completo — anatomia, ordem de queda e até a assinatura da função — e que o dcode
consegue sustentar com o que já sabe.
