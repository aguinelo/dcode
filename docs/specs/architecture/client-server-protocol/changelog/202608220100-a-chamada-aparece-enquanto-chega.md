# A chamada aparece enquanto chega

**Data:** 2026-08-22
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`), `202608081250-client-tui` (`.p`)
**Fonte:** relato de uso — *"as animações e o código sendo criado, animado, não está"*

## O que estava acontecendo

Num `write` de algumas centenas de linhas, o modelo transmitia o arquivo inteiro
e **a tela não mostrava nada**. Nem o nome da ferramenta. Depois de vários
segundos a chamada aparecia já completa.

O `family_openai.go` conhecia as duas coisas e jogava as duas fora:

- em `content_block_start`, o **nome e o id** da ferramenta — e devolvia `nil`;
- em cada `content_block_delta`, os **bytes** que chegavam — `d.append(...)`,
  `return nil, nil`.

Só no `message_stop` o `flush()` emitia a chamada montada. Silêncio exatamente na
parte do turno em que o trabalho está acontecendo, que é o que faz uma interface
viva se ler como uma interface morta.

## O que passou a existir

Dois eventos de stream no provedor — `tool_call_opened` e `tool_call_progress` —
que o laço converte em `progress` com `kind: "arguments"`.

**Não é um evento novo de protocolo.** É o `progress` que já existia, com um
campo `Name`: um sujeito que ainda não existe precisa se nomear, porque o
`tool.requested` — que normalmente carrega o nome — só chega quando a chamada
termina de montar.

Bytes, não linhas: o que chegou é fragmento de JSON, e contar linha dentro de uma
string escapada pela metade é contar algo que ainda não está lá. Sem total: o
modelo não diz quanto a chamada vai ter, e denominador que ninguém mandou é
denominador em que alguém acredita.

## Passo de meio kilobyte

Fragmento pode ter punhado de bytes, e um evento por fragmento poria milhares de
linhas no registro de um único `write` grande. O throttle fica **no laço**, não
no provedor: o provedor relata o que vê, o protocolo decide o que vale dizer.

O primeiro relato é sempre enviado — é ele que põe a linha na tela.

## No cliente

A linha aparece na hora, marcada como `Arriving`: está rodando, mas ainda não
executando — nada foi feito, e a contagem é de bytes de um pedido, não de
trabalho. Por isso ela diz `2.0k` e não `2048`, que se leria como trabalho
pronto.

Quando o `tool.requested` chega, ele **preenche a linha que já existe** em vez de
desenhar outra. Sem isso a mesma chamada ficaria na tela duas vezes, uma
meio-chegada e uma real.

## Compatibilidade

Consumidor que ignora os dois eventos novos vê exatamente a sequência que via
antes, e há teste para isso. Um provedor que não consegue relatá-los fica calado
em vez de fingir — o cliente desenha no `tool.requested` como sempre desenhou.

## Como foi achado

Alguém usou o produto e disse que nada animava. Não havia teste que pudesse
pegar isto: cada camada estava correta isoladamente, e o defeito era o que
**nenhuma delas dizia** entre uma e outra.
