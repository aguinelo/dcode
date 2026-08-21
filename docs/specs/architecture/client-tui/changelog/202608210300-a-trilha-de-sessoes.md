# A coluna lista as conversas deste workspace

**Data:** 2026-08-21
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** `refs/design/HANDOFF.md` (v5, §2)

## O que mudou

Sob os arquivos, a coluna lateral passa a listar as conversas gravadas deste
workspace, com a aberta marcada por `●` — caractere, não só cor.

É o `dcode -r` promovido a coluna permanente, no modo que o design chama de
**passiva**.

## Mesma fonte, mesmo filtro

A lista vem de `session.Browse` e passa pelo mesmo `choicesFrom` que o `-r` usa,
lida uma vez no início pela borda. Duas maneiras de listar as conversas de um
workspace acabariam discordando sobre quais existem.

Conversa em que nada foi perguntado continua fora, e isso é a maior parte do que
um diretório de gravação guarda: um registro é escrito toda vez que a interface
abre. Enterrar quatro conversas reais sob trinta vazias é o que o picker já
recusa a fazer.

O cliente não lê disco. A lista chega por `Options`, como o idioma e o conjunto
de comandos.

## Dois modos ficaram de fora, e um deles não podia entrar

O design descreve três modos: passiva, navegando (`^R`) e **nomeando**.

Nomear **não tem onde ser guardado**. O nome que a pessoa dá precisa sobreviver à
sessão, e um diretório de gravação guarda transcrições, não títulos — o
`Summary.Title` é derivado da primeira pergunta toda vez que é lido. É mudança em
`internal/session`, não no cliente, e está no `docs/ROADMAP.md` com as três
formas possíveis de guardá-lo e o motivo de uma delas sobreviver à poda.

Navegar com `^R` espera em parte pela mesma decisão: mover cursor e continuar
conversa já são possíveis, e o `/resume` já faz o continuar. O que o `^R`
acrescenta é fazer isso sem sair do teclado, e vale quando houver o que nomear.

## Detalhes que a tela decidiu

- **Conversas sozinhas já abrem a coluna.** Perguntar só pelos arquivos a
  esvaziava no primeiro minuto de toda sessão, quando nada foi tocado ainda.
- **Título cortado diz que foi cortado.** Um que apenas para deixa o leitor sem
  distinguir conversa curta de conversa truncada. E o corte é em **células**, não
  em bytes — runa não é coluna, e é em título com acento que isso erra.
- **A lista cede primeiro** quando a coluna fica sem altura: lista truncada de
  conversas passadas custa menos que lista truncada do que está sendo escrito
  agora.
