# Decisões do desktop — D21 em diante

A continuação de [`DECISIONS.md`](DECISIONS.md), que tem a introdução, D1 a D20, as
decisões em aberto, as lacunas e os pedidos ao núcleo. A numeração é uma só; D39
está em [`DECISIONS-3.md`](DECISIONS-3.md).

**D21. A lista de sessões vem do protocolo** (N3): vivas e gravadas, com resumo, e
um fluxo só de mudanças da lista. Abrir uma gravada continua a conversa numa
sessão nova (`CreateSession{Resume}`), como a TUI faz. Com o N3 feito, a lateral
lê `GET /v1/conversations` e acompanha `/v1/conversations/events`. Descartados: ler o disco como a TUI (acopla o desktop
ao formato do registro e duplica a leitura); só as vivas (a lateral perde o
histórico).

**D22. `↵` nega a aprovação, como na TUI** (era A1). Permitir é sempre escolha
explícita: `1` uma vez, `2` nesta sessão; `3`, `esc` e `↵` negam. A opção
destacada é negar, e só isso muda no design. O campo de mensagem fica desabilitado
enquanto a aprovação espera, com o texto "Responda à aprovação acima". Medido
antes de decidir: a tela 03 fica a 1,21% das referências, dentro do limite.
O `↵` da janela só responde com o foco em lugar nenhum: num botão de resposta ou
numa linha da lateral, o `↵` é daquele controle — senão `↵` sobre "Permitir uma
vez" negaria. O campo, ao ser desabilitado, devolve o foco à janela, e com ele as
teclas da resposta. Descartados: `↵` permite uma vez, como no design (aprovar sem ler vira o gesto mais
barato, e a invariante da TUI teria de mudar junto); `↵` sem efeito (uma regra
diferente em cada cliente).

**D23. Mensagem durante o turno redireciona, como na TUI** (era A2). O texto vai ao
modelo na próxima rodada do mesmo turno (`POST …/steer`) e aparece no fluxo
quando o daemon o entrega (`turn.steered`); imagem e comando que custa turno,
quando existirem aqui, entram na fila. O campo diz "Escreva para redirecionar
este turno" — medido: a tela 02 fica a 1,20%. Descartado: a fila do design — com
ela, corrigir um turno que foi para o lado errado exigiria interrompê-lo, e a
mesma sessão teria uma regra em cada cliente.

**D24. O loop é uma tela sobre o mecanismo que existe** (era A4): os critérios do
`done.toml`, a reentrada por ciclo e um evento por ciclo com o resultado de cada
critério — passou ou não, sem contagem de testes, porque critério é código de
saída (`agent-loop`). A sequência do `/loop` sai da TUI para o daemon, para os dois
clientes verem o mesmo loop (N4). Vem depois da conexão; até lá, a sessão em loop
aparece como rodando. Descartados: o loop novo de `refs/design/desktop/LOOP.md`
(iteração como turno inteiro, orçamentos, contagem por teste, `dcode loop` —
várias specs, e contraria o `agent-loop`); e decidir depois.

**D25. Tokens durante o turno: só o tempo, por enquanto.** A linha de atividade
mostra o tempo desde o `turn.started`; tokens e contexto mudam quando o turno
termina, como na TUI desde o #401. Descartado por ora: um evento de uso por
rodada, que seria mudança de protocolo (MINOR). A lacuna L5 fica, por decisão.

**D26. Os dados do app ficam numa pasta só dele.** O Chromium guarda na pasta de
dados do app o `localStorage` das preferências (D7), caches e cookies. O padrão do
Electron é o nome do produto dentro da pasta de dados do sistema — no macOS,
`~/Library/Application Support/DCode` —, e o sistema de arquivos ali não distingue
`DCode` de `dcode`, a pasta em que o próprio dcode guarda configuração, registros e
estado, o `models.toml` incluso. O app usa `dcode-desktop` nessa pasta, e
`DCODE_DESKTOP_USER_DATA` escolhe outra, absoluta — a régua dá uma a cada cenário.
Apagar os dados do app nunca leva a configuração do dcode junto. Descartado: trocar
o nome do produto, que é o nome da janela e do menu — decisão de design, não de
onde os arquivos ficam.

**D27. O ⌘K procura na lista de conversas, pelo título e pelo projeto.** Cada
palavra digitada precisa estar no título ou no projeto (o rótulo ou o caminho),
sem distinguir maiúsculas nem acentos. Os grupos são os do design: "Ativas"
primeiro — rodando ou esperando você —, depois um por projeto, na ordem da
lateral; uma conversa ativa aparece só em "Ativas". `↵` abre a do cursor pelas
regras de um clique na linha — uma terminada é continuada numa sessão nova (D21).
O vazio diz que nada foi achado e não promete o que não existe: o "↵ cria uma nova
com esse pedido" do design fica de fora até haver onde criá-la. O escopo do
projeto é o da sessão aberta, e some sem sessão aberta.

**D28. Trocar de modelo é continuar a conversa numa sessão nova.** O prefixo de
uma sessão não se reescreve (ADR-03), então nada troca de modelo no lugar: o
botão do modelo, no composer, abre um menu, e escolher continua a conversa numa
sessão nova no perfil ou modelo escolhido (`CreateSession{Resume, Model}`), com
todo o histórico, e a janela abre a nova. A sessão deixada é encerrada depois que
a nova abre — aberta, cada troca deixaria mais uma, até o daemon recusar abrir
outras (`max_sessions_reached`) —; o registro dela fica, e a lista a mostra
terminada. Só troca quem está parado: com um turno rodando ou esperando
resposta, o menu diz para parar ou esperar.

- **O menu lista o que o daemon diz que existe** (`GET /v1/models`, N5): o
  padrão do projeto e cada perfil do `models.toml`, resolvidos como a sessão os
  resolveria. Cada um diz "medido" — os contratos de comportamento do dcode já
  rodaram contra a família — ou "sem medição", com o aviso do daemon no título;
  o atual é marcado pelo modelo e pelo endpoint.
- **Daemon sem a rota, ou que não responde, é dito no próprio menu,** e nada é
  adivinhado: montar a lista com os modelos vistos nas conversas ofereceria
  nomes que não são perfil, e pedir um modelo local pelo nome perde o endpoint de
  que ele precisa.
- **A TUI faz diferente:** `/model` abre uma sessão nova sem o histórico. São
  gestos diferentes — recomeçar noutro modelo, continuar noutro modelo —, e o
  menu diz qual é o dele.

Descartados: manter a sessão anterior aberta (acumula sessões até a recusa);
ler o `models.toml` no desktop (uma segunda cópia da regra de quem vence entre o
do usuário e o do projeto, e da lista de famílias medidas); trocar só para o
próximo turno (o prefixo não muda no meio da sessão).

**D29. Grafite: escuro, um acento, a fonte do sistema.** A janela deixa o visual
do handoff v2 — claro e escuro, Geist, cantos de 10 a 14px, uma conversa por vez —,
que lia como um clone de app de chat, por um próprio: sempre escuro; um acento
lima (`#c6f36b`) para o foco e o que é seu; âmbar só para o que espera você; verde
e vermelho só para o que os critérios disseram. A fonte do sistema (SF no macOS)
no texto, mono no que é código ou nome de máquina — modelo, branch, comando,
caminho —, com a Geist Mono como reserva; cantos de 4px. Escuro só, por ora: um
segundo tema é um bloco de tokens (D6), não uma caça. (O "escuro só" mudou em
D35: há um claro, à escolha, e o escuro continua o padrão.) Descartados: manter o v2 e
só adicionar a grade (a unidade da tela continuaria a conversa); um tema claro
junto (o dobro de ajuste visual para uma proposta ainda não aprovada).

**D30. A janela abre numa grade de atenção.** Em vez de uma conversa por vez, a
janela abre na grade das conversas que pedem você, cada uma numa janela com o selo
do último turno e os critérios de pronto acima da própria conversa (D31). A
unidade da tela passa a ser a prova de cada conversa, com a conversa logo abaixo
dela.

- **Quem ganha lugar**, nesta ordem: esperando você, presa por você (D31),
  rodando, e terminada desde que você olhou — "vista" é abrir a conversa, clicar
  ou escrever na janela dela, ou fechá-la. No mesmo grupo, a mais recente
  primeiro.
- **Os lugares não pulam.** Quem está na grade fica no lugar enquanto pede
  atenção; quem chega ocupa o fim; quem sai libera. O que não cabe é contado na
  barra ("+n sem lugar na grade"), e entra pela prioridade quando abre vaga.
- **Quantos cabem** é automático: até 6 janelas, ou 9 numa janela de pelo menos
  1700×1000; o formato vai de 1×1 a 3×3. As que perguntam onde abrir uma sessão
  contam no mesmo espaço.
- **O que se faz numa janela, e o teclado:** D31.
- **"Tocou a régua"** aparece ao lado dos critérios quando o último turno
  escreveu num caminho protegido, com os caminhos no título (D13).
- **"Sem medição"** aparece ao lado dos critérios quando a família do modelo
  nunca foi medida pelos contratos de comportamento do dcode, como a lista do
  daemon diz (N5): o selo diz o que os critérios conferiram, e quem mediu o modelo
  é outra pergunta. Lista que não veio não marca nada — não saber nunca vira
  "medido".
- **A memória da grade** — fixadas, vistas e desde quando — fica no
  `localStorage`, como as preferências (D7); ilegível, é dita num aviso e a grade
  recomeça.

Descartados: abas (de novo uma por vez); um quadro por estado (move os cartões sob
o ponteiro a cada mudança); lista com prévia (o design anterior).

**D31. Cada janela da grade é uma conversa em que se trabalha.** Pedido depois de
ver a grade: cada quadrante uma janela isolada, como uma conversa própria. A
janela traz a conversa inteira — o selo e os critérios, o fluxo com rolagem
própria que segue o fim, a aprovação e o seu próprio campo —, e se escreve em
qualquer uma sem abri-la. Cada uma guarda o seu rascunho e a sua rolagem.

- **Uma janela fica enquanto é sua.** Abrir uma conversa — pela lateral ou pelo
  ⌘K —, clicar nela ou escrever nela a prende na grade; × a fecha, e o que ela
  tinha feito conta como visto. O que roda ou espera você não mostra ×: o que
  trabalha mantém o lugar. O que roda, espera ou termina continua entrando
  sozinho nos lugares livres (D30).
- **Nova sessão pergunta onde, numa janela.** Os projetos que a lista já conhece
  e "Outra pasta…"; escolher abre a sessão ali, e a janela vira essa conversa,
  com o cursor no campo. Sem projeto conhecido, o seletor de pastas vem direto,
  como antes. Fechar a janela de uma sessão que nunca recebeu mensagem fecha a
  sessão, para nada vazio ficar aberto atrás dela.
- **Uma conversa terminada continua de dentro da janela:** escrever nela a
  continua numa sessão nova (D21), que toma a janela; a troca de modelo faz o
  mesmo (D28).
- **Maximizar** é o ⤢, o duplo clique no topo da janela ou ⌘↵: a conversa ocupa o
  palco, com as outras janelas como pares no topo; Esc e ⌘↵ voltam.
- **Teclado.** ⌘N abre uma sessão; ⌘1–9 põe o cursor na janela n; Esc sai do
  campo. Sem foco num campo, as setas andam entre as janelas, ↵ põe o cursor na
  janela em foco, e uma janela esperando resposta toma 1, 2, 3, esc e ↵ — que
  nega (D22).

Descartados: abrir sempre maximizada (de novo uma por vez); a janela sair sozinha
quando a conversa para (o lugar sumiria debaixo de quem escreve); um campo só,
para a janela em foco (escrever exigiria escolher antes).

**D32. Uma segunda versão da tela, Crew, ao lado da grade.** Pedida depois de
ver o Kiro Crew, sem desmanchar a grade: um seletor na barra do topo alterna
"Grade" e "Crew", lembrado por janela no `localStorage` (como D7), e a grade
continua sendo o padrão — as duas réguas medem a grade. Na Crew, três painéis: a
lista de conversas à esquerda, a conversa em foco no meio, inteira (D31), e à
direita o que ela estabeleceu — o selo e os critérios, os arquivos que mudou
(lidos das chamadas de escrita e das linhas que as ferramentas reportaram, D8),
onde roda e com que modelo —, com o resto da tripulação a um clique.

- **Contraste antes de tudo:** fundo quase preto e texto quase branco, e os
  cinzas quietos legíveis — o mais apagado em cerca de 6:1, onde o do grafite
  ficava perto de 3,6:1. Os tokens moram só em `.look-crew`: a grade fica como
  estava.
- **A fonte do Kiro é a AWS Diatype, comercial, e não pode vir no app.** A Crew usa
  a Geist no texto e a Fragment Mono no código — as duas livres (OFL), a segunda a
  mesma que o Kiro usa para código.
- **Fonte é sempre arquivo próprio no build:** embutida como `data:`, a CSP
  (`font-src 'self'`, D2) a recusa e a face cai em outra sem aviso — foi o que a
  régua visual mostrou com a Fragment Mono.

Descartados: substituir a grade (o pedido foi não desmanchar); copiar a Diatype
(licença); afrouxar a CSP para `data:`.

**D33. A Crew é um painel com trilho de seções, só com o que o daemon tem.** A
primeira Crew (D32) era a lateral da grade com um terceiro painel; estudada contra
um painel de agentes — reimplementado, sem marca, imagem, texto ou nome dele —,
faltavam seções e uma lista de sessões própria. A Crew troca a lateral; a grade fica.

- **Trilho:** Painel (com quantas esperam você), Agenda, Memória, Skills, Apps,
  Conhecimento e Configurações; lembrado por janela (D7); ícones `lucide-react`.
- **Painel:** à esquerda, toda conversa da lista (N3), com a busca do ⌘K (D27), em
  "Ativas" (esperando você em cima), um grupo por projeto com a última semana e
  "Mais antigas", recolhido; clicar numa viva a segue, numa terminada a mostra, e
  escrever a continua (D21). No meio, a conversa inteira (D31). À direita, abas
  Mudanças (D8), Pronto (selo, critérios, tocou a régua, sem medição) e Sessão
  (projeto, branch, modelo e a troca, modo, turnos, contexto). Nada de pull
  request, commits ou revisão: o daemon não sabe deles. "Nova" lista os projetos
  conhecidos e "Outra pasta…"; o ⌘N vai ao seletor de pastas.
- **Skills e Memória leem o daemon** (`GET /v1/skills`, `/v1/memory`, N6): as
  skills com origem, arquivo, gatilhos e o que as faz pedir permissão, e os
  arquivos recusados; a memória por tipo, marcando a velha e a que fica fora do
  que a sessão lê, com os blocos ilegíveis. Skills mostra também as que entraram
  nos turnos acompanhados (`skill.loaded`). Daemon sem a rota (404) diz que ainda
  não expõe, com a resposta dele; outra recusa é falha, nunca lista vazia. Agenda,
  Apps e Conhecimento não têm nada no núcleo e dizem "ainda não", com o porquê.
- **Configurações:** a versão da tela, o daemon (D19) e os modelos de cada projeto
  (D28); nada finge guardar o que o daemon não tem.

Descartados: dados de exemplo (parece funcionar e não lê nada); a lateral da grade
na Crew (dois jeitos de achar a mesma conversa); esconder seção sem dado.

**D34. O texto do modelo é desenhado como o markdown que ele é.** O modelo
escreve markdown, e a janela mostrava os asteriscos e os traços crus: uma
resposta com títulos em negrito e listas chegava como `**Plano**` e `- item`.
Agora o texto do modelo desenha parágrafos, negrito, itálico, código em linha,
blocos de código cercados (fonte mono, rolagem horizontal, sem cores de
sintaxe), títulos de h1 a h4 — modestos, porque moram dentro de uma conversa, e
os níveis 5 e 6 viram 4 —, listas com e sem número e aninhadas, citações,
linhas horizontais, tabelas simples e links. Uma quebra de linha simples
continua sendo quebra, como o texto aparecia antes.

- **O `marked` só lê; quem desenha é o React.** O lexer do `marked` 18.0.14 (MIT,
  sem dependências, cerca de 45 KB minificado e 14 KB comprimido, versão fixa) dá
  a árvore de tokens, e `src/renderer/Markdown.tsx` a mapeia para elementos.
  Nenhuma string de HTML é montada nem injetada: o HTML que o modelo escreve
  aparece como o texto que é, escapado pelo React, e a CSP (D2) fica como está.
  O `react-markdown` com `remark-gfm` faria o mesmo trazendo o ecossistema
  unified/micromark — dezenas de pacotes — para uma tela que só lê.
- **Link sai para o navegador, e só `https`.** O link é um `<a target="_blank">`
  que o processo principal já manda ao navegador pelo `setWindowOpenHandler` e
  que nunca navega a janela. Qualquer outro endereço — `http`, `file:`,
  `javascript:`, relativo — fica como texto, com o endereço no título. Imagem é
  o texto alternativo: nada é buscado de onde o modelo apontou.
- **Durante o streaming cada prefixo é desenhado como o que é até ali:** um `**`
  sem par fica como chegou até o par chegar, e uma cerca sem fim é código até
  onde o texto chegou. Nada é inventado para fechar o que ainda não chegou.
- **Estilos só de tokens** (`styles/markdown.css`), com tamanhos relativos ao
  texto em volta: o mesmo serve o palco, a janela compacta da grade e a Crew, e
  qualquer tema que os tokens nomeiem.

Descartados: escrever o parser à mão (é um formato inteiro, com casos que um
parser mantido já resolve); usar o HTML que o `marked` gera, sanitizado
(`dangerouslySetInnerHTML` com texto do modelo, e mais uma dependência para
limpar); fechar sozinho o `**` sem par durante o streaming (adivinha, e erra em
`2**3`).

**D35. Um tema claro, sóbrio, à escolha; o escuro continua o padrão** (emenda o
"escuro só" de D29). Sistema (segue o `prefers-color-scheme`), Claro ou Escuro,
nas Configurações da Crew e, na grade, no ⚙ do pé da lateral — que antes dizia
"ainda não" e agora abre as três opções. Lembrado por janela no `localStorage`
(como D7 e D32); valor ilegível ou armazenamento que recusa é dito num aviso, e
a janela fica no escuro. Sem escolha guardada, Escuro: as réguas continuam
medindo a janela escura, a 0,00% nas duas.

- **Só tokens.** O resultado vira `data-theme` na raiz do documento, e o claro
  redefine os `--dc-*` em `:root[data-theme='light']` (grafite) e em
  `:root[data-theme='light'] .window.look-crew` (Crew, D32). As cores que ainda
  estavam escritas nos estilos — bordas de hover e foco, o verde do selo de
  modelo, o véu e a sombra da busca, os pontos das janelas — viraram tokens,
  com o mesmo valor no escuro.
- **A paleta:** chão neutro quase branco, nem creme nem azulado, e texto quase
  preto; os cinzas quietos legíveis, o mais apagado acima de 4,5:1 em todo chão.
  O lima não se lê no branco (1,2:1): o acento escurece para um oliva-lima
  (`#4b6b00`) em texto e marcas, e o lima vivo (`#c6f36b`) fica só onde é
  preenchimento sob tinta escura — o botão de enviar. Âmbar, verde e vermelho
  escurecem para serem lidos como texto.
- **Sem piscar ao abrir:** o tema mora no renderer, então a janela nasce
  escondida e aparece no primeiro quadro pintado (`ready-to-show`); se ele não
  vier em 5 s, aparece assim mesmo, com o motivo no log. O `backgroundColor`
  fica o chão escuro, que só aparece atrás de um redimensionamento.

Contraste (WCAG, texto sobre chão; "sel" é a seleção sobre o painel):

| grafite claro | chrome `#eeeff0` | panel `#f9f9fa` | raised `#ffffff` | sunken `#f3f3f4` | sel |
|---|---|---|---|---|---|
| text `#1c1d20` | 14,64 | 16,02 | 16,85 | 15,20 | 13,76 |
| dim `#4a4c52` | 7,45 | 8,16 | 8,58 | 7,74 | 7,00 |
| faint `#5f6168` | 5,37 | 5,88 | 6,18 | 5,58 | 5,05 |
| accent `#4b6b00` | 5,36 | 5,87 | 6,17 | 5,57 | 5,04 |
| ok `#1a6e30` | 5,50 | 6,02 | 6,33 | 5,71 | 5,17 |
| err `#b52f1c` | 5,38 | 5,89 | 6,20 | 5,59 | 5,06 |
| warn `#8a5800` | 5,25 | 5,74 | 6,04 | 5,45 | 4,93 |

| Crew clara | chrome `#f7f7f8` | panel `#ffffff` | raised `#f2f2f3` | sunken `#f9f9fa` | sel |
|---|---|---|---|---|---|
| text `#18181b` | 16,55 | 17,72 | 15,84 | 16,84 | 15,13 |
| dim `#3d3d44` | 10,06 | 10,77 | 9,63 | 10,24 | 9,20 |
| faint `#56565f` | 6,78 | 7,26 | 6,49 | 6,90 | 6,20 |
| accent `#4b6b00` | 5,77 | 6,17 | 5,52 | 5,87 | 5,27 |
| ok `#17723a` | 5,60 | 5,99 | 5,36 | 5,70 | 5,12 |
| err `#b42318` | 6,14 | 6,57 | 5,88 | 6,25 | 5,62 |
| warn `#855300` | 6,06 | 6,49 | 5,80 | 6,17 | 5,55 |

O botão de enviar, tinta `#16181b` sobre o lima `#c6f36b`: 13,92. No aviso de
aprovação (âmbar a 10% sobre o painel), o texto fica em 13,95 e o âmbar em 5,00.

Descartados: o seletor de tema na barra do topo, ao lado do seletor de versão
(mexe nos pixels que as réguas medem, e a grade deixaria de estar a 0,00%);
seguir o sistema por padrão (as réguas passariam a depender do tema da máquina);
o lima vivo como acento no claro (texto e marcas ilegíveis); trocar o
`backgroundColor` da janela pelo processo principal (ele não lê o `localStorage`
do renderer, e um canal só para isso é mais peça que esconder até o quadro).

**D36. O modo da sessão muda pela janela, como na TUI.** Plano, Assistir e Auto
existiam só no `/mode` e no ⇧⇥ da TUI; a janela mostrava o modo e não o mudava.
O chip do modo — no campo da janela da grade, no da conversa maximizada e na aba
Sessão da Crew — vira o botão de um menu com os três, e escolher pede
`POST /v1/sessions/{id}/mode`. O ⇧⇥ passa ao próximo (plano → assistir → auto →
plano, e de um par sem nome para plano), na conversa em foco: a maximizada, senão
a janela em foco da grade ou a do meio da Crew.

- **Nada muda antes do daemon dizer.** O chip troca quando chega o
  `session.mode_changed` da sessão, não quando se escolhe: um pedido recusado
  nunca aparece como aceito, e a recusa vem num aviso com a razão do daemon. A
  troca não interrompe um turno, e o menu diz isso.
- **O chip continua desenhado como era** — `sandbox · modo` —, e as telas de
  referência não mudam. Auto vem no tom de aviso, como na barra da TUI: sem
  fronteira entre o agente e a máquina é o único fato ali que merece a cor.
- **Um par que não é nenhum dos três é "personalizado (sandbox · política)"**,
  nunca o modo mais próximo — o `mode` vazio do fio, como a TUI lê. A política de
  aprovação não vem no `Session`, só o sandbox, então o chip diz o sandbox e que a
  política é outra, sem nomeá-la.
- **⇧⇥ é do campo e de lugar nenhum.** Num botão, continua sendo voltar um
  controle, como no navegador; sem isso, quem navega pelo teclado perde o
  caminho de volta.

Descartados: mudar o chip na hora e desfazer na recusa (otimista: por um momento
a janela afirma uma fronteira que o daemon não aplica); pedir confirmação para o
auto (a TUI decidiu que o aviso é o tom que fica, não uma pergunta que se
responde uma vez sem ler).

**D37. Um comando que o agente propôs pode ser rodado pela pessoa, num terminal
logo abaixo da conversa, e o resultado volta ao agente.** Pedido depois de uma
aprovação negada (`cd /tmp && rm -rf … && npx create-next-app …`, regra
`*rm -rf*`): a pessoa quer rodar ela mesma — às vezes com `sudo`, que pede a
senha num terminal de verdade — e que o agente saiba que rodou e o que deu.

- **Copiar e Rodar no terminal** em todo comando proposto: no cartão de
  aprovação e em cada linha `bash` do fluxo. Aparecem ao passar o ponteiro ou
  com o foco do teclado, sobre o fim da linha e fora do layout: o fluxo lê como
  antes, e as duas referências visuais seguem a 0,00%. Copiar passa pelo
  processo principal (`clipboard`), porque a API do navegador pede uma
  permissão que a janela não pede.
- **O terminal abre logo abaixo do fluxo**, acima do campo, na janela da grade,
  na conversa maximizada e na conversa da Crew — as três desenham a mesma
  `ConversationBody`. Abre na pasta da sessão com o comando **digitado e não
  rodado**: a pessoa lê, edita (põe `sudo`) e aperta Enter. Um por conversa;
  outro comando abre outro shell; fechar encerra o shell. As teclas digitadas
  nele são do shell — ⌃K e ⌃N inclusive —, não da janela (D31).
- **Um PTY de verdade no processo principal (`node-pty`), desenhado pelo
  xterm.js no renderer**, por canais nomeados — criar, escrever, redimensionar,
  saída, fim, encerrar —, conferidos no processo principal: só as janelas do
  app; cada terminal pertence à janela que o abriu, e outra janela não escreve,
  não redimensiona, não encerra nem ouve; tamanho em células; o comando com até
  16 KB; até 8 terminais no app. **A pasta é a que o daemon lista para a
  sessão:** a janela diz a sessão, nunca o caminho. O renderer continua sem
  Node (D2). Janela que recarrega ou fecha leva os seus shells junto; sair do
  app encerra todos.
- **O shell é o da pessoa, como login** (`$SHELL -l`), para ter o PATH que um
  terminal dela teria — app de interface no macOS herda quase nenhum. zsh e
  bash leem os arquivos da pessoa e depois aprendem a marcar o prompt com as
  sequências do FinalTerm (OSC 133): `C` quando um comando começa, `D;código`
  quando o prompt volta, com o comando como o shell o rodou. O zsh entra por um
  `ZDOTDIR` do app que lê o da pessoa e devolve; o bash, por um `--rcfile` que
  lê o perfil como um login leria. Outro shell roda como é, sem marcas, e o
  resultado vai só pelo botão.
- **Digitado, nunca rodado.** No zsh, como colagem entre colchetes (bracketed
  paste), então nem um comando de várias linhas roda sem Enter; em outro shell,
  só uma linha — várias são recusadas, porque ali a quebra rodaria. Nada que
  rode, interrompa ou feche a colagem é digitado: CR, ESC e os outros
  controles, menos tab, recusam o comando com o porquê. Digita depois do
  primeiro prompt marcado, quando o editor de linha liga a colagem
  (`ESC[?2004h`) — antes disso o terminal ainda ecoa o que chega, e os escapes
  da colagem apareceriam na tela —, 250 ms depois do prompt quando nenhum
  editor se anuncia (bash 3.2), e em 3 s quando o shell nunca marca.
- **Quando o comando termina** (a marca `D` depois de um comando), o terminal
  oferece mandar o resultado e **mostra o texto antes**: "Rodei no terminal:
  `<cmd>` — saída (código N):" e a saída num bloco de código. Nada vai sem o
  clique em Enviar. "Enviar resultado ao agente" faz o mesmo à mão a qualquer
  momento: o comando rodando e a saída até ali, sem código, ou o último que
  terminou. Vai pelo mesmo caminho do campo: turno se a conversa está parada,
  correção se roda (D23); uma recusa do daemon é dita como a do campo.
- **O que vai:** a saída sem ANSI, com o que um `\r` deixou na linha, as
  últimas 200 linhas e no máximo 16.000 caracteres, dizendo quando cortou e de
  quantas. **Sem segredo:** a senha do `sudo` não tem eco, então não está na
  saída; mesmo assim toda linha de prompt de senha (`[sudo] password for`,
  `Password:`, `Senha:`, `…'s password:`) sai inteira. O que passa pelo PTY não
  é registrado em log nem guardado pelo processo principal; uma tecla recusada
  é dita no log pelo motivo, nunca pelo que foi digitado.
- **As cores do terminal vêm dos tokens** `--dc-*`, e ele segue o tema da
  janela (D35) quando muda; azul, magenta e ciano, que a janela não usa, são
  tokens `--dc-term-*` próprios, nos dois temas.
- **Dependências, em versões exatas:** `node-pty` 1.2.0-beta.15 — a 1.1.0, a
  última estável, publica o `spawn-helper` do macOS sem permissão de execução,
  e todo shell falha com `posix_spawnp failed`; a beta.15 (agosto) corrige.
  `@xterm/xterm` 6.0.0 e `@xterm/addon-fit` 0.11.0. O xterm carrega só quando o
  primeiro terminal abre: são uns 340 KB.
- **O módulo nativo no build:** o `node-pty` é N-API, então um binário serve ao
  Node e a qualquer Electron, sem recompilar contra os headers dele — macOS e
  Windows usam o pré-compilado que vem no pacote; Linux compila uma vez no
  `npm ci`. O Forge não o recompila (`rebuildConfig.ignoreModules`). Fica fora
  do bundle do Vite (acha o binário ao lado dos próprios arquivos); no
  empacotamento, um gancho copia o pacote para o `node_modules` do app, fora do
  asar (`asar.unpack`), porque o `spawn-helper` é executável e o `.node` é
  biblioteca, e nenhum dos dois roda de dentro de um asar.

Descartados: rodar o comando pelo daemon por fora da regra (é a regra que negou,
e a senha do `sudo` não teria onde ser digitada); um `child_process` sem PTY (sem
TTY o `sudo` não pede senha, e nada interativo funciona); mandar o resultado
sozinho quando o comando termina (a pessoa decide o que o agente vê); adivinhar
o fim pelo silêncio da saída, sem marca (um comando que pensa calado pareceria
terminado).

**D38. Modelo e modo escolhidos antes de a sessão abrir.** Nova sessão abria
sempre no padrão do daemon; trocar depois custava uma sessão nova (D28) ou um
turno no modo errado. A janela que pergunta onde (D31) e o "Nova ▾" da Crew
ganham "+ modelo e modo": ligado, escolher o projeto — ou "Outra pasta…", depois
do seletor — leva a um formulário com o modelo, da lista que o daemon dá para
aquele projeto (`GET /v1/models`, N5, com "sem medição" onde for o caso), e o
modo (Padrão, Plano, Assistir, Auto), e "Abrir sessão" abre ali. Desligado, que é
como começa, escolher abre na hora, como antes.

- **O que não se escolhe não vai.** "Padrão do projeto" e "Padrão" não mandam
  `model` nem `sandbox_mode`: o padrão continua sendo do daemon, não uma cópia
  dele na janela.
- **O modo vai como o sandbox que ele mapeia, e depois como modo se precisar.** O
  `CreateSessionRequest` só leva o sandbox (plano → read-only, assistir →
  workspace-write, auto → full-access); a política de aprovação vem da
  configuração, então a sessão pode abrir com o sandbox certo e outro modo — ou
  nenhum dos três. Quando o modo que o daemon respondeu não é o escolhido, a
  janela pede `POST /mode` logo em seguida (D36). Se ele recusa, a sessão fica
  aberta como abriu, e um aviso diz em que modo ela não está, com a razão.
- **A lista de modelos é do projeto escolhido**, por isso o formulário vem depois
  dele: perfis do projeto e do usuário resolvidos como a sessão os resolveria.
  Daemon que não lista diz isso no lugar da lista, e a sessão ainda abre no
  padrão.

Descartados: o formulário sempre, antes de qualquer projeto (um passo a mais no
gesto mais comum, que é abrir com o padrão); a lista de modelos montada fora do
projeto (ofereceria perfis que aquele projeto não resolve); só o sandbox, sem o
`POST /mode` (a sessão abriria sem o modo pedido, dizendo outro no chip).

**D40. Desfazer, resumir e rodar um comando, pela janela.** O `/undo`, o
`/compact` e o `!` eram só da TUI; na janela, um texto começando com `/` ia ao
modelo como mensagem, em silêncio. Agora:

- **O ⋯ da conversa** — na janela da grade e no topo da conversa na Crew — abre
  "Desfazer o último turno" (`POST /undo`), "Resumir o contexto"
  (`POST /compact`) e "Rodar um comando…", que põe o `!` no campo com o cursor
  depois dele. Com um turno rodando ou esperando, o daemon recusa os três, e o
  menu diz isso em vez de oferecê-los.
- **O desfazer diz as duas metades**, como a TUI: o que voltou e o que ficou
  intocado porque mudou depois do turno — o arquivo que não voltou é o que mais
  importa saber. Um turno que não mexeu em arquivo diz isso.
- **O resumir diz quando não havia o que resumir.** O daemon não emite evento
  nenhum nesse caso, e a resposta é o único lugar que o diz; quando resume, o
  `session.compacted` também aparece no fluxo, como já aparecia.
- **`!comando` roda o comando na sessão** (`POST /exec`), fora de um turno, e a
  chamada chega como as outras, marcada `typed`: a janela a mostra aberta, com a
  saída inteira, porque quem digitou o comando é quem quer ler o que ele
  imprimiu.
- **Texto que começa com `/` não vai mais ao modelo.** `/undo`, `/compact` e
  `/mode <plan|assist|auto>` fazem o que dizem; qualquer outro é "comando
  desconhecido", com os que existem, e o texto fica no campo. Uma paleta de
  comandos é outro PR. Numa conversa terminada, comando não a continua: só uma
  mensagem.

Descartados: deixar o `/` desconhecido ir ao modelo (a pessoa pediu algo ao
harness, e o modelo responderia como se fosse com ele); confirmar antes de
desfazer (a TUI não confirma, e o que não volta é dito, arquivo por arquivo).
