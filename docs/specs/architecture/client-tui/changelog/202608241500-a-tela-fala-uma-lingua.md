# A tela fala uma língua

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** quadro real renderizado em pt-BR

## O que mudou

Nove strings em inglês cravadas no código de desenho foram para o catálogo — o
**modal de aprovação inteiro** entre elas.

## A tela que mais importa estava em inglês

```
┌─ Approval needed ──────────────────────────┐
│                                            │
│  bash crosses: network                     │
│                                            │
│  Commands in this project may reach the    │
│  network.                                  │
│                                            │
│  [d] deny   [a] allow   [A] whole session  │
│  Enter denies.                             │
└────────────────────────────────────────────┘
```

Numa interface em português. É a única tela que pergunta se uma fronteira pode
ser cruzada, e **consentimento dado a uma frase que a pessoa não conseguiu ler
não é consentimento**.

Junto foram o cabeçalho `PLAN` do painel e o rodapé `6 of 7 (1 blocked)`.

## A guarda existente não podia pegar

`TestEveryDeclaredLanguageCoversEveryString` pergunta se toda string declarada
tem tradução. Ela **não tem como** perguntar se o renderizador usa as strings —
então um literal no código de desenho é invisível para ela por construção.

É a mesma forma do defeito de ASCII de hoje cedo: uma guarda que pergunta sobre
um conjunto conhecido, e o modal fora do conjunto desde o dia em que ela foi
escrita. **Duas vezes no mesmo dia, na mesma tela.**

## A nova guarda deriva o que proíbe

Toda palavra que o catálogo inglês usa e o português não, com cinco letras ou
mais. Esse conjunto **cresce sozinho com o catálogo**: string acrescentada
amanhã é conferida amanhã.

O modelo do teste é escrito em português e não carrega nenhuma dessas palavras,
então o que for encontrado veio do layout.

## O que isso não pega

Literal em inglês cujo conceito ainda não existe no catálogo — não há com o que
comparar. A guarda pega o literal a partir do momento em que alguém escreve a
tradução, que é quando o defeito passa a ter conserto conhecido.

Dito aqui porque uma guarda cujo limite não está escrito é uma guarda em que se
confia demais.
