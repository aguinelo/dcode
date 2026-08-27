# O comando faz o trabalho

**Data:** 2026-08-27
**Specs afetadas:** `202608252000-loop-command` — quatro invariantes novas na §8.

## O relato

> `/loop <caminho> implementar ...` →
> **`implementar is not a flag here — only --protect is`**
>
> e, antes disso, a sessão aberta e o agente parado: *"Me chama quando quiser
> que eu monte o plano."*

Dois defeitos, e o segundo é o que importa.

## `/loop` abria a sessão e não fazia nada

Ele carregava a definição de pronto, trocava de sessão, escrevia a nota — e
**não submetia turno nenhum**. Quem digitou `/loop specs/x` tinha que dizer, em
seguida, o que queria.

`/loop specs/x` quer dizer **faça esta spec**. Abrir uma sessão medida contra
ela e sentar é o comando fazendo metade do trabalho, e a metade que sobra é a
que a pessoa achou que estava delegando.

Agora ele submete. O texto padrão nomeia a spec, manda ler antes, e diz que a
sessão já carrega a definição de pronto e que o harness a confere — que é o que
impede o agente de sair procurando os critérios para rodá-los por conta e
relatar pronto na própria palavra.

**O que ele não faz é repetir os critérios.** Eles já são da sessão, o laço os
confere, e uma cópia na primeira mensagem é uma segunda afirmação de algo que
pode se mover.

## Palavra não é flag mistecleada

`implementar` nunca foi uma flag. O parser tratava o **segundo** termo sem
traço como erro e o rendia com a frase de flag desconhecida, então uma frase
inteira era recusada dizendo que era uma flag errada.

Agora: o primeiro termo é o caminho, **tudo depois é o que fazer**, como
digitado. Só o que começa com `-` pode ser flag errada.

`/loop specs/x` faz a spec. `/loop specs/x refaça só o header` faz o que a
pessoa pediu. A segunda forma era a que estava sendo recusada, e é a mais
natural das duas.

## Medido

Spec com dois critérios vermelhos, e **nada dito além do que o `/loop` manda
sozinho**:

```
⏺ ls specs/slug/  ⏺ read done.toml  ⏺ read spec.md  ⏺ read tasks.md
⏺ write slug.sh
⏺ bash sh slug.sh 'Ola Mundo' && …
```

Os dois critérios saem `exit 0`, conferidos fora da sessão. O agente leu o
`done.toml` para saber contra o que estava sendo medido, e não foi porque
alguém colou os critérios no pedido.
