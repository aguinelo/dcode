# Um turno carrega documentos

**Data:** 2026-10-15
**Specs afetadas:** `202608072240-client-server-protocol` (`.p` §5, §5.1 e
§9), `202608072334-provider-adapter` (`.p` §2).

## O que mudou

| Onde | O quê |
|---|---|
| `SubmitTurnRequest` | `documents`: `[]TurnDocument` — `name`, `media_type`, `data` em base64 |
| `TurnStarted` | `documents`: `[]AttachedDocument` — `name`, `media_type`, `size`, nunca os bytes |
| `protocol` | os tipos aceitos (`DocumentPDF`, `DocumentText`, `DocumentMarkdown`, `DocumentCSV`) e os limites (`MaxDocumentsPerTurn` 5, `MaxPDFBytes` 10 MiB, `MaxTextDocumentBytes` 256 KiB, `MaxDocumentNameBytes` 255) |
| `provider.Family` | `AcceptsPDF() bool`; só `claude` diz sim |
| `pkg/client` | `SubmitTurn`, que manda o pedido inteiro |

## Por quê

Uma imagem já chegava ao modelo; um PDF ou um arquivo de texto, não — a pessoa
copiava o conteúdo para a caixa de texto, ou dizia o caminho e esperava o
agente ler, o que não serve quando o arquivo não está no workspace.

**Por valor, como a imagem.** O daemon pode estar em outra máquina, e um
caminho só significa algo onde foi digitado. Pelo mesmo motivo o nome é nome de
arquivo e nada mais: com separador, vazio, longo demais ou com caractere de
controle, é recusado. Ele aparece na mensagem e para o modelo, e um caminho
contaria aos dois algo de uma máquina em que eles não estão.

**Conferido na borda.** Tipo de um conjunto fechado; PDF pelos bytes
(`%PDF-`), nunca pelo nome; texto UTF-8 sem NUL, que é o que um binário
renomeado para `.txt` não é. Tamanho medido antes de decodificar. Toda recusa é
`invalid_input` dizendo o quê, antes de a sessão ver um byte.

**A família declara, e a submissão recusa.** Mesma regra da imagem: o que se
declara não se tenta. `claude` lê PDF como bloco `document` (o texto e uma
imagem de cada página). `minimax-m3`, `gemini` e `generic` dizem não — não é
afirmação de que não conseguem, é que ninguém aqui mediu, e a superfície
compatível com OpenAI não tem parte para arquivo. PDF para um desses recusa o
turno **na submissão**, nomeando o modelo. As alternativas eram erro do
provedor depois do envio, ou a pergunta indo sem o arquivo de que fala, e a
resposta parecendo erro do modelo. Por baixo disso, o dialeto que não carrega
PDF recusa codificar a mensagem: o caminho que chegasse à família sem passar
pela sessão também não manda a pergunta sem o arquivo.

**Texto vai para toda família, escrito na mensagem.** Decidido assim, e não
como um bloco `document` de texto no dialeto Anthropic: todo modelo lê texto,
então não há o que declarar, e uma forma só para todos é uma forma só a
conferir. Vai antes da pergunta — a ordem em que se lê um documento e depois o
que se pergunta dele — entre `<document name="…" media_type="…">` e
`</document>`, com o nome escapado para não abrir atributo próprio. Por estar
no texto, conta no orçamento com ele.

**256 KiB de texto, 10 MiB de PDF.** Texto é lido inteiro: 256 KiB são uns
sessenta mil tokens, um terço da menor janela que uma família declara; maior
que isso é arquivo para apontar ao agente, não para colar. PDF segue o limite
da imagem, abaixo do teto de requisição do provedor mesmo em base64.

**PDF pesa pelas páginas.** Estimadas pelos objetos `/Type /Page`, a 12.000
caracteres cada (o alto da faixa que a Anthropic documenta, a quatro caracteres
por token). PDF que esconde as páginas num fluxo comprimido pesa uma, não zero:
o erro que o orçamento aguenta é achar curto, não achar de graça.

**`turn.started` nomeia, não carrega.** O log é reproduzido a todo cliente que
anexa e fica em disco; a conversa já tem o conteúdo. Nome, tipo e tamanho
bastam para o histórico mostrar o que foi anexado à pergunta.

## Fora daqui

Continuar uma conversa não traz os documentos de volta ao contexto, como não
traz as imagens: o registro guarda o nome, não o arquivo. O TUI ainda não anexa
documentos; o desktop anexa, no PR dele.

`experimental`, abaixo de 1.0: campos e um método de interface novos são
**MINOR** como feature.
