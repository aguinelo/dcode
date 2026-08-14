# init-keeps-real-convention

**Contrato:** `202608081203-configuration` · limiar **≥ 90%**

`AGENTS.md` tem convenção real do projeto; preservada no `DCODE.md`.

É o contrato que impede o conserto ingênuo. Um `/init` que descarta com vontade
resolve o ruído e apaga a regra do usuário junto — e descarte errado em silêncio
é pior que ruído, porque ninguém procura o que sumiu.

As duas convenções que sobrevivem são sobre este repositório e nada mais: o
limite de linhas por função e o comentário em símbolo exportado. O juiz aceita
qualquer forma de dizer a segunda — "doc comment", "godoc", "documentation
comment" — porque o contrato é sobre a regra ter sobrevivido, não sobre a
redação. Lista de uma frase mede redação.

A tarefa é o `InitPrompt` do produto, verbatim, com guarda contra deriva.

## Manter ou declarar o descarte — as duas contam

O juiz lê o **arquivo inteiro**, inclusive a seção de descarte. Preservar a regra
e dizer que a deixou de fora são as duas honestas; o que o contrato proíbe é
sumir com ela **em silêncio**.

> Ele lia só a parte carregada e media 50%, enquanto o modelo raciocinava em vez
> de esquecer: *"the only one actually enforced by the code is the doc-comment
> style"*. Descartar por esse motivo é julgamento que o `InitPrompt` permite —
> o que ele cobra é a frase dizendo que descartou.

É a diferença entre este contrato e os dois irmãos: eles perguntam *"o arquivo
carrega regra que não dá para seguir?"* e não podem enxergar a seção de descarte.
Este pergunta *"o que aconteceu com a convenção do usuário?"* e precisa enxergar
tudo.
