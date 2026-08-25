# O modo viaja pelo log, como todo fato observável

`POST /sessions/{id}/mode` troca o modo comportamental da sessão, e `Session`
ganha o campo `mode` ao lado do `sandbox_mode` que já existia. Adição de rota e
de campo: **MINOR**.

A rota é dela mesma e não uma bandeira em `turns` porque trocar de modo não é um
turno — não começa nem termina nenhum. Responde `204` sem corpo, porque o que
mudou não está na resposta: está no log, como `session.mode_changed`.

Essa é a decisão que vale registrar. O modo **poderia** ser lido de `GET
/sessions/{id}` a cada troca, e a troca ser um canal lateral. Não é, pelo mesmo
motivo que `session.created` já viaja pelo log: um cliente que anexa depois de
a troca acontecer tem de aprender o modo do mesmo lugar de onde aprende todo o
resto. Um fato observável que só existe na resposta de quem o causou é um fato
que o segundo cliente não tem como descobrir — e este repositório já pagou por
essa forma de defeito mais de uma vez, sempre com o mesmo nome: o daemon sabe e
não conta.

O evento carrega `previous` além de `mode`. A transcrição mostra a transição e
não só o destino: "assist → auto" é o que aconteceu; "auto" é onde se parou.
`previous` vem vazio no primeiro anúncio, quando não havia de onde vir.

O campo `mode` de `Session` **não leva `omitempty`**. Vazio é resposta — é o par
que não corresponde a nenhum dos três modos (§2.1 de `sandbox-policy`) — e um
campo ausente diria "não sei se este servidor tem modos", que é outra coisa.

Nome desconhecido é recusado com `4xx` nomeando o que foi enviado, antes de
chegar ao motor. A validação fica no `.p` da sessão e não no handler porque o
mesmo nome chega por três caminhos — a rota, o comando `/mode` e a tecla — e uma
validação por caminho é três validações que divergem.
