# Diff renderizado, prioridade no status e autocomplete

**Data:** 2026-08-09
**Specs afetadas:** `202608081250-client-tui` (`.p`), `202608072240-client-server-protocol` (`.p`), `202608072341-tool-suite` (`.p`)

Adoção do handoff de design em `refs/design/`, com uma
divergência deliberada, registrada abaixo.

## O que mudou

### Diff é renderizado, e não custa token

A seção 3.2 já dizia que o diff é o que se revisa. Não existia: `edit` devolvia
prosa (`edited x.go (2 replacements, +24 −2)`) e o cliente mostrava esse texto.

Agora `edit` e `write` produzem diff unificado, que viaja no evento em
`tool.completed.diff`. **Nunca entra no histórico do modelo** — ele escreveu a
edição e já sabe o que mudou, então o diff é do humano e custa zero token.

**Prévia sem pedir, resto sob `Tab`.** Recolhido mostra `DiffPreviewLines`
(8); expandido, `DiffMaxLines` (40). O corte informa quanto falta e como ver:
`⋯ 19 lines · Tab expande`. Truncar dizendo só "truncated" deixa o leitor sem
como julgar se importa.

### Colunas alinhadas

Nome de ferramenta em 6 células, alvo em 26, resumo depois. Resumos irregulares
são lidos um a um, que é exatamente o que uma parede de chamadas não pode ser.

O alvo cede primeiro em terminal estreito, e encurta **pelo começo**: o fim é o
que identifica um arquivo, os diretórios são o que todo mundo no repositório tem
em comum.

### O status tem ordem de descarte declarada

Duas ordens, e não são a mesma. **Leitura:** estado, `dcode`, modelo, modo,
contexto, plano. **Descarte:** modelo primeiro, depois contexto, depois o plano.

**O modo de sandbox não está na ordem de descarte.** É o único campo em que
estar errado é perigoso, e some por último — nunca.

> **Divergência do handoff.** O estado 06 do mock mostra `⠴ dcode ctx 34% …` num
> terminal estreito, sem modelo e **sem o modo**. Isso contraria a regra que o
> próprio documento enuncia duas seções acima. Adotamos o layout do mock e
> invertemos o que ele descarta: o nome do modelo vai primeiro, o modo fica.

### Painel cresce até 34 colunas

Era fixo em 24, então item de plano longo era cortado à toa num terminal largo.
Agora `clamp(PanelMinWidth, largura/4, PanelMaxWidth)` — 16, um quarto, 34.

### Marca colorida no estado vazio

O mascote usa os três âmbares e o terracota da marca, um papel por linha de
voxel. Quatro papéis novos na paleta (`highlight`, `body`, `shadow`, `eye`),
usados **só** ali: no instante em que uma segunda coisa usa o terracota, o olho
para de ser marcador.

### Bloqueio se anuncia no fluxo

Item que **passa a** bloqueado gera nota no fluxo, com o motivo por extenso. Só a
transição, nunca o estado: repetido a cada atualização de plano vira mensagem
que o leitor aprende a pular.

### Autocomplete de `/`

Menu acima da entrada enquanto a linha é um prefixo `/` sem argumento. Embutidos
primeiro — comando de usuário não pode sombrear embutido, então listar
misturado sugeriria uma disputa que não existe.

O menu **é consequência do que está digitado, nunca um modo**. Enquanto aberto
ele é dono de `↑↓`, `Tab` e `Esc`, e de mais nada. `Esc` fecha para a linha como
ela está; qualquer edição revive.

### Fila com remoção

`^X` remove a mais antiga — a que o usuário teve mais tempo para reconsiderar. A
tecla é anunciada **uma vez**, na primeira linha da fila.

## Impacto

- `tool.completed` ganha `diff` (aditivo).
- Fila e menu tomam linhas do fluxo; sem isso as últimas linhas de saída são
  desenhadas por baixo da entrada.
- Novas teclas: `^X`. `Esc` ganha um nível antes dos existentes.

## Alternativa descartada

Mandar o diff no `Output` da ferramenta, que já vai ao evento. Descartada porque
`Output` é o que o modelo lê: um diff de 400 linhas em todo `edit` seria pago em
todo turno seguinte da sessão, para um leitor que já sabe o que escreveu.
