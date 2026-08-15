# A cópia é dona do teclado enquanto está aberta

**Data:** 2026-08-14

## O que mudou

O bloco de `onKey` que trata as teclas do modo de cópia — `up`/`k`, `down`/`j`,
`y`/`enter`, `esc`/`q`/`ctrl+c` — estava **dentro** do guarda do menu de
autocompletar:

```go
if len(p.model.Completions) > 0 {
    if p.model.Copy.Active { ... }   // só rodava com o menu aberto
    ...
}
```

O menu só aparece depois que algo é digitado, e o modo de cópia só abre com a
linha de entrada **vazia**. As duas condições nunca são verdadeiras ao mesmo
tempo, então nenhuma dessas teclas jamais foi executada.

Na prática: `v` abria a cópia, e a partir daí as setas rolavam o fluxo, `y`
digitava a letra `y` na linha de entrada, e sair só acontecia pelo que o `esc`
geral fazia. O modo era decorativo.

O bloco foi movido para **antes** do guarda do menu, logo depois do modal de
aprovação. Nenhuma linha da lógica mudou — só o nível em que ela vive.

## Por que passou

O comentário acima do bloco afirmava a invariante: *"Copy mode owns the keyboard
while it is open, and nothing else does."* Nenhum teste a cobrava, e a spec não
a declarava. É a forma que este projeto não para de encontrar: **algo declarado
que um lado lê e nenhum lado escreve** — desta vez a afirmação estava no
comentário e a execução não estava em lugar nenhum.

Encontrado por cobertura: `LeaveCopy` e `renderedStream` apareciam a 0%, e a
única explicação para uma função de produto nunca ser chamada era não haver
caminho até ela.

## O que passou a valer

Duas invariantes novas em `## 10. Invariantes verificáveis`, ambas cobradas por
teste:

- Enquanto a cópia está aberta ela é dona do teclado: nenhuma tecla chega ao
  fluxo nem à linha de entrada.
- A cópia sai por `Esc`, `q` e `Ctrl+C`, e copiar fecha o modo dizendo o que
  houve.
