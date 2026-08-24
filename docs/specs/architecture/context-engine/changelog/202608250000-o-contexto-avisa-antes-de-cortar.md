# O contexto avisa antes de cortar

**Data:** 2026-08-25
**Specs afetadas:** `context-engine`, `client-server-protocol` (`context.band`),
`client-tui` (`.p`, seção 10)
**Fonte:** pedido de quem usa — "se chegar a 100% deveria avisar que está
compactando"

## O que mudou

A travessia de faixa é emitida ao **cliente**, e não só anunciada ao modelo. E o
corte passou a dizer quanto foi resumido e quanto ficou.

## Uma correção ao pedido

Foi pedido avisar "se chegar a 100%". Compactar a 100% é tarde: a requisição já
falhou. O gatilho de 80% da janela está certo e já existia.

O que faltava não era o gatilho — era a pessoa **ver chegando**. As faixas
60/80/92 já eram calculadas e anunciadas **ao modelo** há tempo. Ninguém as
anunciava a quem estava olhando, então o resumo aparecia como uma linha dizendo
que tinha acontecido: depois do fato, sem aviso e sem chance de terminar um
raciocínio antes.

## A fração é do orçamento

As faixas são frações do **orçamento** — o espaço até a compactação — e não da
janela. Essa decisão já estava tomada em `budget.go` e o comentário lá explica
por quê: contra a janela, duas das três faixas seriam inalcançáveis, porque o
corte acontece em 0.80 e 0.92 nunca chega.

Contra o orçamento, "80%" quer dizer oitenta por cento do caminho até o resumo,
que é a coisa sobre a qual dá para agir. O evento carrega essa fração, e a
frase na tela diz exatamente isso.

## O corte diz o tamanho

`session.compacted` ganhou `Messages` e `Kept`.

"O histórico anterior foi resumido" diz que algo aconteceu. "40 mensagens
anteriores foram resumidas; 12 mantidas" diz quanto — que é a diferença entre
um aviso e uma resposta.
