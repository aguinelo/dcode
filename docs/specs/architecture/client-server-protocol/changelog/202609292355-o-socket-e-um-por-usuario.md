# O socket é um por usuário

**Data:** 2026-09-29
**Specs afetadas:** `202608072240-client-server-protocol` (`.config`: §1, a
linha de `DCODE_SOCKET`; `.p`: três linhas novas em §9)
**Fonte:** o pedido N1 do cliente desktop (`desktop/docs/DECISIONS.md`, D19),
decidido com a pessoa em 2026-09-29.

## O problema

O caminho padrão vinha de `XDG_RUNTIME_DIR`, senão de `TMPDIR`. São variáveis de
ambiente, e quem as tem varia: um terminal num Linux com systemd tem as duas; um
app aberto pelo Dock tem só o `TMPDIR` que o launchd dá; uma sessão SSH ou um
`cron` podem não ter nenhuma. Dois clientes da mesma pessoa podiam, cada um,
achar "o" daemon e achar daemons diferentes — ou não achar o que estava rodando.

O desktop é aberto pelo Dock e precisa anexar ao mesmo daemon que a TUI. Sem
isto, só acertaria copiando a regra, e cópia é o que diverge.

## O que mudou

- Sem `DCODE_SOCKET`, o socket é `/tmp/dcode-<uid>/dcode.sock`, e o default não
  lê mais nenhuma variável. `DCODE_SOCKET` e `--socket` continuam sendo a escolha
  explícita, e ficam como foram escolhidos.
- A pasta do default é criada com `0700`. Se já existe, tem de ser um diretório
  de verdade (não symlink), do usuário, e fechado para os outros. Senão
  `dcode serve` não escuta, a TUI não anexa e `dcode socket` não imprime — e cada
  um diz o que está errado e como resolver.
- `dcode socket` imprime o caminho em uso. É por ele que o desktop pergunta onde
  está o daemon.

## Por que `/tmp/dcode-<uid>/`

É o que o tmux faz, pelos mesmos motivos:

- **O mesmo para todo processo do usuário**, venha de onde vier.
- **Curto.** Um caminho de socket Unix tem teto perto de 104 bytes no macOS, e a
  pasta de estado do XDG sozinha pode estourar isso — a razão de o caminho ser
  curto, em `docs/DECISIONS.md`, continua valendo.
- **Fora do alcance de um comando confinado.** No Linux, todo sandbox fora do
  acesso total monta um `tmpfs` próprio em `/tmp`, então o `/tmp/dcode-<uid>/` do
  host não existe lá dentro; no macOS, o perfil nega socket Unix de qualquer
  jeito. O default anterior, `$XDG_RUNTIME_DIR/dcode.sock`, ficava em
  `/run/user/<uid>`, que o sandbox do Linux monta só leitura — e, como o próprio
  `internal/sandbox` registra, conectar num socket não é escrever.

Descartados:

- **`$XDG_RUNTIME_DIR` quando existe**, a convenção do Linux: é do ambiente, que
  era o problema, e fica ao alcance do sandbox.
- **A pasta de estado** (`~/.local/state/dcode`, ou a do macOS): depende de
  `HOME`, `DCODE_HOME` e `XDG_STATE_HOME`, pode passar dos 104 bytes, e também
  fica ao alcance do sandbox no Linux.
- **A pasta temporária do usuário no macOS** (`_CS_DARWIN_USER_TEMP_DIR`):
  privada e sem ambiente, mas só existe no macOS e pede cgo ou um subprocesso.
- **Copiar a regra no desktop:** a cópia que diverge.

## Por que recusar em vez de consertar

Uma pasta que estava aberta a outros pode já ter o socket de outra pessoa dentro,
exatamente onde todo cliente deste usuário procura. Apertar o modo agora não
tiraria esse socket de lá. A pasta errada é dita, com o comando que resolve, e
nada escuta nem conecta nela.

## O que muda para quem usa

Quem roda `dcode serve` e a TUI da mesma versão não percebe nada: os dois leem a
mesma regra. Um `dcode serve` de versão anterior, ainda rodando, escuta no
caminho antigo, e uma TUI nova não o acha — sobe o próprio daemon embutido, como
faz quando nenhum responde. Quem fixava `DCODE_SOCKET` segue igual.

Estabilidade da spec: `experimental`. Muda o default de uma variável declarada e
acrescenta um comando: MINOR.

## As invariantes

As três linhas novas em §9 são reivindicadas por
`TestTheDefaultSocketDoesNotDependOnTheEnvironment` e
`TestASocketDirectoryNotOwnedAloneIsRefused`, em `internal/app`, e por
`TestSocketPrintsWhereTheDaemonListens`, em `cmd/dcode` — que passa a constar na
lista de diretórios do guarda do protocolo.

## O que fica de fora

Um `DCODE_SOCKET` escolhido fora de `/tmp` continua ao alcance de um comando
confinado no Linux: o sandbox cobre uma lista de sockets conhecidos, e o do
próprio daemon não está nela. Cobrir é mudança do `sandbox-policy`, com o seu
próprio changelog.
