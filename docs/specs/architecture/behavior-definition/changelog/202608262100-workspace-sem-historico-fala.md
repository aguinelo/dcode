# Workspace sem histórico deixa de ser silencioso

**Data:** 2026-08-26
**Specs afetadas:** `202608080016-behavior-definition` — uma invariante da §8
muda de sentido e três nascem. Sem mudanças em outras famílias.

## O que era

`Repo` era `nil` para um diretório que não é repositório, `renderRepo` devolvia
`""`, e o comentário no campo dizia, com todas as letras, **"ordinary and
silent"**. A invariante dizia o mesmo: o prefixo carrega o estado do
repositório "e **nada** quando não é".

Foi decisão deliberada, e a metade dela estava certa. Não é anomalia trabalhar
num diretório de rascunho.

## O que aconteceu

Auditei um projeto real onde um agente trabalhou um dia inteiro. O que ele
deixou lá:

- 191 linhas de código e 35 arquivos de spec;
- um `DCODE.md` **escrito por ele mesmo** exigindo *"nada de PR sem spec
  aprovada"*, *"cada PR referencia a spec"*, *"80% é mínimo pra merge na
  main"*, commits convencionais;
- um `tasks.md` cuja terceira linha é *"Cada item tem commit próprio."*

E nenhum repositório git. `fatal: not a git repository`.

Nada daquilo podia acontecer. Todo o capítulo de processo era prosa sobre uma
máquina inexistente, e o harness sabia — ele **olhou**, na abertura da sessão,
e escolheu não dizer.

## A distinção que faltava

Ordinário e digno de nota não são a mesma coisa.

Sem repositório não há diff para alguém ler, não há revisão, e não há desfazer
que não seja reescrever o arquivo à mão. Isso não é detalhe de ambiente: muda o
que "terminar o trabalho" significa. É exatamente o tipo de fato que o próprio
`repo.go` já argumentava pertencer ao prefixo:

> "uma regra que precisa de consulta primeiro é uma regra seguida por acidente"

O branch está no prefixo por esse motivo. A ausência do repositório é mais
determinante que o nome do branch, e estava fora.

## O que passou a existir

`Repo.Absent` marca o workspace que não é repositório, e `renderRepo` diz uma
linha: sem histórico, sem diff, sem desfazer, commit/branch/PR indisponíveis —
com a instrução de **dizer uma vez, oferecer `git init`, e seguir o trabalho**.

É fato, não advertência. O trabalho acontece de qualquer jeito; o que muda é
que ninguém descobre depois. Repetir a cada turno seria a chateação que isto
deliberadamente não é, e está escrito na própria linha que não se repete.

## O que **não** mudou, e é a metade difícil

`nil` continua significando **instantâneo não tomado**, e continua silencioso.

"Não olhei" e "olhei e não há" são fatos diferentes, e só o segundo vale uma
linha. Juntar os dois colocaria no prefixo uma afirmação sobre o workspace com
base em nunca ter conferido — que é precisamente o defeito que este changelog
existe para remover.

A distinção custou três guardas dentro de uma função só:

1. **`git` não instalado** → `nil`. Não houve resposta.
2. **`rev-parse` respondeu que não** → `Absent`. Houve resposta.
3. **Sondagem cancelada ou estourou o prazo** → `nil`. Não houve resposta.

A terceira não foi previsão: `TestACancelledReadDoesNotHang`, que já existia,
reprovou na primeira versão desta mudança. O teste estava certo e a
implementação estava afirmando "não é repositório" sobre uma leitura que nunca
completou — o mesmo erro, no mesmo commit que o removia.

## Invariantes

Uma muda, três nascem:

| | |
|---|---|
| ~~carrega o estado do repositório e **nada** quando não é~~ | passa a cobrir só o caso presente |
| **nova** | workspace que não é repositório é dito uma vez, como fato |
| **nova** | a ausência não reivindica branch, árvore nem commits |
| **nova** | instantâneo não tomado não vira afirmação |

## O que isto **não** resolve

Dizer é o piso, não o teto. O agente ainda pode ler a linha e escrever um plano
que exige commit por tarefa — a linha diz que não dá, e obedecer a ela é
comportamento mediado, não garantia.

O que vem depois é a família `working-defaults`: o piso de prática, a
precedência entre o default embutido e o que o arquivo do projeto manda, e a
visibilidade de qual default foi sobreposto e por qual linha. Esta mudança é o
primeiro fato desse piso, entregue sozinho porque é determinístico e não
depende daquele desenho.
