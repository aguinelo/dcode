# O fluxo tem raias

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** design `Coding Agent TUI v2`, projeto Claude Design
`fbbd32a7-28b3-4646-9497-aa948789ccb2`

## O que mudou

Toda linha do fluxo está numa raia: **o que você pediu**, **o que o modelo fez
no caminho**, **o que ele diz**. Uma coluna à esquerda diz qual.

```
▏ Show, antes de sair escrevendo código quero entender o escopo — Crawlee tem
▏ prós e contras aqui que vão definir o resto.
╎ ⏺ bash   pwd && ls -la              exit 0
╎ ⏺ read   DCODE.md                   24 lines
▏ Confirmado, o DCODE.md continua igual ao que escrevi.
```

## O que ela resolve

Num turno longo, prosa e chamadas de ferramenta se alternavam sem nada
estrutural entre elas. Recuperar o fio significava **ler toda linha para
descobrir quais valiam a leitura**.

Com a raia na calha, o olho corre pelo `▏` e pula o `╎`.

## Custa zero colunas

Toda linha do fluxo já reservava duas colunas — o marcador de seleção, ou dois
espaços onde não havia um. A raia toma a primeira e o marcador fica na segunda.

É a mesma propriedade da régua de turno: a mudança que mais melhora a leitura
por coluna gasta é a que não gasta nenhuma.

## A raia `você` não tem glifo

Ela já é marcada duas vezes — pela régua e pelo `❯`. Uma terceira marca no único
bloco que ninguém erra é tinta gasta onde não há confusão.

## Caractere, nunca só cor

Raia distinguida por cor não é raia num terminal sem cor. É a mesma regra que o
cursor da lista, o modal e o corte já carregam, e o teste a afirma nos dois
modos.

## O que não veio do desenho, e por quê

O v2 traz um bloco RESULT no fim do turno, um painel de diff com barras
proporcionais, uma lista WATCH de processos em segundo plano, um painel de
sessão, quatro temas trocados por `t`, e navegação `j`/`k` com contador.

Nenhum deles entrou, e por três razões distintas — registradas em
`docs/ROADMAP.md` §11 nomeando o desenho como fonte:

- **Falta um fato no protocolo.** O bloco RESULT precisa saber qual bloco de
  prosa **conclui** o turno, e `KindAssistant` cobre narração e conclusão sem
  distinguir. Mesma forma da lacuna de `tool.progress`.
- **Contradiz uma medida desta manhã.** Um terceiro painel residente é
  exatamente o que a medição das colunas rejeitou. As barras de diff são boas e
  **não** são repetição do fluxo — elas pertencem à coluna, não a um painel novo.
- **O atalho é letra.** `t` para tema e `j`/`k` para navegar são o defeito
  corrigido duas vezes hoje. Precisam de um modo que tome o teclado, e este
  produto já tem um.
