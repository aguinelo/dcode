# Um comando que sai da máquina pergunta

Último ponto da linha de coordenar máquinas, e o que resta quando a contenção
acaba.

## A assimetria

Para tudo o mais na lista de comandos destrutivos, a sandbox ainda limita **onde**
o estrago cai. `rm -rf` apaga dentro do workspace; um `push` forçado reescreve um
repositório que se pode reclonar.

`ssh deploy@prod 'systemctl stop postgres'` acontece num lugar onde a contenção
**não alcança nada**. Nada ali é desfeito daqui.

Isso muda o estatuto da pergunta. Para os outros, ela é segunda linha de defesa.
Para este, **é a única**.

## O que entrou

Um terceiro grupo em `destructiveCommands`: `ssh`, `scp`, `rsync` para um host,
`kubectl exec`/`cp`/`port-forward`, `ansible`, `aws ssm`, e `docker` apontado
para outro daemon — inclusive via `DOCKER_HOST=`.

**A pergunta dispara na travessia, não no que se pede do outro lado.** O lado de
lá não pode ser lido, e fingir julgá-lo seria pior que admitir que não dá.

## Onde a regra é ancorada, e por quê

No início do comando e nos encadeamentos usuais (`&&`, `;`, `|`). Assim
`cat ~/.ssh/config` não é conexão com lugar nenhum, e `grep ssh arquivo.go`
tampouco.

`git push` **não** dispara, e essa é a decisão mais importante desta lista. Ele
alcança um remoto através de `ssh` sem ser execução remota, e a regra lê **o
comando declarado**, não o que os subprocessos dele fazem.

Dizer isso é o ponto: regra é atenção, e **atenção que dispara em todo push é
atenção que ninguém lê.**

## O que isto não é

Não é fronteira, e o `rulesDoc` ao lado da lista já dizia por quê: um comando é
texto, e a mesma coisa sempre pode ser escrita de outro jeito — por um script,
por um alias, por uma variável. O que a lista compra é atrito contra o acidente,
que é o que de fato acontece.

Contra o outro lado de um SSH, atrito contra o acidente é tudo o que existe. E é
melhor ter isso escrito do que descobrir depois que não havia nada.
