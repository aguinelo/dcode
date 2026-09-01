# O agente não sabia do próprio mecanismo

**2026-08-31** — o bloco de skills passa a ser renderizado sempre, e a dizer onde
as skills moram.

## A resposta que o produto deu

Pedido para instalar e usar uma skill, o agente respondeu:

> Não tenho como instalar skills/plugins no meu ambiente — sou um agente que
> opera só com as tools que tu vê (bash, edit, read, etc.). Plugins e skills do
> Claude Code rodam no Claude Code em si, não são instaláveis em qualquer lugar,
> e muito menos a partir daqui.

Cada frase disso é falsa sobre o produto que ele é. O dcode carrega skills de
`<workspace>/.dcode/skills/` e de `skills/` sob a raiz do usuário. Escrever ali é
**escrita dentro do workspace** — não cruza fronteira, não pede aprovação. Ele
tinha a ferramenta e a permissão; faltava a informação.

## Por que ele respondeu isso

O bloco era renderizado sob `if len(p.SkillIndex) > 0`, e dizia só *"Load one of
these only when the situation matches"*.

Duas consequências. Com zero skills instaladas, **nada** no prefixo mencionava
que o mecanismo existe. E mesmo com skills, nada dizia onde elas ficavam nem que
uma nova podia ser escrita.

Sem informação, o modelo respondeu pelo treino — e o treino dele sobre "skill" é
de outro produto. Ele não alucinou: ele extrapolou, com confiança, de um lugar
onde nós não tínhamos escrito nada.

## O que o bloco passa a dizer

Onde moram, e o que escrever uma faz: é escrita de arquivo comum, indexada a
partir da sessão seguinte. A parte da sessão seguinte importa e é fácil de
esquecer — a descoberta acontece uma vez, na montagem, porque o prefixo é onde o
cache é chaveado (RN-5).

Duas linhas. Não um manual: a economia da RN-7 é a mesma que mantém os corpos
fora do prefixo. O que essas duas linhas compram é a alternativa não ser o
produto desinformando a pessoa sobre ele mesmo.

## Cabeçalho vazio continua proibido

A regra de que seção vazia não emite cabeçalho continua valendo, pelo motivo de
sempre: diferença de bytes contra uma sessão que nunca teve a seção erra o cache.
Este bloco **não é vazio** — o conteúdo é fixo e byte-idêntico entre sessões sem
skill nenhuma.

## Uma guarda que lia string vazia

`TestAbsentSectionsEmitNoHeading` afirmava que `## Skills` era omitida quando
vazia. Ela passava — e passava percorrendo **nada**: o `Prompt` do teste não tinha
`Safety`, então o `Build` falhava, e o laço varria uma string vazia procurando
quatro cabeçalhos. Guarda que não lê nada concorda com tudo.

Corrigida junto, porque é exatamente o comportamento que esta mudança altera, e
deixá-la de teatro enquanto se mexe no que ela diz guardar seria pior que não
tê-la.

## Invariantes

- `TestTheAgentIsToldWhereSkillsLiveEvenWithNoneInstalled` — a seção existe sem
  skill nenhuma, diz o caminho, e cabe em 400 bytes.
- `TestAbsentSectionsEmitNoHeading` — agora sobre um prompt que de fato monta.
