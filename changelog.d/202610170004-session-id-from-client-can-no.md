## en

- **A session id from a client can no longer name a file outside the record
  directory.** Naming a conversation that is not loaded and continuing one
  (`resume`) now hold the id to the rule reading a recorded conversation
  already used — letters, digits, `-` and `_` — and refuse anything else with
  `invalid_input` before touching the disk. Both joined the id into a path
  unchecked: `POST /sessions/..%2Fsecret/name` appended a line to a record one
  directory up, and `resume: "../secret"` read one into the model's context
  and copied it into the new session's record.

## pt-BR

- **Um id de sessão vindo de um cliente não nomeia mais arquivo fora da pasta
  de registros.** Nomear uma conversa que não está carregada e continuar uma
  (`resume`) passam a seguir a regra que ler uma conversa gravada já seguia —
  letras, dígitos, `-` e `_` — e recusam o resto com `invalid_input` antes de
  tocar o disco. Os dois juntavam o id ao caminho sem conferir:
  `POST /sessions/..%2Fsecret/name` acrescentava uma linha a um registro um
  nível acima da pasta, e `resume: "../secret"` lia um para o contexto do
  modelo e o copiava para o registro da sessão nova.
