# Verificação impossível não cancela o trabalho

**Data:** 2026-08-28
**Specs afetadas:** `202608262200-working-defaults` — nasce a P-5 e a RN-9; a
tabela da §6 troca um número medido por outro medido.

> **Estado.** A prática está no piso e medida. Vale ~10 pontos no contrato onde
> a falha era mais nítida, e nada no contrato onde a mesma falha persiste.

## A falha, vista em três contratos que não se conhecem

Três contratos comportamentais de duas famílias, com cenários sem nada em
comum, mostraram a mesma coisa: o turno lê tudo, raciocina certo, **e termina
sem fazer o ato**.

| contrato | o que ficou faltando |
|---|---|
| `qualifier-fixes-broken` | leu a spec e o código, **não propôs** |
| `qualifier-declares-regression` | idem |
| `floor-does-not-ask` | leu o código, **não editou** |

E sempre logo depois de anunciar uma verificação que não podia fazer:

> *"Let me verify whether `gotestsum` or `npm` is available in this project
> before proposing."*
> *"Now let me verify the test actually catches the bug it claims to."*

## A lacuna estava no `Style`, e é de meia regra

A doutrina já dizia:

> *"when you could not verify something, say that instead of claiming success"*

Ela ensina **o que dizer** e nunca diz que o trabalho continua devido. Lida por
um modelo que acabou de descobrir que não consegue conferir nada, é licença para
parar.

A P-5 fecha o caso, e é curta de propósito:

> *"A check you cannot run does not cancel the work. Do the work, say in one
> line what you could not check, and end the turn there — never end it having
> checked nothing and done nothing."*

Ela entra no **piso** e não na instrução do turno qualificador porque a falha
aparece fora dele — `floor-does-not-ask` não é um turno de qualificação. E no
piso ela é sobreponível pelo usuário e pelo projeto, que é onde um default deve
morar.

## Quanto ela vale, medido

A P-5 entrou junto de duas outras correções, e três mudanças simultâneas não
atribuem causa. A ablação separou, revertendo uma de cada vez contra
`qualifier-fixes-broken` — 20 execuções por leitura:

| | P-5 | teto 20 | workspace compila | medido |
|---|---|---|---|---|
| base (0.13.0) | ✗ | ✗ | ✗ | 75% |
| tudo | ✓ | ✓ | ✓ | **100%** |
| sem a P-5 | ✗ | ✓ | ✓ | 90% |
| teto de volta a 12 | ✓ | ✗ | ✓ | 95% |
| workspace quebrado | ✓ | ✓ | ✗ | 95% |

**Nenhuma das três sozinha explica o salto.** As contribuições somam 20 dos 25
pontos, e a P-5 é a maior — ~10 pontos, a única acima do ruído de uma execução
(que a 20 vale 5 pontos).

## Por que ela fica

A RN-8 manda tirar prática que *"nunca reprovou nenhum contrato em três
medições"*. Esta reprovou: sem ela, duas execuções em vinte falham que com ela
passam.

É evidência mais fraca do que se gostaria — uma ablação, n=20, efeito da mesma
ordem de grandeza que os outros dois. Fica registrado como **contribui, medido
uma vez**, e não como resolvido.

## O que ela não resolveu

`qualifier-declares-regression` não se moveu com nada disso: 85% antes, 80%
depois — e o 80% é o número honesto, porque a primeira leitura tinha um terço
das falhas decididas pelo teto. A mesma forma de falha continua lá.

Isso diz que a P-5 trata parte do problema e não a raiz. O que resta é
provavelmente sobre **o que o turno faz quando a verificação é impossível E o
trabalho é ambíguo** — e isso não é uma frase de doutrina, é desenho.
