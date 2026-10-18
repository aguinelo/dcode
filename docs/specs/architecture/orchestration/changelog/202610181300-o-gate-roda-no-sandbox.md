# O gate roda dentro do sandbox

**Data:** 2026-10-18
**Specs afetadas:** `202610132000-orchestration` (`.p` §5.2 `GateResult`, §7.4, invariantes; `.i` passos 9 e 14).

## O que aconteceu

O passo 9 ficou "feito, menos o sandbox do `bash` em volta do gate". O `Gate.Measure` rodava
`/bin/sh -c` sem fronteira nenhuma: numa worktree descartável, sem `GIT_*`, mas com o poder de gravar
onde o daemon grava. Um gate que escrevia no repositório da pessoa ou no home saía **verde**, e a
medição em que um merge se decide vinha de um comando que podia mexer na árvore que dizia não tocar.

## O que mudou

- O comando do gate roda embrulhado pelo `sandbox.Sandbox` — o mesmo que o `bash` do agente usa —,
  com o diretório descartável inteiro (a worktree e um `tmp` só da medição) como workspace. O shell
  entra na worktree antes do comando; o `TMPDIR` do comando é o `tmp` da medição.
- O daemon monta esse sandbox por medição a partir da configuração da raiz do projeto
  (`Daemon.gateSandbox`): backend, caches de toolchain, caminhos ilegíveis, daemons fora de alcance.
  Fica de fora o que `sandbox.writable` nomeia — como o filho, que também não o tem — e o `TMPDIR`
  do daemon (`GateScratch`).
- Sandbox que não se estabelece — indisponível, backend `none`, nenhum — é erro que diz que o sandbox
  é o motivo (`sandbox.ErrUnavailable`), antes de qualquer worktree: nada medido, nada segue. Nunca
  um gate sem fronteira.
- `GateResult` ganha `sandbox` e `sandbox_mode`: a medição nomeia a fronteira em que rodou, como
  evidência ao lado do comando.
- O teste da fase `red` roda na mesma fronteira do gate.
- `Gate` sem `Sandbox` usa `DefaultGateSandbox`: o backend da plataforma, sem rede. Nil é a fronteira
  fechada, nunca nenhuma.

## Decisões

**O modo é `workspace-write`, sempre.** A `.p` não decidia. `read-only` não roda um gate — compilar
e testar grava —, e `full-access` é não ter fronteira. O modo é o do gate, não o das sessões do
projeto: a orquestradora está em `read-only` e o filho em `workspace-write`, e o gate é uma medição
do harness, não trabalho de nenhuma das duas.

**A rede é só a da configuração.** O `bash` da sessão abre a rede por `sandbox.network` **ou** pela
concessão permanente que a pessoa deu ao responder uma travessia. O gate segue a primeira e não a
segunda: a concessão é resposta a uma sessão que perguntou, e o gate não é sessão — é o mesmo
raciocínio do `qualifyingSandbox` do pronto, que mede critérios só com `opts.AllowNetwork`. Um gate
como `make check` roda sem rede; quem precisa dela a declara na configuração, que é revisada.

**O workspace é o diretório descartável, não só a worktree.** No Linux o `/tmp` do bubblewrap é um
tmpfs novo, e um `TMPDIR` fora do que foi montado não existiria lá dentro; no macOS o `TMPDIR` do
daemon é o de todo o resto. Um `tmp` irmão da worktree, dentro do mesmo diretório, é gravável nos
dois, e só dele.
