## en

- **Undo never removes a file it could not read before the turn.** A snapshot
  whose read failed for any reason but absence was recorded as "did not exist",
  so `/undo` and the loop's automatic cycle undo removed a file that existed
  and was merely unreadable. Now only absence counts as absence: an unread
  snapshot is refused and named by the undo, and `write` and `edit` warn when it
  happens, as the recoverable-cycle RN-6 always said.

## pt-BR

- **O desfazer nunca remove um arquivo que não conseguiu ler antes do turno.**
  Instantâneo cuja leitura falhava por qualquer motivo que não ausência era
  registrado como "não existia", e o `/undo` e o desfazer automático do ciclo
  removiam um arquivo que existia e só não pôde ser lido. Agora só ausência é
  ausência: instantâneo não lido é recusado e nomeado pelo desfazer, e `write`
  e `edit` avisam na hora, como a RN-6 do `recoverable-cycle` sempre disse.
