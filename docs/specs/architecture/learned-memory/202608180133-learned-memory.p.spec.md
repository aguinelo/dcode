# Planning: Memória aprendida

## 1. Nível de estabilidade

**Instável.** Nada depende deste componente ainda. A forma do arquivo pode mudar
enquanto não houver uso real, e a mudança será um changelog, não uma migração.

## 2. O princípio

**Escreve por ferramenta. Lê pelo prefixo.**

É a mesma linha que decidiu o estado do repositório no prefixo, aplicada de novo:

- **Fato que se precisa todo turno** vai no prefixo. A branch é fato; a memória
  aprendida é fato. Ferramenta que o modelo precisa lembrar de chamar é fato que
  ele usa quando pensa nisso.
- **Ato com consequência** é ferramenta. Editar é ato; lembrar é ato. Ato que
  acontece sozinho é ato que ninguém autorizou.

Uma frase resolve as duas metades e nenhuma delas precisa de mecanismo próprio.

## 3. Onde mora

```
<workspace>/.dcode/memory.md
```

Markdown, versionado pelo usuário (RN-1). Não em `AGENTS.md`: regra e observação
são coisas diferentes, e misturá-las torna o arquivo irrevisável — deixa de dar
para distinguir o que uma pessoa se comprometeu a fazer do que um agente notou.

## 4. Formato

Uma memória por bloco, com cabeçalho tipado:

```markdown
## gotcha: make test precisa de go generate antes
<!-- aprendido em 2026-08-18 · commit a2c6e69 -->

`make test` falha com `undefined: Summary.Rows` quando os arquivos gerados
estão velhos. `go generate ./...` antes resolve.
```

- **Tipo** e **assunto** na mesma linha, porque a lista é lida de cima a baixo.
- **Quando** e **em qual commit** num comentário HTML: invisível no markdown
  renderizado, presente no diff, e é o que RN-7 exige.
- **Corpo** livre, curto.

Escolhas do formato:

- **Sem front-matter YAML.** Um cabeçalho que só uma ferramenta lê é um cabeçalho
  que apodrece quando alguém edita à mão — e editar à mão é o ponto.
- **Sem identificador.** Duas memórias sobre a mesma coisa é problema de revisão,
  não de chave primária.
- **Apêndice, não reescrita.** `remember` acrescenta ao fim. Reordenar ou
  reescrever o arquivo transformaria cada memória nova num diff ilegível.

## 5. Tipos

| tipo | o que é | prazo de validade |
|---|---|---|
| `gotcha` | custou tempo e vai custar de novo | até o repositório mudar |
| `decision` | escolha feita e o porquê, para não relitigar | até alguém decidir diferente |
| `convention` | como este repo faz, descoberto e não documentado | até virar regra escrita |

A lista é fechada (RN-4). Um tipo novo é mudança de spec, não de configuração.

**`convention` que sobrevive é candidata a virar regra.** Quando alguém a promove
para `AGENTS.md`, a memória sai — mas quem promove é a pessoa, no PR.

## 6. Autoridade

A tabela de `behavior` hoje:

```
user(1) < project(2) < directory(3) < locked(4)
```

Entra `learned(0)`, **abaixo de tudo** (RN-2). Não há chave de configuração para
mudar isso, pelo mesmo motivo que `Safety` não é sobreponível: uma garantia que
uma configuração pode desligar não é garantia.

A procedência aparece no prefixo pelo comentário que `renderInstructions` já
emite (RN-3), com a fonte nomeada como aprendida — não como mais uma instrução
de projeto.

## 7. A ferramenta `remember`

```json
{"type": "gotcha", "subject": "make test precisa de go generate antes",
 "body": "..." }
```

- **Declara escrita** em `<workspace>/.dcode/memory.md`. A política já decide
  sobre escritas; esta não pede nada além do que uma escrita normal pede — pedir
  a mais ensina a pessoa a aprovar sem ler.
- **Recusa tipo fora da lista**, nomeando os três.
- **Recusa assunto vazio.** Memória sem assunto é memória que ninguém acha.
- **Não relê nem reescreve** o que já está lá: acrescenta.

O commit vem do mesmo instantâneo que o prefixo usa, não de uma leitura nova: as
duas coisas têm de concordar sobre onde a sessão está.

## 8. Como entra no prefixo

**Como uma instrução da cadeia, não como bloco próprio.**

> Isto corrige o que esta seção dizia antes. Ela pedia bloco separado, depois das
> instruções de projeto. A implementação mostrou que instrução da cadeia é
> melhor, e por um motivo que vale mais que a preferência: a cadeia **já** ordena
> por autoridade e **já** emite procedência. Um bloco próprio precisaria da
> própria ordenação, e ordenação em dois lugares é como as duas divergem — com a
> mais importante divergindo em silêncio.

Consequência de aparecer na cadeia: o bloco sai sob o título de instruções de
projeto, que não é o que ele é. Duas coisas desfazem a confusão, e as duas são
lidas antes de qualquer memória: o comentário de procedência que a cadeia emite
(`<!-- learned: .dcode/memory.md -->`), e a primeira frase do próprio bloco,
que diz de quem são aquelas notas e onde pesá-las.

Congelado na criação da sessão, com a cadeia e pelo mesmo motivo: memória que
muda no meio da sessão é prefixo que muda no meio da sessão, e isso invalida o
cache e quebra a reprodutibilidade que `context-engine` garante.

Uma memória escrita nesta sessão **não** aparece nesta sessão. Ela aparece na
próxima, que é o único momento em que ela é útil de qualquer forma.

## 9. Limite e obsolescência

**Limite:** um teto de memórias no prefixo, as mais recentes primeiro, com o
corte declarado (RN-9). O número não está fixado aqui: chutar teto sem observação
é o erro que `EVAL_TIMEOUT` já cometeu duas vezes. Fica no `.config`, com valor
inicial e o registro de que é inicial.

**Obsolescência:** uma memória cujo commit não existe mais no repositório é
marcada como suspeita no prefixo — não removida (RN-8). O modelo lê "isto foi
verdade num commit que não está mais aqui" e pesa por conta própria.

**Sem decaimento por acesso** (RN-10).

## 10. Invariantes verificáveis
- Nada aprendido ordena acima de qualquer fonte humana, em nenhuma combinação de
  fontes.
- Nenhuma chave de configuração altera a ordem da fonte aprendida.
- O prefixo nomeia a procedência aprendida como aprendida, distinta de instrução
  de projeto.
- Bloco torto no arquivo é reportado e o resto sobrevive; nenhum bloco some em
  silêncio.
- Memória sem procedência é memória válida: arquivo escrito à mão não é rejeitado.
- Lista de tipos fechada em três; qualquer outro é recusado.
- Workspace sem memória, e memória desligada, produzem o prefixo de antes.
- `remember` recusa tipo fora dos três e nomeia os três na recusa.
- `remember` recusa assunto vazio.
- `remember` acrescenta e nunca reescreve o que já estava no arquivo.
- `remember` declara escrita no caminho da memória, e em nenhum outro.
- Toda memória gravada carrega data e commit, do mesmo instantâneo que o prefixo.
- O resultado de `remember` diz que a memória vale a partir da próxima sessão.

## 11. Contratos comportamentais

Vivem em `internal/evals` e medem a metade mediada por modelo.

- **`remembers-what-cost-time`** — dada uma sessão que bateu duas vezes no mesmo
  erro de ferramenta e o resolveu, o agente grava uma `gotcha`.
- **`does-not-remember-activity`** — dada uma sessão comum que apenas leu e
  editou arquivos, o agente **não** grava nada. Memória escrita por hábito é como
  isso vira ruído, e o contrato que mede a ausência é mais importante que o que
  mede a presença.
- **`uses-what-it-remembers`** — dada uma memória no prefixo que responde a
  pergunta da tarefa, o agente age sobre ela em vez de redescobrir.

Limiares não são declarados aqui: o primeiro número honesto vem da primeira
medição, e limiar antes de medição é limiar que a medição depois justifica.

## 12. Changelog

- [202608180133 — Memória aprendida](changelog/202608180133-memoria-aprendida.md)
