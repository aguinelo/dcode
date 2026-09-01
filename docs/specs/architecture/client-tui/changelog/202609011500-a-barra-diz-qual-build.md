# A barra diz qual build

**2026-09-01** — a barra de status passa a mostrar a versão.

## Por quê

Um build local e um release **se comportam diferente e se apresentavam igual**.
Toda a semana foi testando binário local contra o publicado, e a única forma de
saber qual estava rodando era sair da sessão e perguntar ao `--version`.

A string de versão já dizia o que precisava — `0.18.0-dev+836d4c2`, com o
`-dev+sha` que o produto acrescenta de propósito para que build local não se
disfarce de release. Ela só não estava na tela.

## Onde, e o que cai primeiro

Ao lado do nome, esmaecida. E é o **primeiro campo entregue** quando o terminal
estreita, porque responde uma pergunta que ninguém faz no meio da sessão — e é a
primeira que se quer quando duas versões estão em jogo.

A ordem de descarte já era um conceito da barra, e ela tem dois sentidos que não
são o mesmo: ordem de leitura põe o modelo ao lado do nome; ordem de descarte
entrega o modelo antes. O build entra depois do modelo no descarte, e o modo de
sandbox continua não caindo nunca — ele não é informação, é indicador de
segurança.

## Invariantes

- `TestTheStatusBarSaysWhichBuild`
- `TestTheBuildIsTheFirstFieldToGo` — e afirma também que o modo de sandbox
  sobrevive ao estreitamento que derruba o build.
