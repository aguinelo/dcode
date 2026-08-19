# Uma credencial pode ficar fora de alcance

Primeiro dos pontos levantados ao pensar o dcode como **coordenador de
máquinas** — SSH, servidores, orquestração — em vez de editor de código.

## O que foi medido

Sob `workspace-write`, com a rede concedida, nesta máquina:

| | |
|---|---|
| conexão TCP para `:22` | ✅ funciona |
| **ler `~/.ssh/id_ed25519`** | ✅ **funciona** |
| escrever em `~/.ssh/known_hosts` | ❌ negado |
| socket do `ssh-agent` | ❌ negado |
| `docker.sock` | ❌ negado |

A segunda linha é a que importa. **A sandbox não protege segredo nenhum.**

## Por que era assim, e por que continua sendo por padrão

Leitura é concedida em todo lugar de propósito, e a justificativa está no
código desde o começo: recusá-la impede o interpretador de carregar **antes** de
o comando rodar. Trocar isso por uma lista de permissões de leitura tornaria a
sandbox inutilizável.

O preço é aceitável enquanto o caso de uso é editar código no próprio
workspace. Deixa de ser quando a sessão alcança servidores: aí a chave privada é
a coisa mais valiosa da máquina, e o modelo pode lê-la e escrevê-la em qualquer
lugar onde escreve.

## O que passou a existir

`DCODE_SANDBOX_UNREADABLE` — uma lista de caminhos, separada como `PATH`, que a
sessão **não lê**. Vazia por padrão.

- **macOS:** `(deny file-read* (subpath ...))` **depois** do `(allow file-read*)`.
  A ordem é a regra inteira: o Seatbelt aplica a última que casa, então um deny
  escrito acima seria anulado pela linha que concede tudo.
- **Linux:** um `tmpfs` por cima do caminho. A montagem é o que o kernel lê, e
  não sobra nada para permitir ou negar.
- **`full-access` ignora a lista.** Modo que não promete fronteira não deve
  manter uma escondida.

## O default esconde, sem ninguém pedir

**Nobody asking is not nobody caring.** Uma sessão que nunca ouviu falar desta
variável também não deve conseguir ler uma credencial de nuvem.

```
~/.aws  ~/.gnupg  ~/.kube  ~/.config/gcloud  ~/.azure
~/.netrc  ~/.git-credentials  ~/.npmrc  ~/.pypirc  ~/.docker/config.json
<config-root>/credentials          ← a própria chave do dcode
```

O critério para entrar é estreito: **guarda segredo, e nenhuma ferramenta comum
precisa lê-lo como subprocesso de um comando comum.** É isso que torna esconder
gratuito. `aws` e `kubectl` são as exceções que aparecem primeiro, e são a razão
de a variável **substituir** a lista em vez de somar — quem precisa de uma de
volta diz qual.

A própria credencial do dcode entra pelo motivo mais direto de todos: uma sessão
que lê a chave com que roda pode entregá-la a qualquer lugar onde escreve, e a
redação que a mantém fora das transcrições não faz nada contra uma leitura de
arquivo. O nome do arquivo é duplicado em vez de importado — este pacote monta
perfis e não deve depender do cofre que esconde — e por isso a duplicação tem
guarda: renomear de um lado sem o outro deixaria a chave legível em silêncio.

## `~/.ssh` ainda não está, e a ausência é o ponto mais alto daqui

Chave privada é o segredo canônico. Escondê-la **hoje** pararia `git push` e todo
`ssh` de dentro do sandbox, porque o `ssh` lê a chave ele mesmo.

A saída é o agente. Com `SSH_AUTH_SOCK` alcançável, o `ssh` pede ao agente para
assinar e **nunca lê a chave** — e aí esconder sai de graça. Esse socket é
recusado hoje pela regra que mantém runtime de contêiner de fora.

Sozinho, cada lado é um trade ruim: conceder o socket sem esconder a chave não
protege nada, esconder a chave sem o socket quebra o trabalho. **Juntos não há
trade**, e é por isso que a concessão nomeada de socket vem em seguida e `~/.ssh`
entra junto com ela — não antes.

O home inteiro é recusado como nome, pelo mesmo motivo que `Scratch` recusa
concedê-lo: cobrir tudo abaixo dele não é conjunto nomeado, é outro modo com
nome emprestado.

## O teste tem controle

`TestARealNamedStoreCannotBeReadFromInside` lê o segredo **primeiro** com nada
nomeado, e exige que a leitura funcione. Só então nomeia e exige que falhe. Sem
essa primeira metade, o teste passaria com a sandbox quebrada de qualquer outra
maneira.

## O que este PR não resolve

Os outros pontos da mesma linha: concessão nomeada de socket e de caminho
gravável fora do workspace, e regras que reconheçam execução remota como
categoria — hoje `ssh host 'systemctl stop postgres'` passa sem perguntar, e
`ssh host 'rm -rf /'` só é pego porque a regra vê a string inteira, por acaso.

E o limite que nenhum deles resolve: **a contenção é local.** Do outro lado de
uma conexão SSH ela não alcança nada, e o único instrumento que sobra é o eixo
de autorização.
