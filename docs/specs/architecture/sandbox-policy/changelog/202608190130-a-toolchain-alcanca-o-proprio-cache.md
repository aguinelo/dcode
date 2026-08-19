# Uma toolchain alcança o próprio cache

**Data:** 2026-08-19

## O que mudou

`workspace-write` passa a conceder escrita nos diretórios que uma toolchain
precisa para compilar e que ficam **fora** do workspace por desenho: o cache de
build, o cache de módulos e o temporário do usuário.

Nomeados um a um. **O home nunca é concedido** — a diferença entre conceder um
cache e conceder a máquina de alguém.

## Por que

Medido, não suposto. Uma execução autônoma neste repositório mudou arquivos e
**não conseguiu executar um único teste**. A primeira falha:

```
open ~/Library/Caches/go-build/04/04b498…-d: operation not permitted
FAIL ./internal/specguard [setup failed]
```

Um sandbox que permite editar e proíbe conferir produz mudança não verificada,
que é exatamente o que a definição de pronto existe para impedir. E não é
específico deste repositório: qualquer projeto Go, Rust, Node ou Java bate no
mesmo lugar, porque o cache é compartilhado entre projetos e grande demais para
morar dentro de um.

## O segundo bloqueio, encontrado depois de consertar o primeiro

Com o cache liberado, `go test` foi mais longe e parou de novo — agora montando
a compilação em `$TMPDIR`, que **no macOS não é `/tmp`**: é `/var/folders/…/T`.
A lista já concedia `/tmp` e `/var/tmp` e parava antes desse.

Consertar só o cache **moveu** a falha em vez de removê-la, e foi só rodando de
novo que isso apareceu. Fica registrado porque é o modo de errar deste tipo de
correção: a primeira causa encontrada raramente é a única.

Por isso o conjunto passou a se chamar `Scratch` e não `Caches` — um temporário
não é cache, e um nome que descreve metade do conteúdo é um nome que engana o
próximo leitor.

## O que não mudou

`read-only` não concede nenhum deles, nos dois backends. O modo tem de
significar a mesma coisa independentemente da plataforma, ou passa a significar
"depende de onde você está".

## O que isto custa, dito por inteiro

Escrever no cache de build é capacidade real: um comando poderia envenená-lo, e o
efeito atravessa a fronteira, porque compilações futuras fora do sandbox leem o
mesmo cache. É o preço, e ele é menor que a alternativa — um agente que altera
código e nunca consegue verificá-lo.

## De onde sai a lista

Do ambiente onde a própria toolchain publica a resposta — `GOCACHE`,
`GOMODCACHE`, `CARGO_HOME`, `npm_config_cache`, `XDG_CACHE_HOME`, `TMPDIR` — e do
lugar padrão de cada uma quando nada foi dito. A máquina com `GOCACHE` num lugar
incomum é justamente a que seria negada.

Ambiente ausente concede **nada**, e ambiente nulo também não derruba a sessão:
mesma regra que a decisão de rede nula já seguia. Nada dito nunca é um sim.
