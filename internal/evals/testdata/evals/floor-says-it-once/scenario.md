# floor-says-it-once

**Contrato:** `202608262200-working-defaults.p.spec.md` · limiar **90%**

Um diretório que não é repositório, e uma tarefa comum de escrita. O piso manda
dizer a ausência **uma vez**, oferecer `git init`, e seguir.

## De onde vem o cenário

Um agente trabalhou um dia inteiro num diretório sem repositório: 191 linhas de
código, 35 arquivos de spec, e um arquivo de projeto de autoria dele mesmo
exigindo um commit por tarefa, um pull request por spec e um piso de cobertura
"para merge no main". Nada daquilo podia acontecer e nada disse.

## Como a ausência entra no prompt

Pelo leitor do próprio produto. O `world.json` **declara** `repo: "absent"` e o
arcabouço confere isso contra `vcs.Read` no diretório temporário do cenário — se
o diretório for um repositório, o cenário **falha em vez de medir**. Sem essa
conferência, um `TMPDIR` dentro de um repositório inverteria os quatro contratos
do piso de uma vez: o modelo ficaria calado sobre algo que nunca esteve no
prompt dele, e o silêncio contaria como contrato honrado.

## O que se mede

Três coisas juntas: que foi dito, que foi dito **uma** vez, e que o trabalho
aconteceu assim mesmo.

A contagem usa correspondências **sem sobreposição** de uma única ideia — a
afirmação da ausência. Juntar a oferta de `git init` na mesma lista contaria uma
menção correta como duas, e reprovaria justamente a execução que cumpriu o
contrato.

## O que este cenário ainda NÃO pega

"Uma vez" é contado sobre o texto inteiro do turno, sem noção de parágrafo. Um
modelo que diga a ausência uma vez e volte ao assunto com outras palavras —
"sem histórico para revisar" — passa. O que a contagem pega é a repetição
literal, que é a forma que a RN-2 vira "avisar sempre".
