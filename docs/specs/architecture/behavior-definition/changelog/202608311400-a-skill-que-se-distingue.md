# A skill carrega pelo que a distingue

**2026-08-31** — RN-7 ganha a regra do gatilho; `Match` passa a exigir um acerto
que discrimina, e a lista de palavras vazias deixa de ser só de inglês.

## O que estava errado

A lista de palavras vazias de `significantWords` tinha só inglês, num produto
cujo `LANGUAGE.md` declara duas línguas e cujo usuário escreve prompt em
português. `quando`, `projeto` e `estiver` passavam como palavras
significativas; `when` e `that` não passavam. A mesma frase era filtrada numa
língua e carregava skill na outra.

Com duas skills instaladas — uma sobre cortar versão, outra sobre publicar — a
tarefa `"quando o projeto estiver pronto me avisa"` carregava **as duas**, e
`"olha esse projeto e me diz quando a versão sobe"` também. Nenhuma das duas
frases é sobre release nem sobre deploy. O corpo inteiro de duas skills entrava
no turno por causa de palavra de ligação.

## Por que a lista de português não bastou

Tirar `quando` e `estiver` de circulação resolve a primeira frase e **não**
resolve a segunda: `projeto` e `versão` são palavras de conteúdo, aparecem nas
duas linhas de quando-usar, e dois acertos continuam sendo dois acertos.

O defeito não é que as palavras sejam comuns na língua. É que elas são comuns
**entre as skills do índice** — e uma palavra que as duas dizem não distingue
nenhuma das duas.

## A regra

Sem `triggers` explícito, o carregamento passa a exigir as duas coisas juntas:

1. dois acertos distintos, como já era;
2. pelo menos um acerto numa palavra que **nenhuma outra** skill do índice
   carrega.

A segunda condição é relativa ao índice, e é assim de propósito: discriminar é
uma relação com as alternativas, não uma propriedade da palavra. Uma skill
sozinha discrimina por tudo o que diz, que é a resposta certa — sem vizinha, não
há com o que se confundir.

Isso também protege o caso oposto, que uma regra de "descartar palavra
compartilhada" quebraria: `release-go` e `release-node` dizem as duas `cortar`,
`versão` e `nova`, e continuam alcançáveis porque cada uma ainda tem `golang` e
`typescript`.

## O que ficou de fora

**Uma acerto discriminante só não basta.** A regra de dois acertos é anterior a
esta correção e continua valendo: ela é o que impede que uma tarefa que apenas
menciona um assunto arraste o corpo inteiro. Skill que queira disparar em um
termo só declara `triggers`, que é casado como frase e não passa por este
caminho.

**A lista por língua cobre só as línguas que estão nela.** Está escrito na RN-7
em vez de combinado, e a saída para qualquer outra língua é a mesma:
`triggers`.

## Invariantes

- `TestMatchDoesNotFireOnPortugueseFillerWords` — as duas frases do defeito,
  reproduzidas antes da correção.
- `TestMatchNeedsAWordThatBelongsToThisSkillAlone` — o acerto que discrimina
  seleciona, e um acerto só não carrega mesmo discriminando.
- `TestSkillsInTheSameDomainStayReachable` — vizinhas de domínio continuam
  alcançáveis.
