## en

- **The desktop acts on its orchestrator** (D66), closing what D64 left open.
  The Agentes tab's "◆ N perguntas" queue now answers: one button per option
  the daemon sent — merge, return or close a merge, continue or stop at the
  token ceiling —, the recommended one in the accent, and an optional note on
  a merge or design question. The question leaves when the daemon takes the
  answer, or when a `question.answered` from another client arrives; the
  first answer wins, and `question_already_answered` is said on the question.
  The Crew's "Nova" menu gains "◆ orquestradora", beside "com modelo e modo"
  and "↻ loop": choosing a project opens its orchestrator through
  `POST /v1/projects` and `GET /v1/projects/{id}/orchestrator`, and
  `orchestrator_unavailable` is said with the keys to set. It hides while the
  loop is chosen, as "Outra pasta…" does, because that menu is measured
  reference 11. In the Sessões list a child of an orchestrator goes under its
  project — up its parents, else by `project_id` — and is indented under its
  parent, instead of each worktree showing as a project of its own.
  `GET /children` and `GET /questions` feed the list when an orchestrator's
  conversation opens and each time its stream goes quiet, so a closed child
  leaves when the daemon's retention says so. The main process checks every
  argument of the five new routes before it sends anything.

## pt-BR

- **O desktop age sobre a orquestradora** (D66), fechando o que a D64 deixou
  em aberto. A fila "◆ N perguntas" da aba Agentes agora responde: um botão
  por opção que o daemon mandou — merge, devolver ou fechar num merge;
  continuar ou parar no teto de tokens —, a recomendada no acento, e uma nota
  opcional numa pergunta de merge ou de desenho. A pergunta sai quando o
  daemon toma a resposta, ou quando chega o `question.answered` de outro
  cliente; a primeira resposta vale, e `question_already_answered` é dito na
  própria pergunta. O menu "Nova" da Crew ganha "◆ orquestradora", ao lado de
  "com modelo e modo" e de "↻ loop": escolher um projeto abre a orquestradora
  dele por `POST /v1/projects` e `GET /v1/projects/{id}/orchestrator`, e
  `orchestrator_unavailable` é dito com as chaves a ligar. Ele some com o
  loop ligado, como "Outra pasta…", porque esse menu é a referência 11, medida.
  Na lista de Sessões um filho de orquestradora vai para o projeto dela —
  subindo pelos pais, senão pelo `project_id` — e fica recuado sob o pai, em
  vez de cada worktree aparecer como um projeto à parte. `GET /children` e
  `GET /questions` alimentam a lista ao abrir a conversa orquestradora e cada
  vez que o fluxo dela fica quieto, de modo que um fechado sai quando a
  retenção do daemon diz. O processo principal confere cada argumento das
  cinco rotas novas antes de enviar qualquer coisa.
