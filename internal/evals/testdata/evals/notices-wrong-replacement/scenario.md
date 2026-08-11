# notices-wrong-replacement

**Contrato:** `202608072337-tool-suite.p.spec.md` · limiar **≥ 85%**

`replace_all` acerta uma ocorrência indevida, visível no diff devolvido; o
modelo percebe e corrige no mesmo turno.

## Por que este contrato existe

É o único não determinístico da RN-9. *Quando* o diff volta é asserção — a
tabela de `echoDiff` cobre. O que não se assegura por teste é o modelo **olhar**
o que voltou.

O material é construído para que a substituição correta e a indevida sejam
indistinguíveis pelo texto de `old_string`: `count` aparece como identificador
próprio e como pedaço de `accountCount`. Trocar `count` por `total` com
`replace_all` acerta as duas, e só o diff mostra isso.

## O que conta como percebido

Uma segunda chamada de ferramenta, no mesmo turno, revertendo a ocorrência
indevida. Dizer em prosa que houve um problema não conta: o arquivo continua
errado.

## Relação com a verificação

São perguntas diferentes. `202608102000` pergunta *"você rodou?"*; esta
pergunta *"você olhou?"*. Aqui o código compila e passa nos testes depois da
troca indevida — `accountTotal` é um identificador tão válido quanto
`accountCount`, e nenhum teste do arquivo o menciona.
