## en

- **Run from the sources on macOS, the menu bar and the Dock say DCode Box.**
  `npm start` now first runs `scripts/dev-name.mjs`, which writes the name into
  the development Electron's Info.plist (in node_modules) and signs that bundle
  again ad hoc, since macOS reads those two places from the bundle and not from
  `app.setName`. It does nothing when the name is already there, and `npm ci`
  undoing it is fixed on the next start. A failure is said and never stops the
  app. A packaged app is untouched.

## pt-BR

- **Rodando a partir do código no macOS, a barra de menus e o Dock dizem DCode
  Box.** O `npm start` agora roda antes o `scripts/dev-name.mjs`, que grava o
  nome no Info.plist do Electron de desenvolvimento (em node_modules) e assina
  esse pacote de novo, ad hoc, porque o macOS lê esses dois lugares do pacote e
  não do `app.setName`. Não faz nada quando o nome já está lá, e um `npm ci` que
  o desfaça é corrigido no próximo início. Uma falha é dita e nunca impede o
  app de abrir. O app empacotado não muda.
