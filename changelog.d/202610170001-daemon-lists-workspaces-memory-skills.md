## en

- **The daemon lists a workspace's memory and skills.** `GET /v1/memory`
  answers what a session in a workspace reads as memory: every entry of
  `.dcode/memory.md` with its kind, subject, body and provenance, marked stale
  and shown as the session's prefix marks and shows it, and every block that is
  not a memory with what is wrong in it — the reason now comes from the memory
  package, which only kept the line. `GET /v1/skills` answers the skills a
  session there has, the user's and the project's, each with what it declares
  about itself and where it comes from, a held one marked held, and the files
  that did not load with why. Both read through the calls sessions use, never
  carry a key or a skill's body, and refuse a missing, relative or unreadable
  workspace with the reason. Before, a client learned either only by reading
  the files itself and repeating five rules that live in the daemon. This is
  what the desktop's crew view reads.

## pt-BR

- **O daemon lista a memória e as skills de um workspace.** `GET /v1/memory`
  responde o que uma sessão num workspace lê como memória: cada entrada do
  `.dcode/memory.md` com tipo, assunto, corpo e procedência, marcada velha e
  mostrada como o prefixo da sessão a marca e a mostra, e cada bloco que não é
  memória com o que há de errado nele — o motivo agora sai do pacote de memória,
  que só guardava a linha. `GET /v1/skills` responde as skills que uma sessão
  ali tem, as do usuário e as do projeto, cada uma com o que declara de si e de
  onde vem, a retida marcada como retida, e os arquivos que não carregaram com o
  motivo. As duas leem pelas chamadas que a sessão usa, nunca carregam a chave
  nem o corpo de uma skill, e recusam workspace ausente, relativo ou ilegível
  com o motivo. Antes, um cliente só sabia de uma ou de outra lendo os arquivos
  por conta própria e repetindo cinco regras que moram no daemon. É o que a
  visão de equipe do desktop lê.
