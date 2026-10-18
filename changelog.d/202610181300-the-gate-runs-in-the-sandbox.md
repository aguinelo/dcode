## en

- **The orchestrator's gate runs inside the sandbox.** The gate command, and a
  child's red test, run wrapped by the same sandbox the agent's bash uses, in
  `workspace-write` with the measurement's throwaway directory — the worktree
  and a `TMPDIR` of its own — as the workspace: it writes there and in the
  toolchain caches the policy already grants, and nowhere else. The network is
  open only when configuration opens it, so `make check`-like gates run offline;
  a standing grant answers a session, and a gate is none. A sandbox that cannot
  be established, backend `none` included, is a gate that does not run, with
  an error saying the sandbox is why — never a gate without a boundary.
  `GateResult` records `sandbox` and `sandbox_mode` as evidence. Before, the
  gate ran `/bin/sh` unconfined, and a gate that wrote to the person's
  repository or home measured green.

## pt-BR

- **O gate da orquestradora roda dentro do sandbox.** O comando do gate, e o
  teste vermelho de um filho, rodam embrulhados pelo mesmo sandbox do `bash` do
  agente, em `workspace-write` com o diretório descartável da medição — a
  worktree e um `TMPDIR` só dela — como workspace: grava ali e nos caches de
  toolchain que a política já concede, e em nenhum outro lugar. A rede só abre
  quando a configuração a abre, então gates como `make check` rodam sem rede; a
  concessão permanente responde a uma sessão, e o gate não é sessão. Sandbox
  que não se estabelece, backend `none` incluído, é gate que não roda, com erro
  dizendo que o sandbox é o motivo — nunca gate sem fronteira. O `GateResult`
  registra `sandbox` e `sandbox_mode` como evidência. Antes, o gate rodava
  `/bin/sh` sem confinamento, e um gate que gravava no repositório da pessoa ou
  no home media verde.
