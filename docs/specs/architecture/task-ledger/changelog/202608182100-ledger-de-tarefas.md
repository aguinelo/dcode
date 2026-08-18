# Ledger de tarefas

**Data:** 2026-08-18

## O que entrou

O desenho de um backlog que sobrevive à sessão: o que falta fazer, em que ordem,
com que critérios, e o que já foi conferido de verdade.

Desenho, não código. Nada disto existe implementado ainda, e a `.p` diz isso na
primeira seção.

## O problema

Um backlog de dez horas não é um turno de dez horas. São trinta tarefas
sequenciais com critérios de aceite. Havia duas peças e faltava a terceira:
`plan` vive dentro do turno e morre com ele; `done.toml` define pronto para o
**workspace**, não por tarefa. O que falta fazer não atravessava a sessão.

## A decisão que gera todas as outras

**Quem executa não assina o aceite.**

O desenho de referência nesse espaço (`snarktank/ralph`) deixa o modelo escrever
`passes: true` — o mesmo ator que fez o trabalho declara que os critérios foram
atendidos. Juiz da própria causa.

Aqui o modelo pode **começar**, **bloquear** e **afirmar**. A ausência de uma
quarta ação é o desenho: a linha que fecha uma tarefa é escrita pelo executor, a
partir do selo.

## Três consequências diretas

**Dois arquivos, não um.** A intenção (`backlog.toml`) é escrita por pessoa e o
agente nunca escreve nela. O progresso (`backlog.log`) é append-only e só
ferramenta escreve. Juntar os dois é como edição à mão é atropelada.

**Conferido e afirmado nunca recebem a mesma palavra.** Critério com comando
roda; critério em prosa é afirmação. Tarefa cujos critérios são todos afirmação
jamais aparece como verificada. A palavra "pronto" não existe sozinha em lugar
nenhum do modelo de estados.

**A parada é um fato sobre arquivos.** Não uma frase no stdout do modelo. O
backlog acabou quando nenhuma tarefa está pendente ou ativa.

## O que se copiou de fora

Uma coisa, e é boa: **sessão nova por tarefa**. Não brigar com o contexto — jogar
fora. Se cada tarefa cabe num contexto, ninguém precisa de um turno de dez horas,
e isso resolve sozinho a lacuna registrada em `docs/ROADMAP.md` §7.

## O que ficou de fora, e por quê

**Mais de um repositório.** Sandbox, resolver, política e cadeia de instruções
são ancorados num workspace. Um ledger atravessando repositórios prometeria o que
a fronteira não sustenta.

**O agente propor tarefas.** Se entrar, tarefa inventada por ele tem de ser
distinguível da que uma pessoa pediu — mesma lógica de `learned` ordenar abaixo
de tudo que uma pessoa escreveu.

**Grafo de dependências.** Transforma um backlog legível num formato que só
ferramenta lê. A ordem já expressa "depois de".

## Duas ausências que são o repositório funcionando

A `.p` traz **Invariantes previstas**, não "verificáveis": aqui uma invariante
verificável é uma reivindicação sobre um teste que existe, e não há o que
reivindicar antes do código.

E **não há `.i.spec.md`**. A tentativa de escrever uma foi recusada pela guarda,
que exige que todo caminho citado exista no repositório — uma spec de
implementação descreve o que **está** construído, não o que se pretende
construir. A distinção não estava clara para mim antes de tentar; agora está
escrita na `.p` §1 para que ninguém leia nenhuma das duas ausências como
descuido.
