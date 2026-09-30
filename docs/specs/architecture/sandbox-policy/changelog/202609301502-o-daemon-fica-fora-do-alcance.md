# O daemon fica fora do alcance do sandbox

**Data:** 2026-09-30
**Specs afetadas:** `202608072336-sandbox-policy` (`.p`: oito linhas novas em §6)
**Fonte:** achado ao mudar o socket padrão (N1, PR #407) e reproduzido pela
montagem do próprio daemon, numa sessão de verdade.

## O defeito

Um daemon do dcode roda sem confinamento: abre sessões, inclusive em acesso
total, e responde aprovações. Um comando confinado que alcance o socket dele
pede uma sessão em acesso total, ou aprova a si mesmo — e a fronteira deixa de
existir.

- **No macOS**, o perfil deixa um socket unix alcançável onde se pode escrever,
  e `workspace-write` escreve em `/tmp` e no `$TMPDIR`. É exatamente onde os
  daemons escutam: o `dcode serve` em `/tmp/dcode-<uid>/` desde o #407 (antes,
  no `$TMPDIR`), e o daemon embutido da TUI no `$TMPDIR`. Com a rede concedida,
  que é o default, um comando dentro da sessão alcançava o daemon. Reproduzido
  na `main`, numa sessão de verdade: `"exit 0\nCONTROL\nREACHED\n"`.
- **No Linux**, o sandbox cobre uma lista de sockets de contêiner, e o do daemon
  não estava nela. O que fica sob `/tmp` some no `tmpfs` do sandbox, mas um
  socket fora dele ficava ao alcance: `$XDG_RUNTIME_DIR/dcode.sock`, o default
  até o #407, ou um `DCODE_SOCKET` escolhido.
- **Entre sessões**, cobrir só o socket da própria sessão não bastaria: o
  daemon embutido de outra TUI, ou o `dcode serve`, continuariam ao alcance.

O changelog do N1 (`client-server-protocol`,
`202609292355-o-socket-e-um-por-usuario.md`) afirmou que no macOS o perfil
nega socket unix de qualquer jeito. Não nega: libera onde se pode escrever. A
frase foi corrigida lá, apontando para este.

## O que mudou

- **`sandbox.Config.Daemons`** diz onde os daemons do usuário escutam: o socket
  da sessão, o de `DCODE_SOCKET` e a pasta por usuário `/tmp/dcode-<uid>/`. Tudo
  isso fica fora do alcance em todo modo abaixo do acesso total.
- **No macOS**, é negado por último, depois de toda liberação, porque o Seatbelt
  fica com a última regra que casa. Vai nas duas grafias, porque `/tmp` é link
  para `/private/tmp`.
- **No Linux**, é coberto como um socket de contêiner — `/dev/null` sobre um
  socket, `tmpfs` sobre uma pasta —, por cima de qualquer concessão. O que está
  sob `/tmp`, sem nada que o traga de volta (o workspace, um caminho gravável),
  já some no `tmpfs` e não é montado por cima.
- **O daemon embutido da TUI** passa a escutar na pasta por usuário
  (`/tmp/dcode-<uid>/tui-*/`), para a mesma negação cobrir todos.
- **Conceder** um desses lugares em `sandbox.sockets` não abre nada, e o daemon
  diz isso ao subir, em vez de deixar uma concessão que não faz efeito.
- **A configuração do sandbox** é montada num lugar só (`sandboxConfig`), para
  a sessão e para a medição dos critérios, que antes repetiam a lista.

## Um defeito que o teste expôs

O primeiro teste que rodou um comando de verdade pelo sandbox de uma sessão, num
Linux com Docker — o runner do CI —, falhou antes de rodar qualquer coisa:

```
bwrap: Can't create file at /var/run/docker.sock: No such file or directory
```

No Ubuntu, `/var/run` é link para `/run`, e o bubblewrap segue link no destino de
uma montagem a partir da raiz dele, não da do sandbox: o destino não existe para
ele, e o comando inteiro cai. Todo comando confinado, numa máquina assim, teria
caído. O comentário no código dizia o contrário — que o destino ia sem resolver
porque o bubblewrap resolvia sozinho.

Agora cada destino é coberto no caminho resolvido, e uma vez só: a raiz do
sandbox é a do host, então o caminho resolvido é o mesmo arquivo lá dentro, e
dois nomes do mesmo socket são uma montagem. Reivindicado por
`TestASocketNamedThroughALinkIsCoveredWhereItResolves`.

## Por que todos os daemons, e não só o da sessão

Qualquer daemon do usuário abre sessão em acesso total para quem pedir pelo
socket; o `0700` separa usuários, não sessões. Uma sessão confinada que alcança
o daemon de outra está tão solta quanto a que alcança o próprio.

## Por que negar a conexão, e não esconder o arquivo

No macOS a leitura é livre por projeto — recusá-la impede o interpretador de
carregar —, então o arquivo do socket continua visível. O que se nega é
conectar: `network-outbound` sobre o caminho. No Linux o socket coberto deixa
de ser socket, e conectar falha no kernel.

## As invariantes

As oito linhas novas em §6 são reivindicadas por:

- `TestAConfinedCommandCannotReachItsDaemon` (`internal/app`), pela montagem do
  próprio daemon;
- `TestADaemonIsDeniedAfterEveryAllowInTheProfile` e
  `TestADaemonIsCoveredInTheArgumentsEvenWhenGranted` (`internal/sandbox`), no
  texto do perfil e nos argumentos;
- `TestADaemonSocketIsOutOfReachEvenWhereWritingIs` (macOS) e
  `TestARealDaemonSocketIsCoveredEvenWhenGranted` (Linux), no kernel;
- `TestAGrantOfADaemonSocketIsSaidAtBoot` e
  `TestAnEmbeddedDaemonListensBesideTheOthers` (`internal/app`);
- `TestASocketNamedThroughALinkIsCoveredWhereItResolves` (`internal/sandbox`).

É correção de fronteira: PATCH.

## O que fica de fora

- Um `DCODE_SOCKET` que só outro processo conhece, e esta sessão não, não é
  coberto por nome. Fica fora do alcance só se estiver na pasta por usuário, ou
  sob `/tmp` no Linux.
- `full-access` não promete fronteira, e não mantém uma.
