## en

- **A turn carries documents: a PDF, or a text file.** `SubmitTurnRequest`
  gains `documents` — name, media type and base64 bytes — checked where they
  arrive: a closed set of types (`application/pdf`, `text/plain`,
  `text/markdown`, `text/csv`), a PDF by its `%PDF-` bytes and never its name,
  text as UTF-8 without NUL, a file name that is not a path, and limits
  declared in the protocol (5 per turn, 10 MiB a PDF, 256 KiB of text). Each
  family declares whether it reads a PDF, like it declares pictures, and only
  `claude` says yes, as a `document` block; a PDF for any other model refuses
  the turn at submit with `invalid_input` naming the model, and a dialect that
  cannot carry one refuses to encode rather than send the question without
  it. Text goes to every family, written ahead of the question between markers
  that name the file and carry a boundary taken from its bytes, so a body
  holding `</document>` cannot close its own block. A PDF costs context by its pages. `turn.started` names
  the files — name, type, size, never the bytes — so history shows what was
  attached, and `pkg/client` gains `SubmitTurn`. Before, a document reached
  the model only by pasting it or naming a path the daemon might not have.

## pt-BR

- **Um turno carrega documentos: um PDF, ou um arquivo de texto.**
  `SubmitTurnRequest` ganha `documents` — nome, tipo e bytes em base64 —
  conferidos onde chegam: um conjunto fechado de tipos (`application/pdf`,
  `text/plain`, `text/markdown`, `text/csv`), PDF pelos bytes `%PDF-` e nunca
  pelo nome, texto UTF-8 sem NUL, nome de arquivo que não é caminho, e limites
  declarados no protocolo (5 por turno, 10 MiB por PDF, 256 KiB de texto). Cada
  família declara se lê PDF, como declara imagem, e só `claude` diz sim, como
  bloco `document`; PDF para qualquer outro modelo recusa o turno na submissão
  com `invalid_input` nomeando o modelo, e dialeto que não carrega PDF recusa
  codificar em vez de mandar a pergunta sem ele. Texto vai para toda família,
  escrito antes da pergunta entre marcadores que nomeiam o arquivo e levam uma
  fronteira tirada dos bytes dele, para um corpo com `</document>` não fechar o
  próprio bloco. PDF pesa no
  contexto pelas páginas. `turn.started` nomeia os arquivos — nome, tipo,
  tamanho, nunca os bytes — para o histórico mostrar o que foi anexado, e
  `pkg/client` ganha `SubmitTurn`. Antes, um documento só chegava ao modelo
  colado, ou por um caminho que o daemon podia não ter.
