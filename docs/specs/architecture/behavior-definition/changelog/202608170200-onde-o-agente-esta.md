# O prefixo diz onde o agente está

**Data:** 2026-08-17

## O que mudou

`Prompt` ganha `Repo`, e o prefixo passa a carregar uma seção `This repository`
com branch, branch principal, estado da árvore e commits recentes — congelados
na criação da sessão, como a cadeia de instruções já é.

```
## This repository

A snapshot from when this session opened. It does not update, and your own
commits move it — run git yourself when the answer has to be current.

Current branch: fix/algo
Main branch: main

Uncommitted changes:
 M internal/app/app.go
?? internal/vcs/

Recent commits:
6e5eba2 fix: a category of tools is not a tool name
```

O pacote novo `internal/vcs` tira o instantâneo. **Só leitura**: nada ali
commita, estagia, cria branch ou empurra. O git é do usuário, e agente que mexe
nele sem pedir é agente que ninguém deixa sozinho num repositório. Rodar git
continua sendo trabalho do `bash`.

## Por que não é uma ferramenta

Porque o modelo teria de lembrar de chamá-la. A branch em que se está não é esse
tipo de fato: **toda regra sobre onde o trabalho pertence depende dela**, e regra
que exige consulta antes é regra seguida por acaso.

O `CLAUDE.md` deste projeto diz "um tema, uma branch", "nunca `git add -A`",
"estagie caminhos nomeados". Cada uma existe porque foi quebrada. E o agente não
conseguia seguir nenhuma sem alguém dizer, todo turno, em que branch ele estava.

## As decisões que o formato carrega

- **Instantâneo declarado.** O repositório se move enquanto a sessão roda — os
  commits do próprio agente o movem. Apresentar isso como corrente seria uma
  afirmação verdadeira no início e silenciosamente falsa depois do primeiro
  commit.
- **Árvore limpa é dita, não deduzida.** "Nada mudou" e "não olhei" leem igual
  quando os dois são vazios.
- **`HEAD` destacada não vira branch.** `git rev-parse --abbrev-ref HEAD`
  responde o literal `HEAD` quando não há nome. Tratar isso como nome é como um
  agente reporta trabalhar numa branch chamada `HEAD`.
- **Branch principal é perguntada, não adivinhada.** `origin/HEAD` é o que o
  remoto declara. Sem remoto, procura `main` e `master` — e **verifica que
  existem** em vez de nomear uma que não existe. Principal errada é pior que
  nenhuma, porque lê como resposta.
- **Status limitado a 40 linhas, com o corte declarado.** Repositório em merge
  com quatrocentos arquivos gastaria a janela numa lista que ninguém lê.
- **Diretório que não é repositório não produz seção alguma.** É o caso comum de
  um diretório de rascunho, e seção dizendo nada é pior que seção nenhuma.
- **Leitura limitada a 2s.** Roda antes do primeiro quadro, e sonda lenta é
  sentida como início lento.

`Build` continua pura: o instantâneo entra como dado, nunca como leitor.
