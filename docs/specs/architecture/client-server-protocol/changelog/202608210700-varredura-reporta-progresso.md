# Varredura reporta quão longe foi, e o resultado pousa na chamada certa

**Data:** 2026-08-21
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`), `202608081250-client-tui` (`.p`)
**Fonte:** `refs/design/HANDOFF.md` (v5, §3)
**Continua:** `202608210600-o-evento-de-progresso.md`

## O que mudou

`kind: "files"` entra no conjunto declarado. O `grep` diz `n de N`, o `glob`
manda só a contagem, e o card mostra `150/184` onde antes havia reticência.

O relator viaja **no contexto da chamada**, não no `State`.

## Por que no contexto e não no `State`

O `State` é **por sessão** e compartilhado. Duas varreduras rodando em paralelo
escreveriam a contagem pelo mesmo campo, e a tela mostraria o progresso de uma
sob o nome da outra.

O contexto já é por chamada e já atravessa o `Execute`. Nada de mutex, nada de
campo que só está certo enquanto uma chamada roda por vez.

`Progress(ctx)` nunca devolve nil: ferramenta deve dizer quão longe foi sem antes
perguntar se alguém escuta, e desreferência nula no meio de uma varredura é um
crash no meio do trabalho de alguém.

## O que cada ferramenta pode dizer honestamente

| Ferramenta | Reporta | Por quê |
|---|---|---|
| `grep` | `n de N` | tem a lista **antes** de varrer |
| `glob` | `n` | está descobrindo enquanto anda; total que ele ainda não terminou de contar seria número inventado |
| `read` | nada | faz `os.ReadFile` e divide: aprende o total no mesmo instante em que aprende o conteúdo. Não existe momento em que "n de 240" seja verdade |
| `bash` | nada | contar teste que passou exige parsear a saída, que o comentário do `ToolCompleted` proíbe |

Por isso não há `kind` para linhas nem para testes. **Kind que só poderia ser
preenchido desonestamente é kind que não se declara.**

O `grep` também fica **calado durante a própria caminhada**: duas fases com
totais diferentes mostrariam uma contagem que sobe, reinicia e sobe de novo, o
que se lê como defeito.

## A cada 25, não a cada arquivo

Um por arquivo poria um evento por arquivo no log e no registro — varredura de
dez mil arquivos escrevendo dez mil linhas que ninguém lê. A cada vinte e cinco
mantém uma varredura de cem honesta em quatro atualizações e uma grande longe de
inundar. O primeiro é sempre enviado: é o que transforma "algo está acontecendo"
em "tantos até agora", e numa varredura pequena pode ser o único.

## Um defeito latente que isso descobriu

O `ToolCompleted` casava com a **última entrada rodando**, o que está certo
exatamente enquanto uma chamada roda por vez. Com duas em vôo, o primeiro
resultado pousava na linha da segunda — **os números eram reais e a linha era a
errada**.

`Entry.CallID` conserta os dois: o roteamento do progresso e o do resultado. Foi
achado porque o progresso precisava do mesmo endereçamento, não porque alguém
notou a tela errada — que é o jeito barato de achar.
