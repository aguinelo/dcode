# O registro para de se copiar

**Data:** 2026-08-25
**Specs afetadas:** `client-server-protocol` (`.p`, seção de eventos)
**Fonte:** medição, ao investigar por que `dcode -c` carregava tanto

## O que mudou

A conversa continuada entra no **log** da sessão nova e não no **registro** dela.
O registro guarda o marcador `session.resumed`, e ler um registro segue a cadeia
para trás.

## Crescimento quadrático

Continuar copiava o registro anterior inteiro para o novo. Então uma sessão que
continuou uma que continuou uma guardava três cópias da primeira:

```
sessão A: 900 eventos
  -c  →  B: 900 copiados + os próprios
  -c  →  C: 1800 copiados + os próprios
  -c  →  D: 3600 copiados + os próprios
```

O maior registro desta máquina tem **3,6 MB e 18.410 eventos**. A conversa em si
não teve dezoito mil eventos; a maior parte é ela mesma, repetida.

## Por que a cópia existia

O comentário original está certo sobre o problema e errado sobre o preço:
*"o cliente desenha a tela a partir de eventos, o registro é escrito a partir de
eventos, e a próxima sessão a continuar esta reconstrói a partir do registro.
Uma colocação responde às três."*

Responde mesmo — só que a terceira não precisa da cópia. Precisa de **saber onde
ela está**, e o marcador já diz.

## A separação que já existia

`EventLog` já tinha o registro como destino separado da memória. `AppendUnrecorded`
usa isso: os eventos carregados vão para o log e para todo assinante, e não para
o disco. A tela continua servida do log em memória; o disco fica linear.

## E ler segue a cadeia

`Carry` caminha para trás pelo marcador, mais antigo primeiro, e `Rebuild` passou
a ser construída sobre ela. Ler o arquivo direto reconstruiria só a última perna
e entregaria ao modelo uma conversa que começa no meio de si mesma.

Duas decisões dentro disso:

**Ancestral que sumiu é conversa mais curta, não falha.** Recusar ali tornaria um
registro podado ilegível para toda sessão que um dia o continuou.

**Cadeia que aponta para si mesma é lida uma vez.** Registro que se nomeia, ou dois
que se nomeiam, é um par corrompido e não um par impossível: o id é um carimbo de
tempo mais um sufixo aleatório, e nada garante que a seta aponte para trás. O
teste tem prazo, porque a forma de falhar sem o guarda é não terminar.

## O que isto não é

Não é a assinatura em janela que estava no plano. Aquela economizaria 98 ms de
redução e a transferência, ao custo de uma janela parcial em que uma chamada sem
resultado carregado renderiza como rodando para sempre. Medido, e trocado por
esta — que ataca o motivo de os registros ficarem grandes em vez de contornar o
sintoma de eles serem grandes.
