# A coluna lateral mostra o que o turno tocou

**Data:** 2026-08-21
**Specs afetadas:** `202608081250-client-tui` (`.p`, seções 7 e 10)
**Fonte:** `refs/design/HANDOFF.md` (v5, §1)

## O que mudou

Uma coluna à esquerda lista os arquivos que o turno tocou, com o estado de cada
um e a contagem de linhas de quem terminou. `^B` dobra e expande.

`clamp(20, w/5, 30)` de largura, some abaixo de 100 colunas, e a escolha
explícita vence nos dois sentidos — exatamente as maneiras que o painel de plano
já tinha. Responder a mesma pergunta de dois jeitos daria às duas colunas
comportamentos diferentes no mesmo terminal.

## Derivada, não guardada

O handoff põe `tree FileTree` no `Model`. Aqui ela é **função pura sobre
`Entries`**, que já são a redução do log.

Um campo seria uma **segunda** coisa reduzida dos mesmos eventos, e duas
reduções de um log são duas coisas que podem discordar. Derivando, "mesma sessão
reaberta reproduz a mesma árvore" passa a ser verdade por construção em vez de
por cuidado — e é justamente o invariante que o design pede primeiro.

`Entry` ganhou `Added`/`Removed` como **números**. O `Summary` já os renderiza,
mas como frase, e ler a contagem de volta da frase é o que o comentário do
protocolo proíbe com todas as letras.

## Duas camadas, não uma árvore inteira

A coluna tem 20 a 30 caracteres, e cada nível de indentação são dois deles
tirados da única parte que identifica um arquivo: o nome. Então a linha de pasta
carrega o caminho inteiro — `internal/tui/` numa linha — e os arquivos ficam um
nível abaixo.

É a compactação de filho único do design levada até o fim. A primeira versão
indentava por componente de caminho e **derivou**: `model.go` num nível,
`rail.go` no seguinte, porque profundidade de caminho não é profundidade visual
depois que uma pasta foi compactada. Achado olhando a tela, não o diff.

## Três defeitos que só a tela mostrou

Além da indentação:

- **`ARQUIVOS 6 6 tocados`** — o `plural()` já traz o número, e eu somei outro.
- **O `+38` encostava no divisor**, lendo-se como parte da moldura. Agora sobra
  uma coluna de calha.
- **O divisor `│` estava cravado**, e em ASCII saía um caractere de caixa que o
  terminal não desenha. **Esse já existia** no divisor do painel; ficou visível
  quando uma segunda coluna repetiu o mesmo erro. Agora os dois seguem o
  `g.Unicode`.

## Movimento nunca é a única pista

O design pede pulso âmbar na linha tocada. Em ASCII e sem cor não há pulso, então
o conjunto de glifos mantém rodando, concluído e falhou **distintos por
caractere** — `*`, `x`, `!`. É a mesma regra que o painel de plano já carrega, e
pelo mesmo motivo: se bloqueado e concluído colapsam no mesmo caractere, o pior
erro possível fica invisível.

## O que não entrou

O pulso animado da linha tocada. Ele exigiria o quadro alcançando a coluna, e o
estado já é legível pelo glifo e pela cor — animação seria a terceira pista para
o que duas já dizem, custando repintura por linha.
