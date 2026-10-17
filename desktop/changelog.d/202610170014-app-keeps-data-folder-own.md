## en

- **The app keeps its data in a folder of its own.** Electron's default is the
  product name in the system's data folder, `~/Library/Application Support/DCode`
  on macOS — and the file system there does not tell `DCode` from `dcode`, the
  folder dcode keeps its configuration, records and state in. Every `npm start`
  wrote Chromium's storage, caches and cookies beside `models.toml`, and deleting
  the app's data would have taken dcode's configuration with it. The app now uses
  `dcode-desktop` there, and `DCODE_DESKTOP_USER_DATA` names another, absolute
  folder — which the check already passed and the app ignored. What earlier runs
  left in dcode's folder is Chromium's and can be deleted by hand; dcode's own
  entries there are `doctrine`, `models.toml`, `sessions` and
  `update-check.json`. `docs/DECISIONS.md` D26.

## pt-BR

- **O app guarda os dados numa pasta só dele.** O padrão do Electron é o nome do
  produto na pasta de dados do sistema, `~/Library/Application Support/DCode` no
  macOS — e o sistema de arquivos ali não distingue `DCode` de `dcode`, a pasta em
  que o dcode guarda configuração, registros e estado. Cada `npm start` escrevia o
  armazenamento, os caches e os cookies do Chromium ao lado do `models.toml`, e
  apagar os dados do app levaria a configuração do dcode junto. O app agora usa
  `dcode-desktop` ali, e `DCODE_DESKTOP_USER_DATA` escolhe outra pasta, absoluta —
  que a régua já passava e o app ignorava. O que as execuções anteriores deixaram
  na pasta do dcode é do Chromium e pode ser apagado à mão; o que é do dcode ali é
  `doctrine`, `models.toml`, `sessions` e `update-check.json`. `docs/DECISIONS.md`
  D26.
