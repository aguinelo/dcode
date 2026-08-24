# O painel de sessão é o que rodou

**Data:** 2026-08-24
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** observação de quem usa, olhando o painel numa sessão real

## O que mudou

O medidor de contexto saiu do painel de sessão. O espaço foi para a lista do que
rodou.

## Pela segunda vez hoje, uma coluna repetindo a tela

De manhã a lista de arquivos foi escondida por medida: ela era uma segunda cópia
do que o fluxo tinha acabado de dizer.

À tarde, o painel que a substituiu carregava `contexto 112.0k / 1.0M` e um
medidor embaixo — **três linhas reafirmando um número que a barra de status já
mostra como `ctx 6%`**.

É a mesma objeção, na coluna que substituiu a primeira, escrita pela mesma mão.
Está dito assim porque é o padrão que importa: não basta ter a regra, é preciso
aplicá-la ao que se acabou de escrever.

O que ficou é a única linha ali que fala do turno **da pessoa** e não do modelo:
quanto do que foi pedido ela permitiu.

## Comando perdia o começo

`recentRow` encurtava todo alvo pelo *basename*. Para um caminho está certo;
para um comando de shell corta na última barra. Um `curl` para
`…/web/api/v4/trips/lowest-price?from=maringa-pr` aparecia no painel como

```
17:20 bash lowest-price?from=maringa-pr&to=sao-paulo-barra-…
```

que se lê como um arquivo que ninguém tem.

Quem decide é `looksLikePath`, a única resposta deste pacote para "isto é um
caminho" — a mesma decisão que a elisão da linha de chamada toma, e pelo mesmo
motivo: uma segunda lista de nomes de ferramenta divergiria da primeira.

## O relógio só quando muda

Uma rajada de chamadas cai toda no mesmo minuto, então onze linhas de `17:20`
são onze repetições de um fato, tirando seis colunas do comando em cada uma.
Onde o minuto é igual ao da linha de cima, o espaço vai para o que rodou.

```
17:20 edit DCODE.md
      edit DCODE.md
      bash uv run python -m unittest discover -s tests -v 2…
      read test_client.py
      bash uv lock --quiet && uv sync --quiet 2>&1 | tail -…
```
