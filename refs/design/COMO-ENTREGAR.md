# Como entregar isto ao Claude Code

> **Executado em 2026-08-20.** O material vive em `refs/design/`, versionado, e
> o handoff é o `HANDOFF.md` desta pasta. Os caminhos abaixo foram corrigidos
> para onde as coisas de fato ficaram; o resto é como o autor escreveu, e a
> ordem da seção 4 é a proposta dele.
>
> As divergências entre este material e o repositório estão no `README.md` desta
> pasta — não foram aplicadas aqui nem no handoff, de propósito.

## 1. Coloque a pasta no repositório

Copie a pasta inteira (esta, com o `README.md` e os três `.dc.html`) para dentro
do repo do dcode:

```
dcode/refs/design/
```

Faça commit. Assim a referência de design fica versionada junto do código que
ela descreve, e qualquer sessão futura acha o arquivo sem você explicar nada.

## 2. Abra o Claude Code na raiz do repo

```
cd ~/work/dcode
claude
```

## 3. Cole este prompt

> Leia `refs/design/HANDOFF.md` — é a referência de design da
> TUI do DCode. Os `.dc.html` na mesma pasta são maquetes em HTML, não código
> para copiar: a implementação é Go + Bubble Tea v2 em `internal/tui`.
>
> Antes de escrever código, a seção final do README lista quatro acréscimos que
> ainda não existem na spec (árvore de arquivos na coluna lateral, trilha de
> sessões permanente + `^N` nomear, tarefa como card com progresso, verbo de
> atividade) e uma colisão de tecla (`^E` já é "fim da linha"). Escreva o
> changelog de cada um em `docs/specs/architecture/client-tui/` seguindo o
> formato dos changelogs que já estão lá, e resolva a colisão de tecla na spec.
> Só depois implemente.
>
> Comece me mostrando o plano: quais changelogs, em que ordem, e o que muda em
> `model.go`, `render.go` e `style.go`. Não escreva código ainda.

## 4. Depois do plano, uma coisa por vez

Peça na ordem em que o README as apresenta — cada uma cabe num turno com a
definição de pronto rodando no fim:

1. `Model.tree` + a seção ARQUIVOS (redutor puro sobre o log de eventos)
2. `Model.rail` + a trilha de sessões e seus três modos
3. Card de tarefa com progresso vindo do metadado da ferramenta
4. Barra de atividade com o verbo
5. Modo ASCII / `NO_COLOR` para tudo que foi adicionado

## Duas coisas que valem repetir para ele

- **Estilo nunca altera largura medida**, e monocromático não emite escape
  algum. Toda medida usa `lipgloss.Width` / `go-runewidth`, nunca `len()`.
- **Animação é `tea.Tick` só enquanto o turno roda.** Ocioso: nenhum tick,
  nenhum quadro. Onde a maquete usa opacidade, o terminal usa degrau de cor.

## Se preferir não commitar a pasta

Cole o conteúdo do `HANDOFF.md` direto no chat e diga: *"esta é a referência de
design; implemente em `internal/tui`"*. Funciona — só perde o versionamento e a
possibilidade de outra sessão retomar de onde a anterior parou.
