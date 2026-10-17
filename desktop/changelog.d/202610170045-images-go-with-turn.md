## en

- **Images go with a turn.** The daemon has taken pictures on a turn since
  `SubmitTurnRequest.images`, and the TUI sends them; the window sent text only,
  and its "+" said attaching did not exist yet. Now "+" opens the system's
  picker (images only), a paste with an image on the clipboard attaches it, and
  files dropped on the field attach. The main process reads a picked file and
  checks pasted or dropped bytes the same way — the type from the bytes (PNG,
  JPEG, GIF, WebP), up to 10 MiB each and 20 per turn — and checks them again
  when the turn is sent; the renderer never sees a path. Thumbnails sit above
  the field, each with its ×, kept per conversation like the draft. A refusal
  is a notice and a line under the thumbnails, with the reason. A correction
  carries text only, so a message with images during a turn is held and said
  why, never steered or queued. The message in the flow shows the images it
  went with; the events carry none, so a conversation read from its record
  shows the text alone. Nothing at rest changes: both visual references
  measure 0.00% (D46).

## pt-BR

- **Imagens vão com o turno.** O daemon aceita imagens num turno desde
  `SubmitTurnRequest.images`, e a TUI as manda; a janela mandava só o texto, e o
  "+" dela dizia que anexar ainda não existia. Agora o "+" abre o seletor do
  sistema (só imagens), colar com uma imagem na área de transferência a anexa,
  e arquivos soltos sobre o campo são anexados. O processo principal lê o
  arquivo escolhido e confere do mesmo jeito os bytes colados ou soltos — o tipo
  pelos bytes (PNG, JPEG, GIF, WebP), até 10 MiB cada e 20 por turno — e confere
  de novo quando o turno é enviado; o renderer nunca vê um caminho. As
  miniaturas ficam acima do campo, cada uma com seu ×, guardadas por conversa
  como o rascunho. Uma recusa vira um aviso e uma linha abaixo das miniaturas,
  com o motivo. Uma correção leva só texto, então uma mensagem com imagens
  durante um turno fica guardada, com o porquê, e nunca vira correção nem entra
  na fila. A mensagem no fluxo mostra as imagens que foram com ela; os eventos
  não as carregam, então uma conversa lida do registro mostra só o texto. Nada
  em repouso muda: as duas referências visuais medem 0,00% (D46).
