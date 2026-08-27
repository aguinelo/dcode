# Um objetivo em vez de um caminho

**Data:** 2026-08-27
**Specs afetadas:** `202608252000-loop-command` — nove invariantes novas na §8.
Implementa a **RN-7** e a **US-2**, escritas em 2026-08-25 e não construídas.
Toca `client-server-protocol` de forma aditiva.

## O relato

> `/loop implemente todas as specs pendentes` →
> **`read /Users/.../codeplain/em/tasks.md: no such file or directory`**

A prosa virou caminho. A primeira palavra da frase virou nome de pasta, e o
parser foi procurar `em/tasks.md`.

É a mesma forma do defeito que esta família inteira persegue — prosa virando
critério — apontando para o outro lado. E a US-2 dizia, desde o começo:

> digitar `/loop implemente absolutamente todas as specs` e ver o agente iterar
> sobre as 16 specs da plataforma

Nunca foi construída, e o parser não tinha regra: a primeira palavra era o
caminho, sempre.

## A regra, e ela não toca disco

| argumento | é |
|---|---|
| `specs/home-page` | caminho — tem separador |
| `home-page` | caminho — uma palavra é o que um nome de pasta parece |
| `specs/home-page refaça o header` | caminho, e o resto é o que fazer |
| `implemente todas as specs pendentes` | **objetivo** |

Determinística, e o cliente não consulta disco nenhum — ele pode não estar
perto do disco do daemon. Uma palavra só continua sendo tratada como caminho,
e o erro nomeia a pasta quando ela não existe.

## "Pendente" é medido, não contado

A pergunta *"quais specs faltam"* tem uma resposta barata e errada — contar
`- [ ]` no `tasks.md` — e uma cara e certa: **rodar os critérios da pasta**.

Uma marcação é feita por quem teve vontade de fazer. O critério é a definição
de pronto, e é a única coisa neste produto que responde por ela. Então a
descoberta roda cada `done.toml` de cada spec, pelo mesmo sandbox que um turno
usaria, e o que sobrar é o trabalho.

E, por isso, o daemon é quem decide: ele tem o disco e o sandbox, e dois
clientes discordando sobre o que falta é pior que qualquer uma das respostas.

**Pasta sem critério nenhum é pendente.** Não há prova de que terminou, e
tratar "nada a conferir" como "pronto" é o defeito que esta família existe para
impedir. Critério indisponível conta igual: um comando que ninguém conseguiu
rodar não é evidência de que algo passou.

## O que a primeira medição real mostrou

Rodado contra o Code Plain: **11 de 28 specs voltaram "ilegíveis"** — todas com
`spec.md` e sem `tasks.md`. E, pior, `Pending()` devolvia falso para elas, então
as 11 ficavam **fora da fila**.

Exatamente ao contrário. Uma spec escrita e ainda não quebrada em tarefas é a
coisa mais pendente da lista: nada afirma que terminou e nada foi sequer
planejado.

Depois do conserto: **28 de 28 pendentes, nenhuma ilegível.** Só apareceu
porque a descoberta rodou contra um repositório de verdade, com 28 pastas em
estados diferentes, em vez de contra fixtures.

## O plano vem antes de qualquer coisa rodar

`/loop <objetivo>` mostra **todas** as pastas e onde cada uma está, não só as
pendentes. Uma lista só do que falta deixaria alguém sem distinguir "esta spec
terminou" de "o dcode não viu esta spec" — e as duas pedem reações diferentes.

Depois disso, uma sessão por spec, em ordem de nome, cada uma com a própria
`DoneSet`, o próprio orçamento e o próprio relatório (RN-2). A fila é do
operador: interromper entre specs mantém pronto o que ficou pronto.

E a fila de specs só anda **depois** da fila digitada pela pessoa. Algo que
alguém escreveu enquanto olhava é sobre a spec na tela; passar para a próxima
antes mandaria aquilo para a spec errada.

## Um guard consertado no caminho

`TestNoEnglishSurvivesAPortugueseScreen` deriva palavras do catálogo inglês e
as procurava por **substring** numa tela em português. A palavra `works`, vinda
de uma frase nova, casou dentro de **`workspace-write`** — que é um valor na
tela, não texto de layout.

Passou a casar palavra inteira. É a terceira vez hoje que "casar string solta"
reprova ou classifica errado, nos três lugares sem relação entre si.
