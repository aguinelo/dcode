# Marca

🇬🇧 [English version](README.md)

<img src="mascot.svg" width="96" alt="mascote do dcode"> &nbsp;&nbsp; <img src="logomark.svg" width="96" alt="logotipo do dcode">

Duas marcas com papéis diferentes. O **mascote** é o personagem — adesivo, objeto físico,
onde personalidade ajuda. O **logotipo** é o identificador — favicon, cabeçalho, onde o
nome precisa ser reconhecido de relance.

Mesma gramática de pixel, mesma paleta, para lerem como um sistema só.

## Arquivos

| Arquivo | Uso |
|---|---|
| `mascot.svg` | o personagem de três caixas |
| `logomark.svg` | o D — haste contínua, **primário** |
| `logomark-segmented.svg` | o D com a haste quebrada nos vãos |
| `favicon.svg` | o logotipo em escala de favicon |
| `VOXELS.md` | as grades que geram tanto os SVGs quanto o modelo 3D |

## Paleta

| Token | Hex | Papel |
|---|---|---|
| realce | `#EFC066` | face superior de cada caixa |
| corpo | `#E0A030` | face frontal — a cor primária |
| sombra | `#B87D1E` | borda inferior e direita |
| olho | `#A8452A` | o marcador, e nada mais |

Âmbar e terracota são matizes análogos, então o contraste não vem do matiz — vem da
**luminância**. `#E0A030` é claro e `#A8452A` é escuro. É por isso que o olho sobrevive a
16 px, onde só diferença de matiz já teria virado uma mancha só.

Três tons de âmbar dão volume sem contorno: face iluminada, face frontal, face sombreada.
O mesmo raciocínio de um render isométrico.

## O olho é o marcador de ferramenta

O `⏺` aparece em toda linha de execução da TUI — `⏺ read`, `⏺ edit`, `⏺ bash`. Usá-lo
como olho faz cada linha na tela ser uma repetição da marca.

É o oposto de um logotipo aplicado por cima: emerge do produto. E é o único traço do
mascote, o que basta de personalidade para não cair no vale da estranheza nem impor
caráter a uma ferramenta.

## Por que existe a variante segmentada

O esboço aprovado quebrava a haste nas mesmas linhas do bojo, o que lê como três peças
soltas em vez de letra quando reduz. O `logomark.svg` mantém a haste contínua e quebra só
o bojo — continua mostrando três caixas, e continua lendo como D a 16 px.

O `logomark-segmented.svg` é o original. Mantido porque a versão segmentada é mais forte
em tamanho grande, onde a letra nunca está em dúvida e a construção é justamente o ponto.

## Objeto físico

Desenhado como **três peças que encaixam**, não uma impressão só. O nome vira o objeto:
você monta empilhando as caixas.

| Propriedade | Valor |
|---|---|
| Pirâmide | 12 → 10 → 6 voxels de largura |
| Altura montado | 64 mm com voxel de 4 mm · 96 mm com 6 mm |
| Suporte | nenhum — cada peça imprime na própria face |
| Ressalto | nenhum acima de 45° |
| Encaixe | pino de 3 mm em furo de 3,2 mm, por fricção |
| Olho | furo passante de 2 voxels, ou pino terracota de 2 mm |

A base larga põe o centro de massa no terço inferior, então fica de pé sem lastro e
imprime sem raft.

## Regras

- **Nunca recolorir o olho.** É o único elemento fixo; todo o resto pode se adaptar.
- **Nunca adicionar traços de rosto.** Um marcador é o rosto inteiro.
- **Nunca contornar em preto.** O volume vem dos três tons de âmbar.
- **Abaixo de 16 px, use o logotipo**, não o mascote — as três caixas param de se separar.
- O mascote precisa renderizar dentro do produto, no terminal. Marca que não pode aparecer
  na própria ferramenta é decoração externa.
