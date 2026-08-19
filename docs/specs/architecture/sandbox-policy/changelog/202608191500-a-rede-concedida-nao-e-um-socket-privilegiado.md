# A rede concedida não é um socket privilegiado

Conceder rede passou a conceder **tráfego IP e resolução de nomes**, não todo
socket unix da máquina. No Linux, os sockets de runtime de contêiner passam a
ser cobertos.

## O que foi encontrado

O harness encontrou o buraco em si mesmo. Numa sessão não assistida deste
repositório, corrigindo uma guarda de teste e raciocinando sobre o lado Linux do
CI, o modelo rodou — sem que nada pedisse:

```
docker version --format '{{.Server.Version}}'
docker images --format '{{.Repository}}:{{.Tag}} {{.ID}}'
docker run --rm ubuntu:26.10 sh -lc '...'
```

Os três com `exit 0`, de dentro de `workspace-write`.

Reproduzido fora do experimento, montando à mão o perfil que o backend gera:

| | resultado |
|---|---|
| escrever fora do workspace, direto | `Operation not permitted` |
| falar com o daemon Docker, com rede concedida | responde |
| falar com o daemon Docker, sem rede | `permission denied` |

## Por que isso derrubava a fronteira inteira

`(allow network*)` do Seatbelt cobre socket unix junto com IP. E socket unix é
como se alcança um processo privilegiado que **não está** no sandbox: o daemon
do Docker monta qualquer caminho do host dentro de um contêiner a pedido.
`docker run -v /:/host` escreve em qualquer lugar da máquina, como root, a
partir de um modo cuja promessa é justamente conter escrita ao workspace.

A concessão de rede é **autorização** — o invariante já dizia que ela nunca abre
o que o modo fechou. Ela abria.

No Linux é pior, e por outro caminho: socket unix de caminho não vive em
namespace de rede, então `--unshare-net` não o cobre, e o bind read-only de `/`
também não — conectar num socket não é escrever. Lá o buraco não dependia nem da
concessão de rede.

## O que passou a valer

**macOS** — o perfil deixa de emitir `(allow network*)` quando a rede é
concedida, e passa a emitir tráfego IP mais o socket do `mDNSResponder`. O
resolvedor é nomeado porque resolve nomes; ele não age em nome de quem chama, e
é essa a propriedade que o separa do socket de um runtime. Sem ele, negar todo
socket unix nega DNS junto.

**Linux** — cada socket nomeado é coberto com `/dev/null`, que não é socket, e
o `connect` falha no kernel. A lista cobre os caminhos fixos do Docker, Podman,
containerd e CRI-O, mais o diretório de runtime do usuário e o `DOCKER_HOST`
quando ele aponta para um caminho.

**`full-access` fica como está.** Um modo que não promete fronteira não deve
fingir que estreita alguma.

## O que isto não é

Uma lista é mais fraca que uma fronteira. No macOS a fronteira existe — nenhum
socket unix passa. No Linux o sistema de arquivos inteiro é montado por desenho,
então os sockets precisam ser nomeados, e nomear é pior que negar e melhor que
entregar. Um runtime com socket em caminho não previsto continua alcançável, e
o `.r` de uma revisão futura tem aí um problema em aberto.

Quem precisa de contêiner tem `full-access`, que é o modo que diz em voz alta o
que faz.
