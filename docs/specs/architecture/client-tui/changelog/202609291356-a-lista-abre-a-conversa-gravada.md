# A lista abre a conversa gravada

**Data:** 2026-09-29
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 10)
**Fonte:** conferência do handoff de desenho do desktop contra o código —
`refs/design/desktop/CONFERIDO.md`, divergência 12, na branch
`docs/desktop-design-brief` (aguinelo/dcode#400).

## O que quebrava

A lista do `^R` é o que este workspace **gravou**, lida do disco quando a
interface abre (`recordedSessions`, em `cmd/dcode/tui.go`). O `enter` sobre uma
linha chamava `GetSession`, que só responde pelas sessões que o daemon tem
**agora** (`Manager.Get`, atrás do `getSession` do servidor).

Conversa de uma execução anterior está no disco e não no daemon. Com o daemon
embutido, que nasce vazio a cada execução, isso é **a lista inteira**: escolher
qualquer linha devolvia `could not resume …: no session …`. A lista falhava
justamente naquilo para que existe.

## Por que nada pegou

O transporte falso dos testes respondia `GetSession` para qualquer id. Agora
responde só pelas sessões que `ListSessions` devolve, como o servidor de
verdade, e o teste que reproduz o defeito entrou vermelho, num commit anterior
ao da correção.

A frase que levou ao desvio está em `202608210300-a-trilha-de-sessoes`: "o
`/resume` já faz o continuar". Não fazia — o `/resume` **reanexa** a uma sessão
viva. O `^R` foi ligado ao mesmo caminho e herdou dele a pergunta errada. O
`202608210500-a-coluna-toma-o-teclado` já dizia o certo: `enter` **continua** a
conversa sob o cursor.

## O que mudou

Escolher uma conversa na lista pergunta primeiro ao daemon se ela está viva.

- **Viva, ela é anexada.** Um daemon compartilhado por dois terminais
  (`dcode serve`) pode ter aberta, no outro, uma conversa que também está
  gravada. Continuá-la poria a mesma conversa em duas sessões que seguem
  separadas, e a desta tela não seria a que está sendo trabalhada lá.
- **Só gravada, ela é continuada**, com `CreateSession{Resume}` — o que o
  `dcode -r` faz com ela: sessão nova, com id próprio, levando a conversa. A
  marca `session.resumed` no topo do log dessa sessão é o que diz na tela que é
  continuação, e de onde veio. O cliente não escreve essa linha por conta
  própria: linha escrita pelo cliente some quando alguém reanexa, e a marca não.
- **Qualquer outra falha é dita**, e nada é aberto por palpite. Só o daemon
  dizer que não tem a sessão faz da escolha uma continuação; continuar uma
  sessão que estava viva, afinal, a partiria em duas.

## As regras do `dcode -r`

A lista é o `dcode -r` promovido a sobreposição, e continuar a partir dela
segue as regras de lá.

O **modelo** pedido é o que a borda resolve para o `-r`: vazio quando nada nesta
execução nomeou um modelo, para o daemon reconectar ao pacote com que a conversa
foi construída (`202609161600`), e o explícito quando algo nomeou. Resolvido uma
vez e injetado, como a língua e a própria lista — duas maneiras de continuar
uma conversa acabariam discordando sobre em que modelo ela volta. O `/model`
digitado nesta sessão não entra nessa conta: ele troca o modelo da sessão em que
foi digitado, e não diz nada sobre conversas que ainda serão continuadas.

A **fronteira** é a da sessão em que a pessoa está, como no `/clear` e no
`/model`: abrir uma conversa de dentro da interface não a devolve, no meio do
trabalho, ao modo de sandbox da configuração.

## O que não mudou

**O `/resume <id>` continua só reanexando.** A listagem dele é a das sessões
vivas, e a seção 8 diz "lista sessões e reconecta". Fazer dele também um
continuar é outra mudança, numa superfície `stable`, e não é a que foi pedida.

## O que fica de fora

A marca `●` diz qual conversa está aberta comparando ids, e uma continuação tem
um id que a lista, lida na abertura, não tem. Depois de continuar uma conversa
pela lista nenhuma linha fica marcada, e escolher a mesma de novo abre outra
continuação dela, sem os turnos feitos na primeira — que continuam gravados, e
vivos, ao alcance do `/resume`. Se a conversa de origem conta como "a aberta" é
pergunta de desenho com tema próprio, e vai em branch separada.
