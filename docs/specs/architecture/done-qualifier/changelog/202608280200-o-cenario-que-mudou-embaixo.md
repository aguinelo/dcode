# O cenário que mudou embaixo do número

**Data:** 2026-08-28
**Specs afetadas:** `202608261730-done-qualifier` — a §8 troca três números
medidos por três números medidos. Sem mudança de código no produto.

> **Estado.** Dois de três contratos fecham. Os três números publicados na
> 0.13.0 foram substituídos, não acumulados.

## Por que substituir e não acumular

Uma taxa pertence a um cenário. Quando o cenário muda, o número velho descreve
algo que não existe mais — que é o mesmo defeito da linha do "Estado atual"
copiada da release anterior, com a diferença de estar nos dados em vez de na
contagem.

Duas coisas mudaram debaixo destes três contratos, e nenhuma delas é o modelo.

**O teto de rodadas era 12.** Um turno de qualificação lê a specificação, lê o
bastante da base para saber o que dá para rodar, e só então produz uma proposta.
É a mesma forma "explore e então aja" que já tinha levado `initRounds` e
`exploreThenActRounds` a 20, com a história escrita em ambos. Os qualificadores
ficaram em 12 por omissão, e um terço das falhas do `declares-regression` eram
execuções ainda lendo quando o arcabouço as cortou — a interrupção sendo medida,
não o modelo.

**O workspace compartilhado não compilava.** `internal/config/toml.go` chamava
`splitLines` e `cut`, e nenhum dos dois existia. Esse arquivo aparece no
transcript de quase todo cenário, e um modelo cuidadoso notava:

> *"Those helpers exist only as references — the file may not even compile.
> That's not directly the work for specs/median…"*

Um cenário ensinando todo modelo que o repositório está quebrado contamina muito
além do contrato onde apareceu. Agora `TestTheSharedWorkspaceCompiles` roda
`go build` offline sobre a árvore, e foi vista vermelha nas duas direções.

## O que ficou

| ID | 27 ago | 28 ago | alvo |
|---|---|---|---|
| `qualifier-fixes-broken` | 75% | **100%** de 20 | ≥ 85% ✅ |
| `qualifier-proposes-commands` | 96% | **98%** de 50 | ≥ 95% ✅ |
| `qualifier-declares-regression` | 85% | 80% de 20 | ≥ 90% |

O `declares-regression` **piorou**, e é o mais honesto dos três: com o teto
solto, nenhuma falha é mais decidida pelo arcabouço, e 80% é o que o modelo faz.

## A ablação, porque três mudanças juntas não atribuem causa

O `fixes-broken` saltou 25 pontos com três correções simultâneas — a P-5 do
piso, o teto e o workspace. Revertendo uma de cada vez, 20 execuções por
leitura:

| | P-5 | teto 20 | ws compila | medido |
|---|---|---|---|---|
| base | ✗ | ✗ | ✗ | 75% |
| tudo | ✓ | ✓ | ✓ | **100%** |
| sem P-5 | ✗ | ✓ | ✓ | 90% |
| teto 12 | ✓ | ✗ | ✓ | 95% |
| ws quebrado | ✓ | ✓ | ✗ | 95% |

Efeito **conjunto e aproximadamente aditivo**, sem causa dominante: ~10 pontos a
P-5, ~5 cada um dos outros dois. Cinco pontos é uma execução em vinte e não se
distingue de ruído; dez, sim.

## O que fica em aberto

`qualifier-declares-regression` não respondeu a nenhuma das três correções, e a
falha dele é a mesma que a P-5 ataca: termina sem propor. Isso significa que a
prática trata parte da forma e não a causa.

## Uma nota operacional que custou caro

Três execuções longas ficaram penduradas por horas e duas medições foram
perdidas — uma inteira para falha de DNS, outra a meio caminho. A causa não era
o arcabouço: **a máquina dormia**. O relógio monotônico do Go não avança durante
o sono, então `go test -timeout` nunca dispara e a execução fica de pé
indefinidamente enquanto o relatório não sai.

Uma suíte que custa dinheiro e pode ficar pendurada a noite inteira sem que nada
avise é uma suíte que alguém desliga — a mesma frase que justifica o build tag.
`make eval` sob `caffeinate` está no `ROADMAP.md`.
