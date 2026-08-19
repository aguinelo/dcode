# Um recurso de fora pode ser concedido por nome

Segundo ponto da linha de coordenar máquinas, e é ele que torna o primeiro
gratuito.

## A regra continua inteira

**Socket unix é alcançável exatamente onde já se pode escrever.** É isso que
mantém o daemon do Docker fora, e não mudou.

Conceder é dizer a exceção em voz alta:

| variável | o que nomeia |
|---|---|
| `DCODE_SANDBOX_SOCKETS` | sockets alcançáveis mesmo sem o diretório ser concedido |
| `DCODE_SANDBOX_WRITABLE` | caminhos graváveis fora do workspace |

O literal **`ssh-agent`** vale por `$SSH_AUTH_SOCK`. Ele é por boot e por login
— `/var/run/com.apple.launchd.*` no macOS — então nenhum arquivo de configuração
poderia nomeá-lo e nenhum default poderia adivinhá-lo. Sem agente rodando, o
token concede **nada**: a string vazia como caminho nomearia coisa demais em
algum backend, e "não há agente" não é "conceda tudo".

`~/.ssh/known_hosts` é o caso que aparece primeiro do lado gravável, porque sem
ele a **primeira** conexão a um host novo falha — e falha de um jeito que parece
problema de rede.

## O par que se resolve

Sozinho, cada lado era um trade ruim:

- esconder a chave privada enquanto o `ssh` precisa lê-la **para todo `git push`
  e toda conexão**;
- conceder o socket do agente com a chave ainda legível **não protege nada**.

Com o agente alcançável, o `ssh` pede a ele para assinar e **nunca abre a
chave**. Aí esconder sai de graça — e `~/.ssh` entra no conjunto escondido por
default, **condicionado a essa concessão**.

Essa condição é o desenho inteiro, e está numa linha só: o default esconde o que
pode ser escondido sem custo, e quem quer o custo zero concede o agente.

## Config, não pergunta

O perfil é montado a partir destes valores **antes de existir qualquer turno**.
Uma pergunta no meio da sessão não teria onde aterrissar, e permissão que só vale
depois de reiniciar é permissão que o usuário vê falhar — que é exatamente o
comentário que a concessão de rede já carrega no código.

## O que continua fora

`docker.sock` não ganhou default nenhum. Quem quiser, nomeia — e ao nomear está
dizendo, sabendo, que o daemon do outro lado não está no sandbox e monta qualquer
caminho do host a pedido.

E o limite que nenhuma destas linhas resolve: **a contenção é local.** Do outro
lado de uma conexão SSH ela não alcança nada. O que sobra ali é o eixo de
autorização, e é o próximo passo.
