# O Chromium alcança o primeiro quadro

**Data:** 2026-08-24
**Specs afetadas:** `202608072336-sandbox-policy` (`.p`, seção 6)
**Fonte:** sessão real de quem usa — um agente desistiu de um navegador três vezes seguidas

## O que mudou

O perfil Seatbelt passou a conceder duas coisas:

```lisp
(allow mach-register)
(allow iokit-open (iokit-user-client-class "RootDomainUserClient"))
```

Sem elas, **qualquer Chromium morre com SIGSEGV antes de desenhar nada**.

## Como isto se escondia

Não era uma negação. Era um **sinal**.

Uma negação de sandbox produz `Operation not permitted` — uma frase que aparece
na saída da ferramenta e que alguém consegue ler. Um SIGSEGV produz um stack
trace e um código de saída, e nada em lugar nenhum diz que houve fronteira
envolvida.

Então o que a tela mostrava era um navegador quebrando. Quem estava olhando leu
como **timidez do modelo** — "ele não está tendo autonomia de executar as
coisas". O modelo tinha feito 167 chamadas naquela sessão e contornado o sandbox
sozinho duas vezes; o que ele não conseguia era abrir um navegador.

Atingia tudo que é Chromium: Playwright, Puppeteer, Lighthouse, um Electron sob
teste.

## É um par

Nenhuma das duas sozinha passa do crash. As duas juntas passam, 3 de 3
execuções.

Achado bisectando um perfil Seatbelt contra o binário real, e não lendo o
código do Chromium. A primeira hipótese — `mach-register` sozinho, tirada do
`bootstrap_check_in: Permission denied` que aparece no log dele — **estava
errada**, e só o teste mostrou isso.

## O que cada uma custa

`iokit-open` está **escopado a uma classe**, e é essa a razão de ser acessível.
Aberto, ele é a superfície de GPU e HID — historicamente fonte de exploit de
kernel. `RootDomainUserClient` é o root domain de energia, que o Chromium abre
para tomar uma assertion contra suspensão. O escopo foi encontrado apertando a
concessão ampla até a menor classe que ainda funciona.

`mach-register` é o alargamento real: o processo passa a poder registrar um
serviço Mach nomeado no bootstrap namespace da própria sessão. Esta fronteira é
sobre arquivos e rede, então não é onde ela mora — mas é superfície, e está
dito aqui em vez de enterrado no perfil.

As duas ficam no **preâmbulo**, junto de `process-exec` e `mach-lookup`, que são
a mesma classe de coisa: o que um programa precisa para **começar**, e não o que
ele pode ler ou escrever depois.

## O que continua recusado, e certo

Um Chrome **completo** escreve o banco do Crashpad em
`~/Library/Application Support/Google/Chrome/` independentemente do
`--user-data-dir`, e o sandbox recusa. Isso é a fronteira funcionando: é escrita
fora do workspace. Quem quiser o navegador completo concede aquele caminho pelo
nome, que é para isso que serve concessão nomeada.

O **headless shell** — que é o que o Playwright e todo navegador dirigido por
agente de fato lançam — roda com as duas regras e mais nada.

## O teste pergunta ao kernel

O perfil pode ser lido e aprovado enquanto a coisa que ele governa morre. Não
havia negação para afirmar e não havia erro na saída da ferramenta; a única
forma de guardar isto é lançar um navegador de verdade dentro da fronteira e
olhar se ele chega a um quadro.

Confirmado que ele falha com a concessão removida.
