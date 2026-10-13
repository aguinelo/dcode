# Decisões do desktop — D21 em diante

A continuação de [`DECISIONS.md`](DECISIONS.md), que tem a introdução, D1 a D20, as
decisões em aberto, as lacunas e os pedidos ao núcleo. A numeração é uma só.

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
segundo tema é um bloco de tokens (D6), não uma caça. Descartados: manter o v2 e
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
