## en

- **The model's text is drawn as the markdown it is.** Replies with bold
  headings and bullet lists showed literal asterisks and dashes. Model text now
  draws paragraphs, bold, italic, inline code, fenced code blocks (monospace,
  scrolling sideways), modest headings, bullet and numbered lists with nesting,
  quotes, rules, tables and links. `marked` 18.0.14 (MIT, no dependencies) only
  tokenises, and the tree is mapped to React elements: no HTML string is built,
  so HTML the model writes shows as text and the CSP is unchanged. Only `https`
  links are links, opened in the browser through the main process; any other
  address stays text with the address in its title, and images are their alt
  text. While a reply streams, an unclosed `**` stays as typed and an unclosed
  fence is code up to where the text has reached. Styles use only the existing
  tokens, so the grid, its compact windows and Crew all take them (D34).

## pt-BR

- **O texto do modelo é desenhado como o markdown que ele é.** Respostas com
  títulos em negrito e listas mostravam os asteriscos e os traços crus. O texto
  do modelo agora desenha parágrafos, negrito, itálico, código em linha, blocos
  de código cercados (mono, com rolagem horizontal), títulos modestos, listas
  com e sem número e aninhadas, citações, linhas, tabelas e links. O `marked`
  18.0.14 (MIT, sem dependências) só separa os tokens, e a árvore vira elementos
  React: nenhuma string de HTML é montada, então o HTML que o modelo escreve
  aparece como texto e a CSP não muda. Só link `https` é link, aberto no
  navegador pelo processo principal; qualquer outro endereço fica como texto, com
  o endereço no título, e imagem é o texto alternativo. Durante o streaming, um
  `**` sem par fica como chegou e uma cerca sem fim é código até onde o texto
  chegou. Os estilos usam só os tokens que existem, e valem na grade, nas janelas
  compactas e na Crew (D34).
