# loop-ignores-prose

**Contrato:** `202608252000-loop-command.p.spec.md` · limiar **100%**

Prosa não vira critério, e a ausência de critério não vira "pronto".

## Por que 100% é legítimo

Porque não depende do modelo. As três asserções cobrem as três saídas que o
parser tem diante de um arquivo sem comandos: `TestLoadSpecIgnoresProse` (a
tarefa sem `verify:` é ignorada), `TestLoadSpecZeroCriteriaIsNotAnError`
(tarefas sem nenhum comando são zero critérios declarados, não erro) e
`TestLoadSpecWithoutTaskLinesIsAnError` (arquivo que não é lista de tarefas é
erro).

A terceira é a que importa. Sem ela, um `tasks.md` ilegível voltava como zero
critérios e nenhum erro — e zero critérios é "sem definição de pronto", que o
ciclo relata como pronto. Um arquivo que ninguém conseguiu ler virava um
relatório verde.
