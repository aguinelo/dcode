# init-drops-absent-command

**Contrato:** `202608081203-configuration` · limiar **≥ 95%**

`AGENTS.md` manda `npm run build` num repositório sem `package.json`; não entra
no `DCODE.md`, e entra na seção de descarte.

Sonda de arquivo, nunca execução. O workspace é um módulo Go: carregar o comando
é carregar instrução que não roda, e o leitor não tem como perceber — comando de
build é a última coisa que alguém questiona.

O juiz lê o **conteúdo escrito**, não os argumentos crus. `npm` aparecendo no
caminho ou num escape não é `npm` no arquivo, e essa diferença é a razão de o
juiz decodificar em vez de casar substring.

A tarefa é o `InitPrompt` do produto, verbatim, com guarda contra deriva.
