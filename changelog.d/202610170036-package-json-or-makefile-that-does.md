## en

- **A `package.json` or `Makefile` that does not parse is said.** The probe
  read a malformed manifest as "declares nothing", silently, so the prompt
  looked exactly like a project with no gate. The session now warns at open
  (`manifest_unreadable`, naming the file and the error); an absent manifest
  still says nothing, and one unreadable file no longer hides the other.

## pt-BR

- **Um `package.json` ou `Makefile` que não se lê é dito.** A sondagem lia
  um manifesto malformado como "não declara nada", em silêncio, e o prompt
  ficava igual ao de um projeto sem portão. A sessão agora avisa ao abrir
  (`manifest_unreadable`, nomeando o arquivo e o erro); manifesto ausente
  continua sem dizer nada, e um arquivo ilegível não esconde mais o outro.
