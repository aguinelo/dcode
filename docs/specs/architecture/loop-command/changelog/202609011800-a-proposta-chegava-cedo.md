# A proposta chegava cedo

**2026-09-01** — RN-10: a proposta é escrita no evento de turno concluído.

## O sintoma

`/loop entender o projeto e propor melhorias sem implementar nada` abria a
sessão de qualificação com a frase certa — e imediatamente:

```
could not write the proposed definition of done: invalid_input:
nothing was proposed for 1a05dd01b225554ea98
```

O erro chegava **antes** de o modelo terminar de pensar. Esse é o sinal: se
tivesse vindo do fim do turno, viria depois.

## O mecanismo

O gatilho estava escrito assim:

```go
} else if p.model.State == protocol.SessionStateIdle && p.loopQualified != "" {
```

Sessão nasce **ociosa**, e o `attach` reproduz o log desde o começo. Então o
primeiro evento de uma sessão de qualificação recém-aberta encontrava
`State == idle` com `loopQualified` já preenchido, e mandava escrever uma
proposta que ninguém tinha feito.

## A correção, e por que ela não vale para o vizinho

O gatilho passa a ser o **evento de turno concluído**.

Vale notar por que a drenagem da fila, logo acima, continua lendo o estado e
está certa: ela quer **qualquer** momento em que nada está rodando, para mandar
o que a pessoa digitou enquanto assistia. A proposta quer **um** momento
específico — aquele turno acabando.

Estado ocioso responde a primeira pergunta. Só o evento responde a segunda. As
duas linhas parecem a mesma condição e são perguntas diferentes, e é por isso que
uma copiou a forma da outra e ficou errada.

## O que este defeito escondia

Ele só apareceu porque as duas correções anteriores funcionaram: a frase virou
objetivo qualificado (RN-8/RN-9) e o cliente parou de morrer ao trocar de sessão
(RN-20 da `client-tui`). Cada uma destravou a seguinte. Três defeitos em fila,
e só o primeiro era visível do lado de fora.

## Invariantes

- `TestTheProposalIsNotCommittedBeforeTheTurnStarts`
- `TestTheProposalIsCommittedWhenTheTurnCompletes`
