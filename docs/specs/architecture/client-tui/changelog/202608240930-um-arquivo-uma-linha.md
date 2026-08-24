# Um arquivo, uma linha

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** sessão real reproduzida pelo redutor — `1a030dd642cd50b9f68`

## O que mudou

Um alvo de ferramenta é normalizado contra o workspace no momento em que entra
no modelo. Uma causa, três sintomas resolvidos.

## Três sintomas, uma causa

A coluna lateral, na sessão real, mostrava:

```
ARQUIVOS 15 tocados
▾ /Users/aguinelo/workspac
  ✓ DCODE.md
✓ DCODE.md            +79
```

O mesmo arquivo, duas vezes. Uma ferramenta o nomeou `DCODE.md` e a seguinte
`/Users/…/craw/DCODE.md`, porque o alvo é o que o modelo escreveu, e o modelo
não é obrigado a ser consistente consigo mesmo.

Duas grafias são duas linhas, dois contadores e um cabeçalho afirmando quinze
arquivos quando foram onze. E a linha de pasta, carregando o caminho absoluto
como rótulo, comia a coluna inteira sem identificar nada.

## Onde a normalização mora

Em `Apply`, onde o alvo entra no modelo — **não** na coluna que percebeu o
problema.

A coluna era um de três leitores: a linha de chamada também desenhava o caminho
absoluto, e o próximo leitor teria de aprender a regra de novo. Normalizar na
entrada faz `Entry.Target` ser sempre a grafia curta, e todos os leitores
herdam.

O workspace vem de `session.created`, então faz parte do fluxo de eventos: o
mesmo log reproduzido produz as mesmas grafias. É o que mantém "mesma sessão
reaberta, mesma árvore" verdadeiro por construção, e não por cuidado — a mesma
razão pela qual a árvore é derivada em vez de guardada.

## Prefixo, não `filepath.Rel`

`Rel` responde para qualquer par de caminhos, inclusive saindo do workspace com
`../..`. Um arquivo acima do workspace é melhor mostrado pelo caminho absoluto
que o encontra do que por uma escada que ninguém consegue abrir. `/tmp/…`
continua `/tmp/…`.

## O que o teste afirma

Pelo `Apply`, não pela função auxiliar. O workspace de que isto depende chega
num evento; um teste que passasse o workspace na mão passaria mesmo que nada
lesse `session.created` — que é justamente a metade que pode quebrar.
