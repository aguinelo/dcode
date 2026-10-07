# O aviso de abertura chega a quem anexa

**Data:** 2026-10-07
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`: o tipo `Notice`
em §5, o evento `session.notice` em §5.1, três linhas novas em §9);
`202608072334-provider-adapter` (`.p`: a linha da família sem medição vira duas
em §7)
**Regras que não estavam sendo cumpridas:** RN-11 de `provider-adapter` e
RN-6.2 de `configuration`.

## O problema

A RN-11 de `provider-adapter` diz que família sem medição avisa isso na sessão.
A RN-6.2 de `configuration` diz que instrução escrita para outra ferramenta e
não traduzida é avisada, com o número medido, quando a sessão começa. Nenhum dos
dois avisos chegava a ninguém, por dois defeitos empilhados:

- **Um aviso tomava o lugar do outro.** O `New` guardava a admissão da família
  (`provider.Unmeasured`) numa string só e, com `instruction.notice` ligado — o
  padrão —, atribuía por cima o aviso das instruções, que é `""` num workspace
  sem nada a traduzir. Na configuração padrão, uma sessão em `claude` ou
  `generic` não tinha nada a dizer. Vem de 11 de agosto (#45), quando o aviso da
  `generic` entrou antes do das instruções em vez de ao lado dele.
- **Ninguém lia o que sobrava.** `Session.Notice` não teve leitor desde que
  existe (#27, 10 de agosto): nem o daemon, que monta a sessão do protocolo sem
  ele, nem a TUI, nem o desktop, nem o caminho de uma tarefa só
  (`dcode "tarefa"`). O aviso das instruções também nunca chegou.

A guarda da RN-11 ficou verde o tempo todo. A linha de §7 prometia duas coisas —
que a família avisa na sessão, e que a lista de quem avisa é conferida contra as
medições — e era reivindicada pelo teste da lista, que estava certa. A metade
que falhava não tinha teste. O changelog de `202610070001` já registrava o
defeito, para branch própria; até aqui, o menu de modelos era o único lugar em
que o aviso chegava a alguém.

## O que mudou

- **`session.notice`**, evento novo, um por aviso, com `Notice{code, message}`.
  Os códigos são `family_unmeasured` — a admissão da própria família — e
  `instructions_untranslated` — o aviso das instruções, inclusive o de um
  `DCODE.md` cujas origens mudaram desde a tradução.
- **O `New` guarda os dois**, em `Session.Notices`, a família primeiro.
- **O daemon os entrega à sessão**, que os diz depois do `session.created` e da
  conversa continuada, antes da resposta de criação.
- **O `Carry` deixa para trás** os avisos da sessão continuada.
- **Quem lê:** a TUI mostra cada aviso como nota; o desktop, como nota em tom de
  aviso; o caminho de uma tarefa só imprime cada um, com `⚠`, sob a linha que
  nomeia o modelo.

## Como, e por quê assim

- **Um evento, e não um campo no `session.created`.** Aviso é dito uma vez e
  lido na transcrição. Como campo de `Session`, viraria parte de como a sessão
  se descreve — em `GET /sessions` e na resposta de criação, a cada listagem —,
  e cada cliente teria de transformar a descrição de volta em linha. As duas
  formas foram postas para a pessoa, que escolheu o evento.
- **E não `session.error`.** Nada falhou, e um cliente que mostra erro como
  falha estaria certo em mostrar.
- **Um evento por aviso.** Assim um não toma o lugar do outro por construção, e
  cada um tem o seu código.
- **A forma do `Error`.** `code` é com o que o cliente age — marcar a sessão
  como sem medição, apontar o `/init` —, e `message` é a frase do daemon, em
  inglês. Outra língua, se vier, pendura-se no código, como no `Error`.
- **Depois da conversa continuada, não antes.** Quem continua está olhando o fim
  do que foi carregado, e um aviso acima de dezoito mil eventos é um aviso que
  ninguém lê. Antes da resposta, para `last_seq` contar os avisos e quem lê até
  ele os ler.
- **No registro.** Os avisos são da sessão que abre, e o registro é onde alguém
  a lê depois. A conversa carregada continua só no log.
- **Continuar não carrega os avisos da sessão continuada**, pelo motivo por que
  o `Carry` já pula o `session.created` dela: falam de uma sessão que não está
  mais em vigor. Trocar de modelo é continuar numa sessão nova (D28 do desktop);
  carregado, o aviso de família sem medição continuaria dizendo isso de uma
  conversa que já está num modelo medido.
- **Guardados e ditos depois, como a conversa carregada.** O `New` roda antes de
  a sessão do protocolo existir, e o log abre com a criação dela.

## As invariantes

Em `provider-adapter`, a linha da RN-11 vira duas, uma promessa cada — a regra
de `202609281513`:

| Linha | Teste |
|---|---|
| Família **sem nenhuma medição** registrada avisa isso na sessão: quem anexa recebe a admissão dela num `session.notice`. | `TestAnUnmeasuredFamilySaysSoToWhoeverAttaches` |
| A lista de quem avisa é conferida contra as medições que existem, nos dois sentidos. | `TestEveryUnmeasuredFamilySaysSo` |

A guarda da família passa a ler também `internal/app`, onde a sessão é montada e
anexada, com o motivo escrito onde ela é alargada.

Em `client-server-protocol`, três linhas novas:

| Linha | Teste |
|---|---|
| Os avisos de abertura saem todos, um `session.notice` cada: o das instruções de outra ferramenta não toma o lugar do da família sem medição. | `TestNoOpeningNoticeTakesAnothersPlace` |
| Aviso de abertura entra no log depois do `session.created` e da conversa continuada — no fim do que foi carregado, onde quem continua está olhando — e antes da resposta de criação, cujo `last_seq` já o conta. | `TestOpeningNoticesComeAfterTheContinuedConversation` |
| Continuar não carrega os avisos de abertura da sessão continuada: eles falam de uma sessão que não está mais em vigor. | `TestACarriedConversationLeavesItsOpeningNoticesBehind` |

O teste que reproduz o defeito abre a sessão pelo daemon de verdade, num socket,
com a configuração padrão, e lê o log como um cliente anexado: `claude` pelo
prefixo do modelo, `generic` por um perfil do projeto. Vermelho antes da
correção — nenhum aviso — e commitado assim.

## Consequência visível

Num repositório com `AGENTS.md` ou `CLAUDE.md` e sem `DCODE.md`, toda sessão
nova abre dizendo isso, com o número medido: é a RN-6.2 cumprida. Na TUI, a
linha toma o lugar da tela inicial, como o aviso de versão já fazia.
`instruction.notice = false` desliga esse aviso, e só ele: o da família continua.

## SemVer

Evento, tipo e códigos novos; nenhum contrato existente muda de sentido. O
protocolo é `experimental`, e o aditivo nele é MINOR pelo critério de
`202610070001` — a próxima versão do núcleo já é MINOR por ela. O commit é
`fix:` porque o que ele faz é cumprir duas regras que já estavam escritas.

## O que fica de fora

- **O `memory_unreadable`.** O `New` emite um `session.error` quando a memória
  do workspace não pode ser lida, e pelo daemon ele cai no mesmo intervalo —
  antes de a sessão do protocolo existir, quando o emissor descarta o que chega.
  Mesmo formato de defeito, outra regra e sem teste nenhum; branch própria.
- **Uma frase do desktop por código.** O desktop mostra a frase do daemon, em
  inglês, como já mostra a do `session.error`. Dizer "sem medição" com as
  palavras do design é decisão de interface, do desktop.
- **O `--dump-prompt`.** Ele audita o que vai ao modelo, e os avisos não vão.
