## en

- **`generated.ts` carries documents on a turn.** The core's
  `SubmitTurnRequest` gains `documents` (`TurnDocument`: name, media type,
  base64), `TurnStarted` gains `documents` (`AttachedDocument`: name, media
  type, size), and the accepted types and limits are constants. Regenerated,
  and the event decoder accepts `turn.started.documents` as optional; nothing
  in the app sends or draws them yet — the composer does in its own
  pull request.

## pt-BR

- **O `generated.ts` carrega documentos num turno.** O `SubmitTurnRequest` do
  núcleo ganha `documents` (`TurnDocument`: nome, tipo, base64), o
  `TurnStarted` ganha `documents` (`AttachedDocument`: nome, tipo, tamanho), e
  os tipos aceitos e os limites são constantes. Regenerado, e o decodificador
  de eventos aceita `turn.started.documents` como opcional; nada no app manda
  nem desenha isso ainda — o composer faz isso no PR dele.
