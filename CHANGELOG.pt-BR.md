# Changelog

🇬🇧 [English version](CHANGELOG.md)

Registro vivo do que muda no dcode. **Toda alteração entra aqui**, no topo, na
mesma branch que a faz — não depois, não em lote.

O estado atual fica na primeira seção e é reescrito junto com cada entrada. São
um arquivo só de propósito: status que mora separado do log é status que
envelhece sozinho, e este repositório já tem cicatriz de coisa declarada que
ninguém mantém.

Detalhe fino de decisão continua nos changelogs por família, em
`docs/specs/architecture/<família>/changelog/`. Aqui fica o que mudou e por quê,
em uma linha cada.

---

## Estado atual — 25 de agosto de 2026

**O que é.** Harness de codificação agêntica em Go: um daemon, um cliente de
terminal e o laço do agente entre os dois, num binário estático único, sem cgo
fora do pacote isolado.

**Onde está.**

| | |
|---|---|
| famílias de spec | 13, com 108 changelogs de decisão |
| contratos comportamentais | 43 declarados |
| **contratos medidos contra modelo** | **3** |
| cobertura | 94,1%, com gate em 90% |
| CI | matriz macOS + Linux, gate sobre a **união** dos perfis |
| versão publicada | **0.6.1** |

**Como se instala.** `curl … install.sh | sh`, ou `go install`. Nada mais precisa
ser instalado antes — de rustup, bun, deno, nvm, k3s e uv, nenhum exige ferramenta
externa de verificação, e a primeira instalação é o pior momento para pedir.

O SHA-256 roda sempre. O que diz que o digest deve ser aquele chega por duas rotas
independentes, e **qualquer uma cobre release substituído**: o digest que o
instalador carrega, commitado na `main`, onde mudar uma linha é um commit visível,
e a assinatura cosign, usada quando por acaso está no PATH. O `dcode update`
aplica a mesma regra lendo os digests do instalador da `main`.

Homebrew ainda não é canal — publicava-se num tap que nunca havia sido criado.
Removido em vez de deixado rodando; o `docs/ROADMAP.md` §9 diz o que seria preciso.

**A interface.** A conversa fica com o terminal. A coluna de arquivos nasce
escondida e `^B` a invoca; a lista de conversas é sobreposição em `^R`, que é o
que essa tecla significa no shell de onde ela veio; o painel abre no seu piso e
cresce do que sobra. Toda pergunta abre com uma régua, então uma tela de rolagem
tem um limite dentro dela. Delegação é um card com os filhos dentro, e o filho
que não respondeu é nomeado ali, com o motivo. Chamada de ferramenta aparece no
instante em que começa a chegar do modelo, e cruzar fronteira é perguntado no
fluxo, na raia dela, ficando no lugar com a resposta assim que tem uma.

Esse formato veio de uma medida, não de uma preferência. Reproduzindo uma sessão
real gravada em quatro larguras, a coluna e o painel tomavam 61 de 132 colunas e
deixavam 71 para a conversa, enquanto a mesma sessão em 99 colunas — onde as
duas somem — dava 99. **Alargar o terminal encolhia o texto**, e a virada era uma
coluna só, porque dois limiares estavam no mesmo cem. O que a coluna guardava era
uma segunda cópia do que o fluxo tinha acabado de dizer.

**Onde está o teclado.** A área de digitação é um campo com moldura, porque a
única pergunta sem outra resposta na tela é onde vão as letras que você digita.
Uma linha começando com `!` não é enviada ao modelo — ela roda, pela mesma
ferramenta e pela mesma fronteira, e o campo avisa desde o primeiro caractere.
Nada nessa moldura carrega estado: uma versão anterior a apagava enquanto o
fluxo tinha o teclado, e o teste dela perguntou se aquela distinção sobrevivia
sem cor. Não sobrevivia.

A cópia é `^O`. Foi `v` duas vezes, e a segunda é a instrutiva: a primeira
correção exigiu o cursor no fluxo, o que **estreitou a regra em vez de
aplicá-la**. A linha de digitar é sempre uma linha em que se digita, então
nenhuma condição podia satisfazer "letra não é atalho" — só devolver a letra.

**O que as guardas não conseguiam ver.** Oito dos defeitos corrigidos em 24 de
agosto tinham guarda escrita exatamente para eles, e toda guarda perguntava
sobre um conjunto que já conhecia. A guarda de desenho de caixa derivava os
glifos proibidos das duas tabelas, e a tela de aprovação — desenhada de
literais, em inglês, a única tela que pergunta se uma fronteira pode ser cruzada
— estava fora das duas, de dois jeitos diferentes, achado duas vezes no mesmo
dia. A guarda de largura dividia em quebras de linha antes de medir, então linha
partida em duas media como duas linhas curtas. A guarda de linha em branco
trimava certo e nunca tinha visto prosa. Cada uma agora é feita como pergunta
sobre a tela inteira, e não sobre uma lista.

**Segurança, em dois eixos.** Contenção é o sandbox — Seatbelt no macOS,
bubblewrap no Linux, com fronteira testada contra o kernel e exercitada na CI.
Autorização é a política de aprovação mais as regras. Os dois são ortogonais, e
essa separação é o que permite ser permissivo sem ser inseguro.

Hoje o sandbox: esconde os cofres de credencial por default (`~/.aws`,
`~/.gnupg`, `~/.kube`, `gcloud`, `~/.netrc`, `~/.docker/config.json` e a própria
chave do dcode); mantém o socket de runtime de contêiner fora de alcance;
concede socket e caminho gravável **por nome**; e esconde `~/.ssh` assim que o
socket do `ssh-agent` é concedido — porque aí o `ssh` assina sem ler a chave e
esconder sai de graça.

**Delegação.** Um filho delegado escreve, dentro do que declarou possuir, com a
contenção do pai estreitada ao conjunto. Posse é fronteira, não combinado.

**O que este documento não diz.** Que o sistema está verificado. Trinta e nove
dos quarenta e dois contratos nunca rodaram contra um modelo, e o relatório da
suíte imprime isso em toda execução para impedir a leitura contrária.

---

## Não lançado

### Corrigido

- **Um comando digitado se anuncia antes de rodar.** O `!` rodava o comando e a
  tela não dizia nada. O daemon emitia a conclusão e não o anúncio, e o cliente
  monta a linha a partir do anúncio e a completa por id — então a conclusão não
  tinha onde cair e era descartada em silêncio. O comando funcionava; do único
  lado que importa, não acontecia nada. O guarda que veio junto pergunta à tela,
  não aos eventos: nenhum arranjo de eventos que deixe de pôr o comando e a
  saída na frente de uma pessoa está certo.

## 0.6.1 — 25 de agosto de 2026

### Corrigido

- **A coluna mostra o contexto, não o que o turno custou.** O terceiro lugar em
  que o mesmo defeito chegou à tela: o modelo calculava certo, e a coluna
  lateral seguia desenhando `5.9M / 1.0M` a partir da contagem cumulativa — sob
  um medidor calculado da fração verdadeira, então o par discordava do medidor
  abaixo dele e da barra acima. Cada um dos três foi encontrado por alguém
  olhando a própria tela, e cada um foi consertado com um teste que perguntava
  sobre o único lugar onde acabara de ser encontrado. Agora há um guarda que
  pergunta à tela inteira, em quatro larguras, e ele pega os três.
- **Um alvo que não é caminho mantém a cabeça.** A lista de recentes cortava
  todo alvo no que vinha depois da última barra, o que é certo para um arquivo e
  errado para uma URL: `.../trips/lowest-price?from=maringa-pr` aparecia como
  `lowest-price?from=maringa-pr`, que lê como um arquivo que ninguém tem. Quem
  decide é `looksLikePath`, a mesma decisão que a linha de ferramenta já toma.

### Alterado

- **O daemon e o emissor estão cobertos.** Dez testes recuperados de trabalho
  que nunca entrou: os fragmentos do emissor, as marcas do plano e os ramos
  opcionais do daemon. `internal/app` de 92,3% para 93,0%.
- **A memória aprendida está no repositório.** `.dcode/memory.md` estava fora do
  controle de versão, e a spec diz por extenso o que ela deve ser — versionada
  pelo usuário. Uma memória que vive só na máquina que aprendeu é uma memória
  que a próxima pessoa não recebe.
- **A regra das 500 linhas nunca foi aplicada, e a árvore não está nem perto.**
  Dez arquivos de produção e dezessete de teste passam dela, o maior com 1915.
  Registrado em `docs/ROADMAP.md` em vez de consertado: quebrar vinte e sete
  arquivos não torna nada mais correto, e escrever o guarda primeiro põe vinte e
  sete arquivos vermelhos de saída — o erro que a §5 já registra sobre o portão
  de cobertura.

## 0.6.0 — 25 de agosto de 2026

### Adicionado

- **`/update` por dentro.** Verifica, confere a assinatura e o digest, e
  substitui o binário — o mesmo atualizador que o comando monta, com as mesmas
  recusas: build local não é substituído, um pin é respeitado, e andar para trás
  não é atualizar. O que ele não faz é reiniciar. Substituir o binário sob um
  processo em execução deixa o processo em execução sendo o antigo, então a nota
  manda reabrir em vez de insinuar o contrário. As duas formas de pedir passam
  pela mesma porta, que é o que mantém a garantia de que nada substitui este
  binário sem ter sido pedido.

- **Um comando que você mesmo roda: `!`.** Uma linha começando com `!` não é
  enviada ao modelo — ela roda. A saída chega à tela como os eventos de
  ferramenta que a transcrição já desenha, e ao histórico como uma mensagem do
  usuário, porque foi ele quem rodou; sem isso o modelo responde sobre um
  workspace cujo estado não pode ver. Passa pela mesma ferramenta `bash` e pela
  mesma política, então um cruzamento é posto à pessoa exatamente como seria se
  o modelo tivesse pedido: `!` é atalho por cima do modelo, nunca por cima do
  sandbox. A área de digitação avisa desde o primeiro caractere, enquanto a
  linha ainda pode ser apagada.

### Alterado

- **Quem decide é a fronteira, não o modelo.** Um relato abria com *"Não vou
  rodar `npm install`… você roda localmente"* e *"Não vou rodar `vitest`… você
  roda localmente"* — uma recusa que ninguém deu, respondida no lugar do
  usuário, devolvendo o trabalho para ser feito à mão. A máquina de aprovação
  existe exatamente para aquele momento e nunca chegou a ser acionada. A
  doutrina dizia que um cruzamento é perguntado e que a recusa é final; nunca
  dizia que **decidir de antemão não cabe ao modelo**. Agora diz, em duas
  frases, a cerca de sessenta tokens por turno — o custo é real e compra de
  volta o sentido inteiro de ter fronteiras que uma pessoa responde. Um contrato
  novo, `boundary-decides` a 90%, mede isso: o juiz pede a tentativa, não o
  sucesso, porque ser negado é a fronteira funcionando.

### Corrigido

- **A linha que você está digitando continua visível.** Maior que a caixa, ela
  era uma linha só e a linha era cortada — então tudo além da borda direita, o
  cursor incluído, ficava invisível enquanto era digitado. Não há como ler o que
  não se vê nem como corrigir um erro que não se acha. A área de digitação conta
  as linhas quebrando, em vez de contando quebras de linha, e o cursor é levado
  junto pela quebra para cair onde o próximo caractere vai aparecer. A quebra é
  por coluna e não por palavra: o que se digita ali costuma ser um comando, e um
  caminho ou uma flag quebrada num espaço lê como dois argumentos.


- **Atualização é algo mais novo, não algo diferente.** O aviso perguntava se a
  versão em execução era diferente da última conhecida, então um binário à
  frente da última release — um build local, ou uma release que a verificação
  em cache de um dia ainda não alcançou — recebia `dcode v0.4.0 is available
  (you have 0.5.0). Run `dcode update`.` Uma oferta de andar para trás vestida
  com a palavra update. O próprio `update` já recusava isso, então a ferramenta
  se contradizia em dois lugares na mesma tela; agora os dois comparam versões
  campo a campo, e o `update` recusa uma release mais velha que a em execução
  mesmo quando pedido diretamente.

## 0.5.1 — 25 de agosto de 2026

### Corrigido

- **O medidor da tela é o que foi consertado.** O medidor de contexto foi
  corrigido no modelo e a barra continuou desenhando da contagem cumulativa ao
  lado dele, então lia `ctx 591%` — numa cor calculada a partir da percentagem
  verdadeira, de modo que o número discordava da própria cor. Reproduzindo um
  registro real de 3163 eventos: `input_tokens 5917178` contra uma janela de um
  milhão é o 591; `context_tokens 363500` é o 36 que a barra mostra agora. O
  teto passou para a função que transforma o número em texto, porque estava
  escrito só num campo por onde a tela não passava — e a guarda passou a
  perguntar à tela em vez de ao campo que ela mesma tinha acabado de ver mudar.

## 0.5.0 — 24 de agosto de 2026

### Alterado

- **A aprovação está no fluxo, e fica lá.** O modal saiu: a pergunta é desenhada
  onde foi feita, numa quarta raia, e respondida permanece no lugar com a
  resposta no lugar das teclas. A caixa era lida como se fosse ela a garantir a
  RN-6, e nunca foi — quem é dono do teclado enquanto há fronteira pendente é o
  cliente recusar entregar a tecla ao campo, e isso não mudou. O que a caixa
  fazia era esconder o trabalho que estava sendo julgado e depois se apagar,
  levando junto o registro mais durável que uma sessão produz. A resposta agora
  cai na pergunta por `ApprovalID`: com duas fronteiras em voo, "a última" grava
  uma decisão que ninguém tomou.

### Corrigido

- **Continuar uma conversa longa abre a conversa.** `dcode -c` desenhava a tela
  de abertura e saía, deixando as respostas do terminal às perguntas que ele
  mesmo tinha feito digitadas no prompt seguinte. Três falhas em fila, cada uma
  escondendo a próxima. A resposta da criação da sessão era montada *antes* de a
  conversa continuada entrar nela, então dizia que a sessão não tinha nada; o
  cliente acreditou e pediu eventos a partir de 1, que a retenção já havia
  descartado de uma conversa de dezoito mil; a recusa então era escrita no canal
  de erro e se perdia na corrida com o fechamento desse canal, e o cliente saía
  sem dizer nada. A sessão agora se descreve depois que a conversa está nela e
  informa o evento mais antigo que ainda guarda, o cliente pede a partir dali, e
  um motivo que chega junto com o fechamento é lido antes do fechamento.
- **Um erro fatal sobrevive à tela em que foi escrito.** Ele era desenhado no
  último quadro, e a tela alternativa leva o último quadro embora — então a
  única mensagem de que a pessoa precisava era a que estava garantidamente
  perdida. Falhar era idêntico a não fazer nada.
- **Só uma tag de release dá nome a um build.** Uma tag de backup deixada ao
  lado da branch — `tui-v1`, ponto de restauração e não versão — encobria a
  última release só por ser mais nova, e todo build passou a se chamar
  `tui-v1-dev+411c237`. Dar a um build o nome de algo que não é versão é o mesmo
  defeito que dar a ele o nome da versão que já ficou para trás, que é o que a
  derivação existe para evitar. Script e Makefile agora casam `v[0-9]*`.
- **A versão lê o que a história de fato tem.** `scripts/version.sh` recusava
  derivar qualquer coisa assim que aparecia um merge ou um revert: o assunto de
  um merge não é uma mudança — as mudanças são as dos pais, já no intervalo — e
  `Revert "feat: …"` não casa com convenção nenhuma porque cita uma. Merge é
  pulado; revert é classificado pelo que desfez, já que remover comportamento é
  mudança da mesma classe que a adição foi. A própria recusa também imprimia uma
  palavra por linha, que é como uma mensagem escrita para quem vai consertar
  chega ilegível.

- **O registro para de copiar o que continua.** Continuar copiava o registro
  anterior inteiro para o novo, então uma sessão que continuou uma que continuou
  uma guardava três cópias da primeira — o maior registro desta máquina tem 3,6 MB
  e 18.410 eventos, a maior parte ele mesmo, repetido. A conversa carregada vai
  para o log e não para o registro; o registro guarda o marcador, e ler um segue
  a cadeia para trás. O crescimento agora é linear.
- **Retomar desenha uma vez.** Continuar escreve o log antigo inteiro na sessão
  nova, então a conexão reproduzia todos os eventos dele — 3544 numa sessão real
  — e o Bubble Tea desenha depois de cada mensagem, então a tela repintava 3544
  vezes com a janela seguindo o próprio fim. Agora mostra uma linha enquanto lê,
  com um contador, e a conversa uma vez quando alcança. A linha se move: sessão
  lendo histórico está parada, e fiapo congelado sob a palavra "lendo" é como
  uma tela travada se parece.
- **O medidor de contexto mede o contexto.** Ele marcava `ctx 175%`, que não é
  um contexto 175% cheio — é um turno que gastou 1,75 janelas de entrada.
  `InputTokens` é cumulativo pelas rodadas do turno, e cada rodada reenvia o
  contexto. Agora o daemon diz o que o contexto montado custa, com a mesma
  estimativa que o gatilho de compactação lê, para o medidor e o limiar
  concordarem por construção. Os provedores não podiam responder isso: as duas
  famílias discordam sobre se a contagem de entrada já inclui o cache.
- **Um Chromium alcança o primeiro quadro dentro do sandbox.** Sem
  `mach-register` e um `iokit-open` escopado, qualquer Chromium morria com
  SIGSEGV antes de desenhar nada — Playwright, Puppeteer, Lighthouse, um Electron
  sob teste. Escondia-se porque era **sinal**, não negação: uma recusa diz
  `Operation not permitted` em algum lugar que alguém lê, e um crash diz um stack
  trace sem nada nomear a fronteira. O que a tela mostrava era um navegador
  quebrando, e foi lido como timidez do modelo. É um par — nenhuma das duas
  sozinha passa do crash — e o `iokit-open` está escopado a uma classe, que é o
  que o torna acessível.

### Adicionado

- **A cauda preservada tem dois pisos.** `KeepTurns` contava turnos, e contagem
  é a unidade errada: turnos variam em uma ordem de grandeza, então quatro curtos
  protegiam quase nada — o resumo comia uma investigação de quarenta ferramentas
  e mantinha quatro "ok". `KeepFraction` (0,30) é um piso em tokens da janela,
  medido com a mesma estimativa que o gatilho usa, e vence o que proteger mais. A
  regra de que a tarefa corrente nunca é compactada continua acima dos dois.
- **O contexto avisa que está enchendo antes de ser cortado.** As faixas eram
  calculadas e anunciadas ao modelo, e ninguém as anunciava a quem lê — então o
  resumo aparecia como uma linha dizendo que tinha acontecido, depois do fato,
  sem chance de terminar um raciocínio antes. Agora a travessia chega ao cliente
  no mesmo instante em que chega ao modelo, e o corte diz quantas mensagens
  foram e quantas ficaram, em vez de só que algo aconteceu.
- **Um modo onde letra é tecla.** `esc` de uma linha vazia entra no fluxo, e lá
  dentro `j/k` movem, `↵` abre, `t` percorre os temas e `/` volta a escrever.
  Toda tecla que o modo não nomeia é engolida, que é o que torna letra segura ali
  — o rodapé do design oferece essas letras e põe um badge NAV ao lado, e um
  badge é o nome de um modo. `↑` na borda agora rola em vez de caminhar para
  dentro do fluxo, o que remove o estado por onde o defeito do `v` voltava.
- **Quatro temas: neon, ashes, ember, mono.** Os valores do próprio design, sobre
  um mapeamento de papéis compartilhado — mude de que cor é um título e os quatro
  mudam. O teste de contraste roda nos quatro.
- **A coluna lateral é o painel de diff sobre o de sessão, à direita.** Ela
  substitui a lista de arquivos, e a diferença é o que fez a lista ser escondida
  hoje de manhã: aquela coluna repetia o que o fluxo tinha acabado de dizer, e
  estas duas não — barra da mudança, medidor de contexto, quanto do que foi
  pedido a pessoa permitiu, as últimas chamadas pelo relógio. O padrão foi
  invertido de novo, e o teste de largura já foi escrito de três formas num dia
  só, o que está dito no comentário dele em vez de editado em silêncio.
- **Legenda de raias e barra de navegação.** A legenda aparece uma vez no topo e
  só quando a tela está fazendo mais de uma raia. A barra nomeia as teclas que
  são teclas — o design também oferece `j/k` e `t`, que são letras, e essas
  pertencem a um modo que tome o teclado.
- **A interface tem paleta própria.** Neon: fundo violeta, marca magenta,
  verde-água para o que deu certo, âmbar para a pessoa. Até aqui os papéis
  mapeavam para códigos ANSI escolhidos para caber educadamente dentro do tema
  que o terminal já tivesse, e essa educação é o que fazia a tela ler como cinza.
  Um tema carrega o próprio fundo, que é a decisão e não é de graça: a interface
  deixa de herdar as cores do terminal e passa a possuí-las. Cor desligada não
  recebe nada disso — nenhum escape chega à tela, fundo incluído.
- **O plano entra no fluxo.** Era uma coluna própria; virou um bloco no lugar em
  que o modelo o fez, sempre mostrando o plano atual, atualizado no lugar. O
  painel se dissolve junto e seu contador de teto passa para a barra de status,
  então `-no-panel` e `^P` foram embora — contrato removido de superfície
  estável.
- **O fluxo tem raias.** Toda linha diz qual das três coisas ela é — o que você
  pediu, o que o modelo fez no caminho, o que ele diz — marcada por caractere na
  primeira coluna. Num turno longo, prosa e chamadas de ferramenta se alternavam
  sem nada estrutural entre elas, então recuperar o fio significava ler toda
  linha para descobrir quais valiam a leitura; agora o olho corre pela raia da
  resposta e pula o trabalho. Não custa coluna nenhuma: toda linha já reservava
  duas, e a raia toma a primeira enquanto o marcador de seleção fica na segunda.
  Vem do design `Coding Agent TUI v2`; o que não veio dele, e por quê, está em
  `docs/ROADMAP.md` §11.

## 0.4.0 — 24 de agosto de 2026

Dois relatos de quem usa, e os dois com o mesmo formato do release anterior: uma
regra que tinha sido estreitada em vez de aplicada, e um estado que a tela nunca
mostrou.

### Alterado

- **A cópia é `^O`, e `v` é letra.** Foi `v` duas vezes. Na primeira, um `v` numa
  linha vazia comia a primeira letra de qualquer mensagem começada com ela; a
  correção exigiu o cursor no fluxo, o que estreitou a regra em vez de aplicá-la
  — e o mesmo relato voltou, por um caminho que um teste agora percorre: `↑` numa
  sessão sem histórico caminha para dentro do fluxo, e o `v` seguinte é atalho de
  novo. A linha de digitar é sempre uma linha em que se digita, então nenhuma
  condição podia satisfazer a regra; só devolver a letra. Digitar também devolve
  o foco à linha, para navegar e escrever deixarem de ser dois estados ao mesmo
  tempo.

### Adicionado

- **A área de digitação é delimitada nos quatro lados.** Moldura aqui e não em
  volta de uma chamada, que é para isso que serve uma caixa: a entrada é um
  *campo* — região fixa, que não rola, à qual se volta, e que precisa ser
  encontrada sem ler — enquanto uma chamada é conteúdo, e moldura em volta de
  conteúdo é moldura em volta do que já se estava lendo. A moldura não carrega
  estado: uma versão anterior a apagava enquanto o fluxo tinha o teclado, e o
  teste dela perguntou se aquilo sobrevivia sem cor. Não sobrevivia.

## 0.3.0 — 24 de agosto de 2026

O release de que a interface de fato precisava, e o primeiro em que os defeitos
foram achados **reproduzindo uma sessão real gravada pelo redutor e
renderizando**, em vez de por um estado que eu escolhi. Toda entrada abaixo foi
achada assim, ou por uma guarda que precisou ser reescrita para fazer outra
pergunta.

O formato do release inteiro: *uma regra com uma exceção tem mais*, e *guarda que
pergunta sobre um conjunto só acha o que já está no conjunto*.

### Alterado

- **O texto tem hierarquia em vez de um só apagado.** `StyleDim` significava
  cinco coisas em quarenta e sete lugares; agora são seis papéis, e o mapeamento
  é uma decisão numa tabela só. A primeira coisa que essa decisão mudou: a prosa
  do modelo não é mais desenhada apagada. Apagar a frase põe o olho no nome do
  arquivo dentro dela, e apaga a resposta para isso — então o contraste é
  comprado com o termo, que é uma palavra em vez de um parágrafo. Um terminal tem
  três pesos que sobrevivem a fundo desconhecido, e o invariante diz isso.

- **O painel paga a largura que ocupa.** Ele chegava devendo um quarto da tela no
  instante em que passava a ser permitido, então cruzar de 99 para 100 colunas
  custava vinte e cinco delas de uma vez; agora abre no seu piso e cresce do que
  sobra além desse limiar. E a seção TURNO, que existe para avisar que um teto
  vem chegando, era desenhada desde o primeiro evento de toda sessão — gastando
  trinta e três colunas para dizer `iteração 0/2000`. Aparece a partir de metade
  do teto, e sempre que todos os lugares em vôo estão ocupados.

- **Toda linha de conversa diz quando e quanto.** A sobreposição resolveu a
  largura e deixou o problema real à mostra: quatro linhas diziam a mesma coisa
  porque quatro conversas começaram com a mesma pergunta. A meta toma sua largura
  antes do título — o oposto da regra das linhas de arquivo, porque quando os
  títulos colidem a data é a única coisa que os distingue. `relativeDay` recebe o
  relógio como argumento agora: o picker podia ler um, a sobreposição está dentro
  de um render puro sobre o modelo. E `%d turno(s)` virou plural de verdade.
- **A lista de conversas é invocada, não residente.** `^R` no readline é uma
  busca que se invoca — aparece, você escolhe, some — e tomar a tecla emprestada
  para então fazer dela vinte e seis colunas permanentes contradizia a convenção
  que justificou o empréstimo. Agora é sobreposição, como o modal de aprovação já
  era, com sessenta e quatro colunas para mostrar um título em vez de vinte e
  seis. `RailNav` não se moveu: o cursor, o filtro, o modo de nomear e todos os
  testes deles seguem iguais, e só o desenho mudou de lugar.
- **A coluna de arquivos nasce escondida.** Medido sobre uma sessão real: em 132
  colunas, a coluna e o painel tomavam 61 delas e deixavam 71 para a conversa,
  enquanto a mesma sessão em 99 colunas — onde as duas somem — dava 99.
  Alargar o terminal *encolhia* o texto, e a virada era uma coluna só: 99 dava 99
  ao fluxo, e 100 dava 53. O que a coluna guardava era ainda uma segunda cópia do
  que o fluxo tinha acabado de dizer. `^B` a invoca e ela fica como foi deixada,
  que é o que a tecla significa no editor de onde veio. Mudança de contrato numa
  superfície `stable`, então MINOR no mínimo.

### Adicionado

- **O turno começa com um limite visível.** A pergunta era um sinal no mesmo peso
  da prosa em volta, então uma tela de rolagem não tinha limite em lugar nenhum.
  Toda pergunta abre com uma régua agora, recuada na mesma calha que o resto do
  fluxo usa. Régua e não cor: pergunta destacada só por cor não está destacada
  num terminal sem cor, e este é o ponto para onde o olho rola. Custa uma linha
  por turno e nenhuma coluna.

### Corrigido

- **Marcador que ainda está chegando não é desenhado.** Toda palavra enfatizada
  chega à tela como `**` primeiro e o par dela alguns deltas depois, então
  `1. **` ficava sozinho na última linha do fluxo antes de cada título que o
  modelo escrevia. Descartado só quando está no fim do texto e sem par: marcador
  que alguém abriu e deixou no meio da frase foi escrito de propósito, e texto
  que termina em `**` porque um par fechou ali é um par pronto.

- **Toda tela fala a língua da interface.** Nove literais em inglês estavam no
  código de desenho, o modal de aprovação inteiro entre eles — a única tela que
  pergunta se uma fronteira pode ser cruzada, numa língua que o leitor pode não
  ter, e consentimento dado a uma frase que a pessoa não conseguiu ler não é
  consentimento. A guarda existente pergunta se toda string declarada tem
  tradução e não tem como perguntar se o renderizador as usa; a nova deriva o que
  proíbe do próprio catálogo, então cresce junto com ele.

- **Prosa deixa uma linha em branco entre parágrafos, e ela é vazia.** Dividir
  `"a\n\n"` dá três partes e a última é o fim do texto, não um parágrafo; uma
  fronteira de run num `**` dividia o mesmo texto duas vezes, então um bloco saía
  com três linhas em branco entre duas frases. Elas ainda vinham indentadas,
  virando dois espaços em vez de vazio — e toda regra sobre linha em branco aqui
  compara com `""` ou trima, então as duas leituras passavam por cima. O
  invariante existia e a guarda já trimava; o que ela nunca tinha visto era
  prosa, porque o fixture era feito só de chamadas de ferramenta.

- **Linha de chamada continua uma linha.** Comando de shell quebrado em várias
  linhas era escrito no quadro como várias linhas, e tudo depois dele — coluna
  lateral, divisor, painel — ficava desalinhado até o fim da tela. O achatamento
  ficou em `clipStyled`, por onde passa toda linha de toda coluna, para a
  garantia valer também para o próximo campo de uma linha.
- **Comando guarda o começo, caminho guarda o fim.** A elisão guardava o fim dos
  dois, então quatro buscas diferentes desenhavam quatro linhas idênticas
  dizendo `… | sort -u | head -40`. Agora quem decide é o valor, não a
  ferramenta, reusando a única definição de "isto é um caminho" que o pacote já
  tinha.
- **Linha cortada diz que foi cortada.** A coluna enuncia a regra para título de
  conversa e o painel respondia do outro jeito — `✓ 6 CLI sob demanda com contr`
  acabava ali — enquanto a própria coluna não a aplicava a nome de arquivo, onde
  `client.py` e `client.pyi` se distinguem pelo que falta. As duas colunas
  marcam o corte agora, e elidem antes de estilizar, que é a ordem que o
  contrato da paleta pede.
- **ASCII alcança o modal de aprovação.** O modal era desenhado inteiro a partir
  de literais sem alternativa, então a única tela que pergunta se uma fronteira
  pode ser cruzada era a única que um terminal em ASCII não conseguia ler. Foram
  mais sete vazamentos junto. A guarda perguntava se um *conjunto conhecido* de
  glifos escapou, o que só cobre o que as tabelas já conhecem; agora ela
  pergunta se toda runa é ASCII, sobre um modelo montado inteiramente em ASCII.
- **A coluna lateral conta um arquivo uma vez.** O mesmo arquivo chegava com
  duas grafias — `DCODE.md` numa chamada, `/Users/…/craw/DCODE.md` na seguinte —
  e desenhava duas linhas, dois contadores e um cabeçalho afirmando quinze
  arquivos quando foram onze. Normalizado contra o workspace onde o alvo entra
  no modelo, então a linha de chamada também ganha o caminho curto; caminho fora
  do workspace mantém a grafia que o encontra em vez de virar escada de `../..`.

## 0.2.0 — 23 de agosto de 2026

O release de que a interface precisava, e aquele em que cada entrada abaixo foi
achada por alguém **usando** o produto, não por um teste.

O design v5 está construído: coluna lateral com os arquivos que o turno tocou e
as conversas que o workspace gravou, delegação desenhada como um card com os
filhos dentro, verbo na barra de atividade, e chamada de ferramenta aparecendo no
instante em que começa a chegar em vez de depois de pronta.

Quatro defeitos nele foram achados do mesmo jeito — abrindo o produto e dizendo o
que estava errado — e nenhum pelos testes que deveriam cobrir o mesmo terreno.
Isso fica registrado aqui porque é a coisa mais útil que este release ensinou.

### CLI

- **Build local se chama pela versão para a qual vai.** Ele pegava a última tag,
  então todo build entre dois releases reportava o **anterior**: um binário
  carregando dois dias de trabalho se chamava `0.1.0`, e a única coisa dizendo o
  contrário era um hash de commit que ninguém lê. Alguém viu esse número parado e
  concluiu, com razão, que nada tinha sido instalado. Agora ele deriva do
  `scripts/version.sh`, e o mesmo build diz `0.2.0-dev+7b27519`.

- **`-v` imprime a versão.** Ele respondia *"flag provided but not defined: -v"*
  e em seguida imprimia o uso inteiro, enterrando a única linha que diz o que deu
  errado sob vinte que não dizem. O `-h` já estava ali do lado, e um par de flags
  de uma letra em que só uma existe é o par que se erra toda vez.

### Protocolo

- **A chamada de ferramenta aparece enquanto ainda está chegando.** Num `write` de
  algumas centenas de linhas o modelo transmitia o arquivo inteiro e **a tela não
  mostrava nada** — nem o nome da ferramenta — até a chamada estar montada, e aí
  ela surgia já completa. Silêncio exatamente na parte do turno em que o trabalho
  acontece, que é o que faz uma interface viva se ler como morta.

  Os dois fatos já eram conhecidos e os dois eram jogados fora: o nome e o id em
  `content_block_start`, a contagem de bytes em cada fragmento. O provedor passa a
  emitir `tool_call_opened` e `tool_call_progress`, e o laço os converte em
  `progress` com `kind: "arguments"`.

  **Não é evento novo de protocolo** — é o que já existia, com um campo `Name`,
  porque sujeito que ainda não existe precisa se nomear: o `tool.requested` carrega
  o nome e só chega quando a chamada termina de montar.

  Bytes e não linhas, já que o que chegou é fragmento de JSON e contar linha dentro
  de string escapada pela metade é contar o que ainda não está lá. Sem total: o
  modelo não diz quanto a chamada vai ter, e denominador que ninguém mandou é
  denominador em que alguém acredita.

  Throttle de meio kilobyte, **no laço** e não no provedor: o provedor relata o que
  vê, o protocolo decide o que vale dizer. O primeiro relato é sempre enviado — é
  ele que põe a linha na tela.

  Consumidor que ignora os dois eventos novos vê exatamente a sequência que via
  antes, e há teste para isso.

### Cliente TUI

- **Coluna que se esconde diz que existe.** A coluna lateral some abaixo de cem
  colunas — que é a maioria dos terminais — e **não dizia nada**, então coluna
  construída se lia como coluna não construída, e a tecla que a traz de volta
  (`^B`) estava documentada só dentro da coluna que não estava na tela.

  Achado do único jeito possível: alguém abriu o produto e disse que a interface
  não era a que tinha desenhado. Toda verificação por trás dela tinha sido teste
  Go chamando `Render` em memória, nas larguras que eu escolhi — nunca o binário
  num terminal.

  O limiar continua em cem, porque a razão se sustenta: coluna de 20 num terminal
  de 80 deixa 59 para o fluxo, e diff em 59 colunas é ruim. O defeito era o
  silêncio. O painel do plano já tinha pago essa dívida e já carregava a dica; a
  coluna herdou o comportamento sem ela.

### Protocolo

- **Uma conversa pode receber nome, guardado no registro dela.** Três lugares
  foram considerados: arquivo ao lado da sessão, índice por workspace, e o
  registro. O registro vence pelo critério que decide — **nome de conversa que
  não existe mais é pior que nome nenhum.** A poda apaga transcrições, então o
  vizinho vira órfão e o índice guarda títulos de sessões que ninguém abre. Aqui
  o nome morre com o que ele nomeia.

  E mantém a conta em um: depósito ao lado do log é uma segunda coisa que pode
  discordar dele. Não custa leitura — o `Browse` já varre cada linha de cada
  registro para contar turnos.

  A sequência é **lida antes de acrescentar, nunca presumida**: pôr um número que
  já está no arquivo deixaria duplicata num log cujo contrato inteiro é que não há
  nenhuma. Renomear duas vezes é mudar de ideia, então o último vence.

  Nome vazio devolve o título derivado e não é erro — uma operação com valor zero
  que significa algo é uma coisa a acertar em vez de duas. Caractere de controle
  nunca chega ao registro, porque ele é lido de volta linha por linha e uma quebra
  dentro de um nome faria uma linha parecer duas. Nome longo demais é **recusado,
  não aparado**: guardar metade do que foi digitado em silêncio é como alguém
  acaba com um nome que não escolheu.

  Escreve no registro e não na sessão viva, porque a trilha lista o que o
  workspace gravou e quase nada disso está carregado — um rename que só
  funcionasse na conversa aberta funcionaria na única linha que não precisa dele.

### Cliente TUI

- **`r` e `F2` nomeiam a conversa sob o cursor.** Nomear é modo próprio dentro da
  lista, porque é a única coisa ali que muda algo: enquanto está aberto **toda
  tecla é do nome**, então nada mais é alcançável sem querer no meio.

  O rascunho parte do **nome**, nunca do título derivado. Oferecer o título
  transformaria *dê um nome a isto* em *confirme o que te deram*, e o primeiro
  Enter promoveria um título derivado a nome escolhido sem ninguém decidir. `esc`
  cancela e mantém o que havia.

  Nome dado leva `·`. Sem a marca a coluna mostra dois tipos de afirmação —
  derivado e escolhido — e nada os distingue.


- **Varredura diz quão longe foi, e resultado pousa na chamada dele.**
  `kind: "files"` entra no conjunto declarado: o `grep` diz `n de N` porque tem a
  lista antes de começar, o `glob` manda só a contagem porque ainda está
  descobrindo, e o card mostra `150/184` onde havia reticência.

  O relator viaja **no contexto da chamada, não no `State`**. O State é por
  sessão e compartilhado, então duas varreduras em paralelo escreveriam suas
  contagens pelo mesmo campo e a tela mostraria o progresso de uma sob o nome da
  outra. `Progress(ctx)` nunca devolve nil: ferramenta deve dizer quão longe foi
  sem antes perguntar se alguém escuta.

  Continua sem `kind` para linhas nem testes, e isso é achado, não esquecimento.
  O `read` lê o arquivo inteiro e divide, então aprende o total no mesmo instante
  em que aprende o conteúdo — não existe momento em que "n de 240" seja verdade.
  Contar teste que passou exigiria parsear a saída do `bash`, que o comentário do
  `ToolCompleted` proíbe. **Kind que só poderia ser preenchido desonestamente não
  se declara.**

  Os relatos saem a cada vinte e cinco arquivos, não a cada arquivo: um por
  arquivo poria dez mil linhas que ninguém lê no registro de uma varredura de dez
  mil arquivos.

  **Um defeito latente apareceu no caminho.** O `ToolCompleted` casava com a
  *última entrada rodando*, o que está certo exatamente enquanto uma chamada roda
  por vez — com duas em vôo, o primeiro resultado pousava na linha da segunda.
  Números reais na linha errada. O `Entry.CallID` conserta o roteamento do
  resultado e do progresso, e foi achado porque o progresso precisava do mesmo
  endereçamento, não porque alguém notou a tela errada.


- **`progress`: um evento para "quão longe já foi".** Ferramenta contando
  arquivos e turno contando rodadas são a mesma pergunta feita a sujeitos
  diferentes, então é um evento só, com `tool_call_id` vazio quando o sujeito é o
  turno. Acrescentar superfície versionada duas vezes para um tipo de pergunta é
  como ela sai torta — a segunda sempre responde um pouco diferente.

  `kind` é **conjunto fechado, não palavra para imprimir**: o idioma do daemon
  não é o de quem lê. Só o que alguém de fato emite está declarado, então
  `rounds` e `in_flight` estão lá e `files`/`lines` chegam quando as ferramentas
  emitirem. `tests` provavelmente nunca chega — contar teste que passou exige
  parsear a saída do `bash`, que o próprio comentário do `ToolCompleted` proíbe.

  **Ele entra na sequência**, e essa foi a decisão difícil. Deixá-lo fora do
  `Seq` abriria buraco na única propriedade sobre a qual o registro é construído,
  e registro com buraco é registro cuja reprodução não é confiável sobre mais
  nada. O `message.delta` já é conversador e já está lá; progresso segue ele em
  vez de inventar exceção.

  O teto viaja junto da contagem, porque contagem sem limite responde *quantas*
  quando a pergunta é *quão perto*. E turno que respondeu numa passada não
  reporta rodada nenhuma: não há teto se aproximando, e `0/100` na tela é número
  que significa que nada está acontecendo.

### Cliente TUI

- **O painel mostra onde o turno está.** `iteração 2/100` e `em vôo 2·4`, com a
  contagem mudando de estilo ao se aproximar do teto — esse teto é o item 1 do
  roadmap, o único com evidência medida de dano, e o que faltava era algo dizendo
  que ele vinha.

  O painel passa a abrir só com esses números. A maioria dos turnos não tem
  plano, então o teto ficava escondido justamente no painel que só abria quando
  outra coisa já estava lá. Os números sobrevivem ao turno que os produziu, então
  ele abre no primeiro turno e fica, em vez de aparecer e sumir a cada um.


- **`^R` dá o teclado à coluna.** `↑↓` movem, letra filtra, `enter` continua a
  conversa sob o cursor, `esc` limpa o filtro e depois fecha. É o segundo dos
  três modos que o design nomeia; o terceiro, nomear, continua sem onde ser
  guardado.

  Ser dona do teclado não é floreio: lista que se percorre com teclas que também
  digitam na linha de entrada é lista em que toda tecla faz duas coisas, e a
  única vez em que faz a errada abre a tarde de outra pessoa. O bloco fica
  **acima** do guarda do menu de autocompletar pelo motivo que o changelog da
  cópia registra — dentro dele, o modo nunca teria rodado, e nada diria.

  Cada decisão pequena com seu motivo: o cursor é caractere e não cor, e vence a
  marca de conversa aberta, porque com o teclado ali a pergunta é *qual vou
  abrir*; `↑↓` não dão a volta, reusando o argumento do próprio picker; `esc`
  recua uma coisa por vez; digitar volta o cursor ao topo, já que a lista na tela
  passou a ser outra; filtro sem resultado escolhe nada **e diz isso** em vez de
  ficar em branco; escolher a conversa já aberta não faz nada.

  Uma quinta runa Unicode cravada escapou — o cursor do filtro — e o guarda do
  #243 não a pegou, porque enumerava as runas à mão. Agora ele **deriva** a lista
  dos conjuntos de glifos por reflexão, então marca nova entra na proibição
  sozinha.

### Documentação

- **O que o design v5 pede e o cliente ainda não mostra.** Achado por quem rodou
  a interface e disse que ela parecia mais pobre que o desenho, que é o único
  jeito de uma lacuna dessas aparecer.

  A delegação era a grande e está **construída** (abaixo). O que resta são os
  números do turno no painel — `iteração 2/100`, `em vôo 2 · teto 4` —, e isso é
  **lacuna de dado, não render esquecido**: o protocolo carrega
  `StopMaxIterations` como motivo de um turno ter *terminado* e nada enquanto ele
  roda. Registrado primeiro como esquecimento e corrigido depois de conferir,
  porque os dois têm consertos diferentes e só um é barato.

  É a metade-cliente do item 1 do roadmap. Aquele item é sobre o *modelo* nunca
  saber que está ficando sem rodadas; este é sobre a *pessoa* também não saber,
  no lugar onde ela já está olhando. O evento que responder um deve responder os
  dois.

### Cliente TUI

- **Nenhuma runa de desenho de caixa chega a terminal que não as desenha.**
  Quatro literais separados neste pacote presumiam Unicode — o divisor de coluna,
  a calha do diff, o marcador de rodando e a reticência de caminho — e cada um foi
  achado olhando um render em ASCII, **depois** de o anterior ter sido corrigido.
  O divisor saiu no #241; os outros três saem aqui.

  Então o guarda é sobre a tela inteira, e não glifo a glifo. Um quinto ficaria
  esperando um quinto par de olhos.

- **A coluna lateral lista as conversas deste workspace.** Sob os arquivos, com a
  aberta marcada por caractere e não só por cor. É o `dcode -r` promovido a
  coluna permanente, no modo que o design chama de *passiva*.

  Mesma fonte e mesmo filtro do `-r` — `session.Browse` através de
  `choicesFrom`, lida uma vez no início pela borda, porque duas maneiras de
  listar as conversas de um workspace acabariam discordando sobre quais existem.
  Conversa em que nada foi perguntado continua fora; é a maior parte do que um
  diretório de gravação guarda. O cliente segue sem ler disco: a lista chega por
  `Options`, como o idioma.

  **Nomear conversa não pôde entrar, e isso está escrito em vez de aproximado.**
  O nome que a pessoa dá precisa sobreviver à sessão, e um diretório de gravação
  guarda transcrições, não títulos — o título é derivado da primeira pergunta
  toda vez que é lido. É mudança em `internal/session`, e o `docs/ROADMAP.md`
  passa a carregá-la com os três lugares onde poderia morar e por que só um
  sobrevive à poda. A navegação com `^R` espera em parte pela mesma decisão; o
  `/resume` já faz o continuar.

  Detalhes que a tela decidiu: conversas sozinhas já abrem a coluna, porque
  perguntar só pelos arquivos a esvaziava no primeiro minuto de toda sessão; e
  título cortado diz que foi cortado, com o corte em **células** e não em bytes,
  já que runa não é coluna e é em título com acento que isso erra.

- **Uma coluna lateral mostra o que o turno tocou.** Arquivos, o estado de cada
  um, e a contagem de linhas de quem terminou. `^B` dobra e expande. Largura
  `clamp(20, w/5, 30)`, some abaixo de cem colunas, e a escolha explícita vence
  em qualquer largura nos dois sentidos — as maneiras do painel, exatamente,
  porque responder uma pergunta de dois jeitos daria a duas colunas
  comportamentos diferentes num terminal só.

  **Derivada, não guardada.** O handoff põe um campo `tree` no modelo; aqui é
  função pura sobre `Entries`, que já são a redução do log. Um campo seria uma
  *segunda* redução dos mesmos eventos, e duas reduções podem discordar —
  derivando, "mesma sessão reaberta reproduz a mesma árvore" vira verdade por
  construção e não por cuidado. `Entry` ganhou `Added`/`Removed` como números,
  porque ler a contagem de volta da frase do resumo é o que o comentário do
  protocolo proíbe com todas as letras.

  **Duas camadas, não uma árvore inteira.** A coluna tem 20 a 30 caracteres e
  cada nível de indentação tira dois deles da única parte que identifica um
  arquivo. A linha de pasta carrega o caminho inteiro no lugar.

  Quatro defeitos que só a tela mostrou: a indentação derivando um nível depois
  do primeiro arquivo (profundidade de caminho não é profundidade visual depois
  que uma pasta foi compactada), a contagem impressa duas vezes, o `+38`
  encostado no divisor e lido como moldura, e o divisor cravado em `│` — **o que
  o painel de plano já fazia**, visível só quando uma segunda coluna repetiu.
  Agora os dois seguem o `g.Unicode`.

- **A dica de expansão fala o idioma da interface.** Sob corpo recolhido ela dizia
  `⋯ 42 lines · Tab expande` — uma linha, dois idiomas, nas **duas** interfaces: a
  contagem cravada em inglês e o verbo cravado em português, e nenhum dos dois
  seguia o que o usuário escolheu.

  Mesma família do #238 e achado do mesmo jeito, lendo a saída em vez do diff.

- **Chamada de ferramenta que carrega corpo é um bloco.** Uma linha em branco a
  separa do que está em volta; chamada sem corpo continua uma linha só, porque a
  maioria é de uma linha e moldura em volta de uma linha é caixa em volta de
  nada.

  É pouco assim porque quase todo o §3 do design já estava construído — o `…`
  enquanto roda, a duração só a partir de 500 ms, e a coluna "meta pronta"
  inteira (`240 lines`, `created, 38 lines`, `+24 −2`, matches em arquivos,
  `exit 0`) no `summariseResult`. E o card já existia em unidade de terminal: o
  `detailLines` desenha um `│` à esquerda de toda linha de corpo, que é a espinha
  amarrando corpo e cabeçalho. Faltava o respiro que o handoff pede, não uma
  moldura.

  A borda de runas fica registrada como preferência com o preço nomeado — duas
  colunas e duas linhas por chamada, variante ASCII, e a borda entrando na
  seleção do modo de cópia — e não faria nada que a calha já não faça.

  **O gap vai antes, nunca depois**, e isso foi medido e não chutado: posto
  depois, ele empurrou a linha alterada de um diff para fora de um terminal de 40
  linhas, porque a janela é ancorada no fim e branco final custa uma linha do que
  aconteceu para não mostrar nada.

- **A linha de atividade fala um idioma só, a saída inclusive.** Ela dizia
  `^C interrupts` em inglês sob interface em português. O verbo tornou isso
  óbvio ao sentar do lado — `lendo grep … ^C interrupts` — e meia frase em cada
  idioma se lê como defeito do produto, não como tradução faltando.

  Entrada própria no catálogo, e não o `Interrupt` existente, que é a dica do
  `esc` em outro lugar: duas teclas dividindo uma frase é como uma dica acaba
  nomeando a tecla errada num dos idiomas.

- **A barra de atividade carrega um verbo, e o verbo nunca aparece sozinho.** Um
  gerúndio curto passa a acompanhar a ferramenta que roda — `⏺ lendo grep
  \.Save\(` —, sorteado do conjunto da fase e trocando a cada 20 quadros, que
  são os 2,4 s do design no tick de 120 ms. `DCODE_ACTIVITY_VERBS=0` desliga,
  tirando a palavra e deixando os fatos.

  Sozinho ele seria movimento fingindo informação: a tela parece viva e quem lê
  não aprende nada. Então só é desenhado ao lado de uma ferramenta rodando, em
  `dim` contra o `bold` do fato — o que se mexe é o acompanhamento, o que é
  verdade é a ênfase — e sem ferramenta a linha diz sua palavra única, parada.

  Achado ao construir: `working` era ao mesmo tempo a palavra do estado sem
  ferramenta **e** um verbo do conjunto `other`. Com uma string nos dois papéis,
  ninguém distingue verbo girando de verbo parado — nem o leitor, nem o teste. A
  palavra de fallback também entrou no catálogo de idiomas, onde deveria estar
  desde sempre.

- **O tick para quando a sessão fica ociosa.** Ele já se recusava a avançar o
  quadro, e o comentário dizia por quê — *"tela ociosa que fica repintando queima
  bateria de laptop por informação nenhuma"* — enquanto reagendava assim mesmo,
  então a tela repintava oito vezes por segundo para um número que não se movia.
  A frase estava certa; ela apenas não estava sendo cumprida.

  Ele volta quando um turno começa, com uma guarda para religar exatamente um:
  sem ela todo evento acrescentaria um tick e o contador de quadros dispararia,
  movimento afirmando que a máquina está mais ocupada do que está. Nada se perde
  parando — o `Now` é atualizado em todo evento.

### Sandbox

- **Decisão de fronteira segue o modo, não o sistema de arquivos.** O
  `canonical()` devolvia o caminho cru quando ele ainda não existia — o
  `EvalSymlinks` só resolve o que existe —, então o mesmo diretório canonicalizava
  de dois jeitos conforme *quando* se perguntava: `/tmp/ws` antes de criado,
  `/private/tmp/ws` depois. As comparações de `/tmp` então o testavam contra o
  literal `"/tmp"`, e remontar ou não o workspace por cima do `tmpfs` passava a
  depender do que estivesse no disco.

  Agora ele resolve o ancestral mais profundo que existe e recoloca o resto, e as
  comparações rodam contra `/tmp` como o próprio `canonical()` o reporta.

  Foi achado como um teste que passava ou falhava conforme a máquina, escondido
  um dia atrás do cache de testes do Go. Mas a mesma função alimenta o profile do
  **seatbelt**, e o comentário acima dela já nomeava o perigo que ela causava: no
  macOS, profile nomeando caminho não resolvido *"não concede nada e toda escrita
  falha sem explicação"*. Em produção o `bubblewrap` só roda no Linux, onde as
  duas grafias já coincidem, então nenhuma lista de argumentos muda lá.

  O `TestCanonicalFallsBackToTheInput` afirmava o contrato antigo pelo nome e foi
  reescrito para o novo, não afrouxado; três asserções que fixavam a grafia crua
  de fixtures passam a comparar contra `canonical(...)`, que é o que o `args()`
  de fato monta.

### Documentação

- **O que o design v5 pede e o produto não tem está escrito.** Uma seção nova no
  `docs/ROADMAP.md` nomeia o `refs/design/HANDOFF.md` como fonte, para que a
  especificação seguinte saiba de onde veio o pedido.

  Três itens. Ferramenta **não reporta nada enquanto roda** — o protocolo tem
  `tool.requested` e `tool.completed` e nada entre eles, então as contagens em
  andamento do design não têm origem; são quatro camadas e MINOR no mínimo, e
  isso bloqueia a barra de progresso do card e mais nada. A trilha de sessões
  **lê do disco**, o que é premissa e não fato, e no dia em que um cliente se
  ligar a um daemon noutra máquina a trilha não lista nada — silêncio que se lê
  como defeito se não estiver escrito antes. E a borda completa do card fica
  registrada como preferência visual com o preço nomeado, em vez de virar
  mudança de ideia esperando acontecer.

  Um quarto, que não vem do design: **um teste de fronteira passa ou falha
  conforme o que existe na máquina.** O
  `TestKeepingTheWorkspaceVisibleDoesNotMakeItWritable` depende de `/tmp/ws`
  existir, porque o `EvalSymlinks` só resolve caminho que existe. Ficou um dia
  escondido atrás do cache de testes do Go — a mesma armadilha do número de
  cobertura cacheado, por outro caminho.

### Cliente TUI

- **O teclado do v5 decidido por convenção, e nada declarado ainda.** O handoff
  de design propõe cinco teclas e três colidem: `^E` é fim de linha, `^N` já é
  "descer" no picker, e `^Z` é SIGTSTP em todo terminal que existe.

  Decidido no lugar: `^B` para a coluna lateral (o que o VS Code fez a palavra
  significar), `^R` para a trilha de sessões (o *reverse-i-search* do readline —
  sessão **é** histórico, então tomar o acorde emprestado reforça o sentido), e
  `r`/`F2` para renomear com a trilha dona do teclado. A seção ARQUIVOS não ganha
  **tecla nenhuma**: editor nenhum dá acorde global a cada seção da barra
  lateral, então `^B` abre a coluna e as duas seções estão simplesmente lá. Isso
  apaga uma tecla em vez de acrescentar.

  `^Z` é recusado duas vezes — reatribuir a tecla de suspender é hostil, e
  `/undo` já existe deliberadamente, já alcançando o trabalho delegado pelo
  estado que o pai adota.

  A tabela da seção 7 fica **intacta**. Cada tecla entra nela no PR que a
  implementa, com o teste que a cobra: tecla declarada em spec que nenhum código
  executa é o mesmo defeito que o changelog da cópia dona do teclado registra
  contra si próprio.

### Documentação

- **Referências de design vivem em `refs/design/`, e há uma só.** O handoff v2
  estava commitado em `docs/design_handoff_dcode_tui/`, uma cópia byte-idêntica
  dele estava ao lado como `docs/Interface TUI com Tea Bubble.zip`, e o handoff
  novo v3/v4/v5 chegou sem rastreamento em `refs/design/`. Três cópias do mesmo
  material, duas versionadas, nenhuma dizendo qual vale.

  Tudo passa a viver em `refs/design/`, rastreado, com o zip removido e as duas
  referências em prosa que nomeavam o caminho antigo corrigidas, para que nenhum
  link fique morto. O README de lá é índice: o que é cada arquivo, qual vale, e
  que a spec vence onde ela e um handoff divergirem.

  O handoff em si fica **verbatim, como entregue**. As cinco divergências
  encontradas conferindo as afirmações dele contra o código ficam ao lado, não
  dentro dele — a maior sendo que progresso de ferramenta não existe no
  protocolo (`tool.requested` → `tool.completed`, nada entre os dois), o que
  torna "tarefa como card com progresso" uma mudança de quatro camadas em vez de
  uma de TUI. Reescrever o handoff apagaria a diferença entre o que foi projetado
  e o que se descobriu depois.

## 0.1.0 — 20 de agosto de 2026

O release de que o caminho de instalação precisava. O 0.0.1 foi publicado e então
descoberto ininstalável pelo próprio comando documentado, e tudo abaixo sai de
puxar esse fio — incluindo descobrir que a premissa estava errada duas vezes antes
de acertar.

A versão curta: **nada precisa ser instalado antes**, e o digest que diz qual deve
ser o download passa a viajar por uma rota que o download não alcança.

> **Subir do 0.0.1 exige o script de instalação, não o `dcode update`.**
>
> ```
> curl -fsSL https://raw.githubusercontent.com/aguinelo/dcode/main/install.sh | sh
> ```
>
> Um binário 0.0.1 carrega o código de antes da correção, então ele ainda exige
> cosign e para:
>
> ```
> Updating dcode 0.0.1 → v0.1.0
> dcode: cannot verify the release signature: cosign is not installed…
> ```
>
> O conserto viaja **dentro** do 0.1.0, que é justamente o release que o código
> quebrado precisaria buscar — problema de bootstrap, sem correção possível daquele
> lado. Descoberto rodando a atualização em vez de supor que ela funcionava, e
> escrito aqui porque nota de migração que não é registrada no dia em que se
> descobre é nota que ninguém escreve. Do 0.1.0 em diante o `dcode update` já leva
> o caminho novo.

### Distribuição

- **O release para de publicar num tap que não existe.** O
  `aguinelo/homebrew-dcode` nunca foi criado e o `TAP_TOKEN` nunca foi
  configurado, então o passo saía com zero e avisava a cada release — correto por
  desenho, já que falhar ali reprovaria um release que já tinha dado certo, e o
  efeito foi o v0.0.1 reportar sucesso com um canal que nunca existiu.

  Maquinário que roda e não entrega nada é pior que ausência: ocupa o lugar da
  decisão que ninguém tomou e faz o release parecer completo. O
  `scripts/publish-tap.sh`, os testes dele e o passo do workflow saíram.

  A fórmula fica — gerada a partir do `checksums.txt` assinado, anexada ao
  release, e é o artefato que um tap consumiria. O que saiu junto com o passo foi
  o comando documentado, porque `brew install aguinelo/tap/dcode` apontava para um
  tap que não existe e, se existisse, com o nome errado: o script empurrava para
  `homebrew-dcode`, cujo atalho no brew é `aguinelo/dcode/dcode`. Os dois nunca
  concordaram, e nada os segurava juntos.

  Registrado em `docs/ROADMAP.md` com o que seria preciso para criar, e a
  armadilha de nome a evitar.

- **O `dcode update` também não exige mais cosign.** Era o último lugar que
  pedia um pacote — de uma máquina que já tem um dcode funcionando, o que torna
  o pedido mais difícil de justificar, não mais fácil.

  O binário passa a ler os digests do instalador da `main`, o mesmo arquivo que
  o script de instalação carrega dentro de si, então ele tem a mesma segunda
  rota: um digest que não viajou junto do artefato. A regra é a do instalador —
  o digest carregado **ou** a assinatura, qualquer uma basta.

  Uma diferença, deliberada: onde o script de instalação avisa, o `update`
  **recusa**. Há um binário funcionando na máquina, então parar custa uma versão
  e preserva tudo. Assinatura que *falha* continua abortando, seja qual for o
  digest carregado; tornar uma verificação opcional não pode torná-la
  decorativa.

  O `ErrNoVerifier` deixou de ser veredito e virou o que sempre foi: uma rota
  indisponível. A mensagem dele não diz mais "dcode will not install something
  it could not check", porque essa frase *era* a exigência.

  O `DCODE_UPDATE_INSTALLER_URL` sobrescreve de onde a segunda rota é lida, e
  espelho que sobrescreve `DCODE_UPDATE_URL` precisa sobrescrever este também,
  senão o digest independente continua vindo de upstream. A ligação é asserida
  lendo o comando como dado: campo que o updater lê e nenhum comando escreve é o
  defeito do carimbo de build outra vez.

- **O instalador nunca pede outro pacote.** Ninguém instala ferramenta adicional
  para instalar um binário, e a pesquisa que originou o #223 já trazia a prova —
  de rustup, bun, deno, nvm, k3s e uv, **nenhum** exige ferramenta externa de
  verificação, e quatro não verificam nada. Eu tinha o dado e não tirei a
  conclusão inteira.

  Então o cosign deixa de ser assunto do instalador. O que precisa de duas rotas
  é **release substituído**, e duas coisas independentes o cobrem: o digest
  carregado e a assinatura. Qualquer uma basta, então instalação coberta não diz
  nada — relatar assinatura por conferir enquanto a verificação que importa
  passou por uma rota que não depende dela é ruído vestido de diligência. Quando
  nenhuma das duas cobriu, o aviso aponta o instalador que carrega os digests
  deste release, nunca um pacote a instalar: responder a um problema com "instale
  outra coisa primeiro" é entregar um segundo problema.

  Isso não afrouxa "nunca não-verificado em silêncio" — é por isso que a regra
  pôde encolher. Instalação cujo digest carregado conferiu **é** verificada. O
  SHA-256 roda sempre, o cosign continua sendo usado quando por acaso está no
  PATH, e assinatura que falha continua abortando.

  Três testes que afirmavam a regra anterior foram substituídos, não afrouxados,
  e um perdeu uma asserção; o changelog da família nomeia cada um e o porquê.

- **O aviso tem o tamanho do que ficou por conferir.** Perguntado se dava para
  tirar o aviso do cosign. Não dá — "nunca não-verificado em silêncio" é a linha
  que quatro mudanças gastaram estabelecendo —, mas ele estava grande demais, e
  prestes a ficar falso.

  Ele afirmava que o checksum *"pega download corrompido mas não release
  substituído"*. Verdade sem pino. No instante em que um release fixa o
  instalador, o digest carregado cobre substituição exatamente, que é a razão
  inteira de ele existir. Então, com digest carregado e conferido, o aviso cai
  para duas linhas, perde essa afirmação e deixa de se repetir no fim; sem pino
  ele mantém cada palavra e a repetição, porque a rolagem de um `curl | sh`
  enterra o que apareceu no começo.

  Aviso que exagera é aviso que as pessoas aprendem a pular, inclusive na
  execução em que ele finalmente significa alguma coisa.

### Documentação

- **O README descreve a instalação que existe.** Ele ainda afirmava que o script
  *"verifica a assinatura do release e o checksum, e não instala nada se qualquer
  um dos dois falhar"* — falso desde que o cosign virou opcional, e calado sobre
  o digest que o instalador agora carrega. Passa a dizer o que cada uma das três
  fontes cobre e do que cada uma precisa, e por que o digest carregado é o que
  pega release substituído.

  O Homebrew ia entrar junto, já que todo release publica uma fórmula. Conferindo
  antes: `aguinelo/homebrew-dcode` **não existe** — 404. O `publish-tap.sh` sai
  com zero e avisa quando não alcança o tap, por desenho, então o v0.0.1 reportou
  sucesso com esse canal nunca tendo sido criado. O README documenta os três
  canais que funcionam, e o tap é decisão a tomar, não linha a escrever.

### Distribuição

- **O release fixa o instalador que publica.** O pipeline passa a verificar a
  assinatura que acabou de produzir, **então** preencher o bloco `PINNED` a
  partir daquele arquivo de checksums, **então** publicar — com o instalador
  fixado entre os artefatos —, **então** levá-lo para a `main`.

  Cada relação de ordem pesa. Fixar antes de verificar tiraria digests que
  ninguém atestou, que é a falha que a funcionalidade existe para impedir,
  reproduzida dentro do pipeline que a implementa. Publicar antes de fixar
  anexaria o instalador sem pino. Escrever na `main` antes de publicar deixaria
  uma falha ali reprovar um release que já deu certo — por isso esse passo, como
  o do tap, sai com zero e avisa alto em toda condição recuperável, com uma
  exceção: arquivo fixado ausente **reprova**, porque deixar a `main` com os
  digests do release anterior faria todo install cair no `checksums.txt` **em
  silêncio**, que é o comportamento correto de um instalador sem pino e
  portanto invisível.

  A `main` importa, e o asset sozinho não: a URL que o README publica é
  `main/install.sh`, e um pino que não chega lá não chega a ninguém.

- **O `scripts/version.sh` ignora o commit de pino do próprio pipeline.** Esse
  commit cai depois da tag, já que os digests só existem depois dos artefatos
  construídos. Contá-lo faria toda consulta pós-release responder "há commits
  desde a tag" com nada humano tendo mudado, e a derivação subiria PATCH sozinha
  — automação deixando rastro que outro mecanismo lê como sinal, forma que este
  repositório não para de encontrar.

  A isenção é do assunto exato, nunca do prefixo: isentar `chore(release):`
  inteiro daria a qualquer pessoa uma forma de não ser contada. Testado dos dois
  lados. O `scripts/version.sh` não tinha teste nenhum até aqui.

- **O instalador confere contra um digest que ele carrega.** O `install.sh`
  ganhou um bloco `PINNED` que o `scripts/installer.sh` preenche a partir do
  `checksums.txt` **já assinado**. Quando o artefato baixado tem digest fixado, é
  contra ele que se compara — e divergência aborta mesmo que o `checksums.txt`
  do próprio release concorde com o download.

  É a metade estrutural da entrada abaixo. O `checksums.txt` viaja do mesmo host
  que o tarball, então sozinho ele pega download corrompido e não pega release
  substituído: quem troca um troca o outro, e o par continua coerente consigo
  mesmo. Tornar a assinatura opcional foi certo, mas opcional não pode virar
  decorativa — e o que impede isso é o valor esperado passar a viver no
  **histórico do git**, onde um asset de release pode ser trocado sem rastro
  público e uma linha versionada não pode.

  O bloco nasce vazio, e o vazio é silencioso: instalador sem pino cai no
  `checksums.txt` sem reclamar, porque avisar sobre um pino que ninguém aplicou
  ensina a ignorar a linha que importa. Fixado num release e chamado para outro,
  ele diz qual carrega e aponta o instalador que carrega os certos.

  Tirado do `uv`, o único de seis instaladores examinados (rustup, bun, deno,
  nvm, k3s, uv) mais rigoroso que este — quatro deles não verificam nada, e
  **nenhum dos seis exige ferramenta externa de verificação.** A separação do uv
  é melhor que a nossa: o instalador dele vem de um host e os artefatos de
  outro. Sem domínio próprio, histórico do git contra asset de release é a melhor
  disponível, e dizer isso faz parte de tê-la.

  O pipeline que preenche o bloco é a mudança seguinte; esta é o mecanismo, com
  o gerador e seis testes.

- **A falta do cosign não cancela mais o checksum.** O `install.sh` exigia
  cosign, e punha a exigência imediatamente antes da verificação de assinatura,
  com a comparação de SHA-256 depois dela. Numa máquina sem cosign — toda máquina
  comum — a instalação abortava sem ter conferido nada: nem binário **nem**
  verificação, o pior resultado disponível, produzido pela regra escrita para
  evitá-lo. Achado pela primeira pessoa a rodar o comando documentado.

  As duas verificações são independentes e passam a ser tratadas assim: o
  checksum roda sempre, a assinatura roda quando há cosign, e a ausência dela é
  dita alto na hora de pular e de novo na última linha. A linha que se sustenta
  nunca foi "verificado ou nada" — é **nunca não-verificado em silêncio**.
  Assinatura presente que não confere continua abortando, e isso está asserido ao
  lado da degradação para que as duas não se separem.

  Ele também para de baixar o `.sig` e o `.pem` que não consegue ler, que era
  mais um jeito de falhar por motivo que não é do usuário.

  Todo teste existente colocava o cosign no PATH, então a única configuração em
  que todo usuário está era a única nunca exercitada. Quatro testes de reprodução
  passam a cobri-la, os quatro vermelhos primeiro pelo sintoma relatado.

## 0.0.1 — 20 de agosto de 2026

O primeiro release com tag. Ele **não** abre superfície estável: `0.x` diz que a
forma ainda se mexe, e este é o ponto a partir do qual as mudanças passam a ser
contadas, não o ponto em que elas param.

As entradas abaixo são o trabalho dos dias que levaram até aqui. Tudo antes disso
vive nos changelogs por família, onde foi escrito quando a decisão foi tomada.


### Instrumento de medição

- **Medição contra o harness consertado.** Os três contratos, 50 execuções cada,
  92 minutos:

  | contrato | antes | depois |
  |---|---|---|
  | `keeps-writing-that-must-cohere` | 96,0% | **100,0%** |
  | `names-the-child-that-did-not-answer` | 98,0% | **100,0%** |
  | `delegates-writing-when-disjoint` | 50,0% | **52,0%** |

  **A previsão do autor estava errada.** O #216 foi escrito afirmando que a
  recusa do harness convencia o modelo a parar de delegar, e que consertar isso
  faria o terceiro número subir. Com n=50 o desvio é ~7 pontos: **dois pontos é
  ruído.** A causa das não-delegações é outra e ainda não é conhecida.

  O conserto não foi inútil, e o ganho está nos outros dois: agora o modelo tem
  uma opção de delegação que **funciona**, e ainda assim recusa dividir trabalho
  que precisa concordar consigo. Antes ele recusava num mundo onde delegar era
  impossível, o que media muito menos.
- **O harness de eval roda um turno filho** (#216). A recusa antiga era honesta
  mas dizia "do the reading yourself", e isso instruía o abandono. Continua sendo
  o comportamento certo a consertar; o que não se sustentou foi a previsão sobre
  o efeito dele.
- **Três contratos para trabalho dividido** (#214, #215). O limiar do terceiro
  desceu de 80% para 25% depois de quatro medições com dispersão de 25 pontos —
  piso contra regressão, não certificado de qualidade.
- **O release alcança um espelho que responde** (#218). O pipeline de release era
  uma cópia da CI que parou de ser atualizada: sem prazo no `apt`, sem
  `apparmor_restrict_unprivileged_userns`, sem sonda — então **todo teste de
  fronteira pulava calado** no pipeline que decide se publica.

### Coordenar máquinas

- **Comando que sai da máquina pergunta** (#212). `ssh`, `scp`, `rsync` para host,
  `kubectl exec`, `ansible`, `aws ssm`, `docker -H`. `git push` não pergunta.
- **Recurso de fora concedido por nome** (#211). `DCODE_SANDBOX_SOCKETS` e
  `DCODE_SANDBOX_WRITABLE`; o literal `ssh-agent` vale por `$SSH_AUTH_SOCK`.
- **Cofre de credencial fora de alcance** (#210). `DCODE_SANDBOX_UNREADABLE`, com
  default que esconde sem ninguém pedir.

### Sandbox

- **Socket é alcançável onde já se escreve** (#199). Conserta a regressão do #196,
  que fechou o `bind` de porta e derrubou metade da suíte.
- **Rede concedida não é socket privilegiado** (#196). O dcode encontrou a
  própria fuga: rodou `docker run` de dentro de `workspace-write`, e funcionou.
- **Sandbox aninhado é detectado, não adivinhado** (#189).
- **Toolchain alcança o próprio cache** (#188).

### Delegação que escreve

- **Escrita recusada diz que era escrita** (#206).
- **O filho diz o que escreveu** (#205). `Wrote` no relatório, e o desfazimento
  do turno do pai alcança o que o filho fez.
- **Filho delegado escreve só o que possui** (#204). `owns` é pedido que só
  estreita, e a contenção responde por ele.
- **Pesquisa e planejamento** (#201, #202).

### Laço e configuração

- **O backstop acompanha o horizonte do modelo** (#195). Teto de 200 para 2.000 —
  a citação que o justificava falava de 1.959 chamadas.
- **As instruções do projeto descrevem este projeto** (#194). 76% do prompt
  descrevia um projeto Node; caiu de 16.904 para 8.757 bytes.
- **A ferramenta descreve o que sabe fazer** (#207, #208). A descrição negava a
  escrita que o schema oferecia, e o modelo não delegava por causa disso.

### CI e cobertura

- **A CI nomeia um espelho que responde** (#203). O passo do `apt` saiu de 6
  minutos de timeout para 13 segundos.
- **Cobertura afrouxa para o piso que as specs pedem** (#192). Agregado em 90%, e
  o piso por pacote passa a reprovar em vez de só imprimir.
- **O gate de cobertura lê a matriz inteira** (#190).
