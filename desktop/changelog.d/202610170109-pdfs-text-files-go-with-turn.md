## en

- **PDFs and text files go with a turn, like pictures (D59, amending D46).**
  The composer's "+" offers pictures, PDFs and text files; pasting or dropping
  a file attaches it. The main process types each one by its bytes — a
  picture, then `%PDF-`, then UTF-8 without NUL, the extension only saying
  which text — and holds it to the daemon's rules from `generated.ts`: 10 MiB a
  PDF, 256 KiB of text, 5 documents a turn besides 20 pictures, a file name and
  never a path, checked again when the turn is sent. A document is a chip with
  its kind, name, size and ×; past a count it is refused by name. The message
  in the flow shows the documents `turn.started` names, so a conversation read
  back from its record shows them too. A PDF for a model that does not read
  one is refused by the daemon at submit, naming the model, and the window says
  so in the field and a notice, keeping the text and the files. Before, the
  "+" took pictures only. Screens 02 and 03 stay at 0.00%; `check:daemon` 13 of
  13.

## pt-BR

- **PDFs e arquivos de texto vão com o turno, como as imagens (D59, emenda
  D46).** O "+" da caixa oferece imagens, PDFs e textos; colar ou arrastar um
  arquivo o anexa. O processo principal tipa cada um pelos bytes — imagem,
  depois `%PDF-`, depois UTF-8 sem NUL, a extensão só dizendo qual texto — e o
  confere pelas regras do daemon, do `generated.ts`: 10 MiB por PDF, 256 KiB de
  texto, 5 documentos por turno além de 20 imagens, nome de arquivo e nunca
  caminho, conferido de novo quando o turno sai. Documento é um chip com tipo,
  nome, tamanho e ×; passou da conta, é recusado pelo nome. A mensagem no fluxo
  mostra os documentos que o `turn.started` nomeia, então uma conversa lida do
  registro também os mostra. PDF para modelo que não lê PDF é recusado pelo
  daemon na submissão, nomeando o modelo, e a janela diz isso no campo e num
  aviso, mantendo o texto e os arquivos. Antes, o "+" só levava imagens. As
  telas 02 e 03 ficam a 0,00%; `check:daemon` 13 de 13.
