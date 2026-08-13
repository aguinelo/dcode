# Suíte de integração

Roda contra um Postgres real. **Nada neste repositório sobe esse banco** — a
esteira o provisiona antes de chamar a suíte, e localmente é preciso ter um
instância própria em `localhost:5432`.

Sem o banco, `TestAccountsRoundTrip` falha em `Ping`. Não há caminho alternativo:
a suíte existe justamente para exercitar o driver de verdade.
