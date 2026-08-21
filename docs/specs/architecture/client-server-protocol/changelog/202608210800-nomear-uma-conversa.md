# Nomear uma conversa, guardado no registro dela

**Data:** 2026-08-21
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`), `202608081250-client-tui` (`.p`)
**Fonte:** `refs/design/HANDOFF.md` (v5, §2 — modo *nomeando*)

## Onde o nome mora, e por que ali

Evento `session.renamed`, no registro da própria conversa. Três lugares foram
considerados:

| Onde | Por que não |
|---|---|
| arquivo ao lado da sessão | a poda apaga a transcrição e não sabe do vizinho: sobram órfãos |
| índice por workspace | sobrevive à poda, e é justamente o problema — guarda nome de conversa que ninguém consegue mais abrir |
| **o próprio registro** | o nome morre com o que ele nomeia |

O critério que decidiu: **nome de conversa que não existe mais é pior que nome
nenhum**. Uma lista cheia de títulos para sessões que não abrem é pior que uma
lista curta.

E mantém a conta em um. Depósito ao lado do log é uma segunda coisa que pode
discordar dele, e a forma inteira deste protocolo é que todo fato observável
viaja do mesmo jeito.

Não custou leitura: o `Browse` já varre cada linha de cada registro para contar
turnos, então o evento é lido no mesmo passe que já acontecia.

## A sequência é lida, nunca presumida

Acrescentar um número que já está no arquivo poria duplicata num log cujo
contrato inteiro é que não há nenhuma. O `Rename` lê o maior `seq` do arquivo e
acrescenta o seguinte, sob um mutex — registros são append-only e o daemon é um
processo só, então dois renames da mesma conversa chegam na ordem em que
chegaram, e o último vence por ser o último.

## Decisões pequenas, com o motivo

**Nome vazio devolve o título derivado**, e não é erro. Uma operação com um valor
zero que significa alguma coisa é uma coisa a acertar, em vez de duas.

**Caractere de controle não chega ao registro.** O arquivo é lido de volta linha
por linha, e uma quebra dentro de um nome faria uma linha parecer duas.

**Nome longo demais é recusado, não aparado.** Guardar metade do que foi digitado
em silêncio é como alguém acaba com um nome que não escolheu. O cliente para no
mesmo limite, para não desperdiçar a digitação e depois relatar falha.

**Escreve no registro, não na sessão viva.** A trilha lista o que o workspace
gravou, e quase nada disso está carregado — um rename que só funcionasse na
sessão aberta funcionaria na única linha que não precisa dele.

## No cliente

`r` e `F2` abrem o modo com a trilha focada. Enquanto ele está aberto **toda
tecla é do nome** — é a única coisa ali que muda algo, e nada mais pode ser
alcançado sem querer no meio.

O rascunho parte do **nome**, nunca do título derivado: oferecer o título
transformaria "dê um nome a isto" em "confirme o que te deram", e o primeiro
Enter promoveria um título derivado a nome escolhido sem ninguém decidir.

`esc` cancela e mantém o que havia, que é a regra do próprio design.

Nome dado leva `·` na lista. Sem a marca, a coluna mostra dois tipos de afirmação
— derivado e escolhido — e nada os distingue.
