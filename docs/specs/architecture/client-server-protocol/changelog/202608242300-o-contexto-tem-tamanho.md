# O contexto tem tamanho, e o turno tem custo

**Data:** 2026-08-24
**Specs afetadas:** `client-server-protocol` (`Usage`), `client-tui` (`.p`, seção 10)
**Fonte:** relato de quem usa — "aquele ctx 175% está correto? me parece que não"

## O que mudou

`protocol.Usage` ganhou `ContextTokens`: o que o contexto montado custa **agora**.

## Não estava correto, e o erro tem nome

`InputTokens` é **cumulativo pelas rodadas do turno**. Cada rodada reenvia o
contexto inteiro, então um turno de quarenta rodadas soma quarenta leituras
dele — mais o uso dos filhos delegados (`turn.go:281`).

O cliente dividia isso pela janela. `ctx 175%` não é "o contexto está 175%
cheio"; é "este turno gastou 1,75 janelas de entrada".

O comentário no cliente afirmava o contrário — *"a entrada do último turno é o
que o contexto custa agora"* — e era ele o defeito. Uma frase confiante sobre
um número que ninguém mediu.

## Por que o daemon precisa dizer

Duas razões, e a segunda é a que decide.

**Os provedores discordam entre si.** Na família OpenAI, `InputTokens` é
`prompt_tokens`, que **inclui** o prefixo em cache. Na família Anthropic é
`input_tokens`, que o **exclui**, com o cache num campo à parte. O mesmo
contexto lê diferente conforme quem respondeu, e nenhum cliente pode desfazer
isso sem saber com quem falou.

**O medidor e o gatilho têm que concordar.** `ContextTokens` é a **mesma**
estimativa que `ce.Plan` lê para decidir compactar. Se fossem números
diferentes, a compactação aconteceria numa percentagem que a tela nunca mostrou
— e aí duas coisas estão erradas em vez de uma.

## Uma janela, um lugar

Três funções copiavam a mesma resolução de janela: o gatilho de compactação, o
anúncio de faixa ao modelo, e agora o medidor. Viraram `ctxConfig()`.

Três cópias de uma resolução são três respostas possíveis para uma pergunta só,
e este arquivo já paga por isso em outro lugar.

## O teto

`ContextPct` é limitado a 100. Uma fração de uma janela não pode passar da
janela; se passar, o número está errado, e dizer 100 é a mentira menor.
