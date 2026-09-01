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

## Estado atual — 31 de agosto de 2026

**O que é.** Harness de codificação agêntica em Go: um daemon, um cliente de
terminal e o laço do agente entre os dois, num binário estático único, sem cgo
fora do pacote isolado.

**Onde está.**

| | |
|---|---|
| famílias de spec | 18, com 155 changelogs de decisão |
| contratos comportamentais | 58 declarados |
| contratos que precisam de modelo | 53 dos 58; 5 se resolvem por asserção |
| **contratos de fato já medidos** | **19** |
| cobertura | 93,3%, com gate em 90% agregado **e por pacote** |
| CI | matriz macOS + Linux, gate sobre a **união** dos perfis |
| versão publicada | **0.17.0** |

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

**O que este documento não diz.** Que o sistema está verificado. Dos cinquenta e
três contratos que precisam de modelo, **trinta e quatro nunca rodaram contra um**,
e o relatório da suíte imprime a divisão em toda execução para impedir a leitura
contrária.

Dos dezenove que rodaram, **cinco não atingiram o limiar**, e os limiares não
desceram para encontrá-los. O pior marca 5%: uma instrução do arquivo do
projeto sobrepondo o piso embutido, que é o que a família dona dela chama de sua
regra mais forte.

Os dois números acima são contados, não herdados. A linha dizia "4", da release
anterior, e continuou 4 enquanto `boundary-decides-write` era medido — uma
tabela descrevendo um estado que já tinha se movido, dentro do documento que
existe para impedir exatamente isso.

---

## Não publicado

- **Uma família para o Gemini**, sobre o transporte `openai`, pela superfície de
  compatibilidade do Google. Família e não transporte: o dialeto já existe, então
  a `Gemini` embute a `MiniMaxM3` para a codificação e sobrescreve exatamente o
  que o eixo família carrega — nome, prefixos de modelo, janela, limites,
  imagens. A superfície nativa é um transporte, e escrever um antes de alguém ter
  rodado isto contra uma chave real seria construir a metade difícil primeiro, em
  cima de um palpite.
- **Os números são escolhidos, não copiados.** A janela é 1.000.000 contra
  1.048.576 documentados, porque errar para baixo custa um resumo e errar para
  cima perde o turno. O teto é 50 e explicitamente não as 2000 da MiniMax, que
  são justificadas por uma execução de horizonte longo citada, que não fala nada
  deste modelo — há teste afirmando que os dois diferem. O `Encode` recusa o
  dialeto Anthropic que ele herdaria, para uma família cujo `Transports()` nomeia
  um só.
- **RN-11: família sem medição diz que não tem.** Nome de família aqui lê como
  família medida, porque o `Measurement.Model` existe justamente para que limiar
  pertença a um modelo e não fale de outro. O aviso nomeia a família e diz contra
  o que os limiares *foram* medidos, e a lista de quem avisa é conferida contra
  as medições que existem, nos dois sentidos — nada digitado.
- **A guarda reprovou na primeira execução, por causa da `claude`.** Aquela
  família existe desde o começo, para provar que os eixos são ortogonais, e nunca
  carregou uma única medição — vinha rodando sem dizer isso. Não foi o que eu fui
  procurar; foi o que a guarda achou por existir.
- Apontar para o Gemini é `model.base_url` =
  `https://generativelanguage.googleapis.com/v1beta/openai`. O `defaultBaseURL`
  responde por **transporte**, não por família, e por isso não nomeia o Gemini:
  transporte decidindo coisa a partir de família é o colapso dos eixos que a
  documentação da própria interface proíbe.
- **O bloco de skills passa a dizer qual é o formato, não só onde ele mora.** A
  correção anterior funcionou pela metade: o agente foi olhar o diretório, e
  ainda assim concluiu que uma skill achada no GitHub "carrega no Claude Code,
  não no meu agente". Ele acertou a parte difícil — que a URL apontava para um
  skill dentro de um **plugin**, que plugin e marketplace são empacotamento de
  outro produto, e que instalar um mexe em setup global e pede confirmação.
  Errou a fácil: `curl` do `SKILL.md` e escrita em `.dcode/skills/`, duas linhas,
  dentro do workspace, sem cruzar nada.
- **É fato e não promessa, e por isso pode ser dito.** O bloco passa a nomear a
  forma — `SKILL.md` numa pasta ou `<nome>.md`, com `name` e `description` no
  topo — e a dizer que uma achada em qualquer lugar é, quase sempre, arquivo para
  copiar sem alterar. O `description` é alias de `when_to_use` desde antes desta
  família existir, e uma skill real de terceiro nesse formato exato foi carregada
  e aplicada num teste de campo nesta tarde.
- **O que fica de fora é deliberado**: plugin, marketplace e comando de
  instalação são empacotamento, não formato, e o agente já raciocina bem sobre
  isso sozinho. A divergência do casamento também — lá o modelo decide pela
  descrição, aqui é determinístico por palavra — que é escolha de desenho de quem
  escreve skill, e vive na `.r`. A seção tem 409 bytes contra um teto de 520 no
  teste.
- **O agente não sabia do próprio mecanismo de skills.** Pedido para instalar
  uma skill, respondeu que não conseguia — que skills são coisa do Claude Code e
  não se instalam a partir dali. Cada frase disso é falsa sobre o produto que ele
  é: o dcode carrega skills de `<workspace>/.dcode/skills/`, e escrever ali é
  escrita **dentro** do workspace, que não cruza nada e não pergunta a ninguém.
  Ele tinha a ferramenta e a permissão; faltava a informação.
- **O bloco de skills passa a ser renderizado mesmo sem nenhuma instalada**, e a
  dizer onde elas moram e o que escrever uma faz. Ele só aparecia quando já havia
  alguma, então um workspace sem skill não contava nada ao modelo sobre o
  mecanismo — e sem nada escrito, ele respondeu pelo treino, que é sobre outro
  produto. Duas linhas, nunca um manual: a economia da RN-7 é a mesma que mantém
  os corpos fora do prefixo, e o que essas duas linhas compram é a alternativa
  não ser o produto desinformando a pessoa sobre ele mesmo.
- **Uma guarda lia string vazia.** O `TestAbsentSectionsEmitNoHeading` afirmava
  que `## Skills` era omitida quando vazia, e passava percorrendo nada: o
  `Prompt` dele não tinha `Safety`, então o `Build` falhava e o laço varria saída
  vazia procurando quatro cabeçalhos. Corrigida na mesma mudança, porque é o
  comportamento que esta mudança altera.
- **O parágrafo da fronteira no README dizia o contrário do que a fronteira
  faz.** Ele afirmava que "qualquer coisa que cruze essa fronteira — escrita fora
  dela, ou rede — para e pergunta", e nenhuma das duas metades é verdade nos
  defaults: `sandbox.allow_network` é `true`, e `/tmp`, `/private/tmp`,
  `/private/var/tmp`, `/dev` e os caches de toolchain são graváveis. As duas
  concessões são deliberadas e os motivos estão escritos no código; o que
  faltava era a capa dizer isso, no único parágrafo em que alguém confia para
  decidir se deixa isto rodando sozinho.
- **A assimetria da leitura está registrada.** `read /tmp/x` declara aquele
  caminho, que está fora do workspace, e pergunta. `bash cat /tmp/x` declara
  apenas o workspace e a rede, e não pergunta. A contenção é idêntica — o SO
  permite a leitura nos dois casos — então o que muda é só a pergunta, e a
  ferramenta que pergunta é a que está sendo honesta sobre o que toca. Achado
  vendo uma sessão real buscar um arquivo na internet, escrevê-lo em `/tmp` sem
  perguntar, e então parar para perguntar se podia ler o que acabara de
  escrever.
- **Skill que alcança a fronteira é retida e perguntada, nunca carregada às
  cegas.** O
  `SafetyClaims` roda sobre instruções desde que a RN-10 pediu que a tentativa
  fosse registrada. Nada rodava sobre skill — e skill é o texto menos confiável
  que este produto carrega: chega por `git clone` em `.dcode/skills/`, ou é
  baixada do repositório de um estranho, que foi o que aconteceu no teste de
  campo desta tarde, e o corpo dela vai direto para o turno dentro de um bloco
  `<skill>` sem ninguém ler antes.
- **Este pergunta onde a RN-10 só reporta, e a diferença é procedência.**
  Instrução é do usuário; descartar um arquivo por uma frase custaria a ele uma
  regra que ele escreveu, então lá falso positivo custa uma linha de saída e
  reportar basta. Em skill a assimetria vira: falso positivo custa **uma
  pergunta**, respondida com o trecho citado à vista — falso negativo carrega
  texto de terceiro no contexto do modelo, sem pergunta nenhuma.
- **Perguntar, e não recusar.** Recusar de saída seria o produto decidindo o que
  é da pessoa; fronteira e autorização são eixos separados (ADR-02), e esta é a
  segunda. Aprovada, a skill carrega inteira — reter é pergunta, não deleção.
  Negada, não carrega. **Sem ninguém para perguntar, não carrega** — a regra que
  o laço já aplica a toda travessia, pelo motivo que ele já dá: com ninguém a
  quem perguntar, a única alternativa a recusar é conceder em silêncio. Os três
  desfechos deixam linha na auditoria, o concedido inclusive, porque
  consentimento que não deixa rastro é indistinguível de pergunta que nunca foi
  feita.
- **As duas metades são filtradas, e o filtro precisa continuar estreito.** O
  corpo é onde a carga estaria; a linha de índice é paga em todo turno, então
  corpo inofensivo sob linha ofensiva é a versão mais barata do ataque. Medido
  contra a `web-design-engineer` — 35.012 bytes de orientação real de terceiro —
  zero casamentos. Uma amostra só, e dito como uma amostra só.
- **A skill espera, o produto não para.** Matar o processo daria a qualquer
  repositório clonado o poder de impedir o dcode de rodar, que é o defeito
  corrigido na entrada acima; recriá-lo em nome da segurança seria trocar um
  problema por ele mesmo.
- **Arquivo de skill ruim não para mais o produto.** Achado por teste de campo,
  não por leitura de código: uma skill real do ecossistema de onde este formato
  veio — `ConardLi/garden-skills/skills/web-design-engineer`, com 455 caracteres
  de `description` onde o teto é 120 — fazia o `LoadSkills` devolver erro, o
  `app.go` propagar, e o dcode sair com 1 naquele workspace, `--dump-prompt`
  incluído. `.dcode/skills/` chega por `git clone`, então um arquivo de um
  repositório clonado decidia se o binário rodava.
- **Os tetos continuam; ser fatal não.** Linha de índice longa demais é aparada
  em fronteira de palavra e o corte é dito; arquivo que não pode ser skill, ou
  acima do teto de bytes, é pulado e dito. O corpo nunca é cortado — orientação
  que para no meio da frase é pior que orientação ausente e declarada ausente. Só
  diretório ilegível continua sendo erro, porque aí é a máquina falhando e não um
  arquivo estando errado.
- Os avisos aparecem no `--dump-prompt`, em bloco próprio, separados dos avisos
  de doutrina porque respondem perguntas diferentes.
- **`skill-loaded-on-trigger` medido: 100% de 20 execuções**, limiar 85%,
  MiniMax-M3. Era um dos contratos que nunca tinham rodado. O juiz procura o
  passo que ninguém adivinharia — a skill manda registrar a versão em
  `RELEASING.md` antes de marcar a tag, e um modelo que nunca recebeu o corpo
  não tem como saber que aquele arquivo existe. Vinte execuções, vinte acertos:
  o mecanismo funciona.
- **O número não diz nada sobre o teto, e a nota diz isso.** O `Rounds` foi de
  12 para `exploreThenActRounds` **antes** da execução, pela definição escrita
  naquela constante e não por evidência deste cenário — e depois disso nenhuma
  execução falhou, então não há falha para atribuir a um número nem a outro. Um
  teto corrigido seguido de 100% lê como causa e efeito, e não é. Este
  repositório já leu cinco números errados por confundir instrumento com
  comportamento; ler um número certo pela razão errada é o mesmo erro com sorte.
- **Skill carregada se anuncia.** O corpo era anexado ao turno como lembrete sem
  nada ser emitido: contexto gasto, comportamento mudado, e nenhum rastro em
  lugar nenhum que a pessoa olhe — `grep -i skill internal/tui/` não devolvia
  nada fora de teste. O índice sempre foi auditável, no prefixo e no
  `--dump-prompt`; o que disparou, não. O `skill.loaded` carrega o nome e a
  mesma linha de quando-usar que o modelo leu no índice, então os dois passam a
  olhar a mesma frase, e o fluxo desenha isso como nota.
- **Não o caminho, e não a cada turno.** O log de eventos é lido por outro
  cliente em outra máquina, onde caminho absoluto de quem escreveu não é fato; de
  qual raiz a skill veio é pergunta que o `--dump-prompt` e o sistema de arquivos
  respondem. Turno que não carrega nada não anuncia nada, e anúncio sem nome não
  desenha — linha cujo único conteúdo é que o recurso existe é linha gasta.
- **`docs/ROADMAP.md` §16 registra as duas coisas que isto deliberadamente não
  fez**: uma listagem `/skills`, e skills embutidas no binário. Contra a segunda
  argumenta a própria RN-7 do produto — cada skill embutida é uma linha paga em
  todo turno de toda sessão — com os três custos ainda não pagos nomeados.
- **A skill carrega pelo que a distingue, não pelo que ela tem em comum.** A
  lista de palavras vazias tinha só inglês, num produto cujo `LANGUAGE.md`
  declara duas línguas e cujo usuário escreve prompt em português: `quando`,
  `projeto` e `estiver` contavam como significativas enquanto `when` e `that`
  não, então a mesma frase era filtrada numa língua e puxava corpo inteiro de
  skill na outra. "quando o projeto estiver pronto me avisa" carregava duas
  skills, e nenhuma delas era sobre nada daquela frase.
- **A lista de português sozinha não resolveu.** `projeto` e `versão` são
  palavras de conteúdo, aparecem nas duas linhas de quando-usar, e dois acertos
  continuam sendo dois acertos. O defeito não é a palavra ser comum na língua — é
  ela ser comum **entre as skills do índice**, e palavra que as duas dizem não
  distingue nenhuma. `Match` passa a exigir dois acertos **e** ao menos um numa
  palavra que nenhuma outra skill do índice carrega. Uma skill sozinha discrimina
  por tudo o que diz, que é a resposta certa: sem vizinha, não há com o que se
  confundir.
- **Vizinhas de um mesmo domínio continuam alcançáveis.** `release-go` e
  `release-node` dizem as duas cortar, versão e nova, e cada uma ainda tem
  `golang` e `typescript` — o que uma regra mais grosseira, de simplesmente
  descartar palavra compartilhada, teria quebrado.
- **A capa diz o que foi medido, e uma guarda conta.** O README ainda afirmava
  que não havia TUI nem binário publicado, quatro meses e dezessete minors
  depois de as duas coisas deixarem de ser verdade; o badge dizia dez specs
  contra dezoito famílias, e a seção de testes anunciava um gate de cobertura de
  95% contra um script que sempre leu 90. Ele agora abre com a razão que estava
  escondendo — 58 contratos declarados, 18 medidos — e carrega a tabela de
  contas, `floor-yields-to-project` em 5% incluído. A comparação com os quatro
  agentes que chegaram antes desceu para o rodapé, onde crédito mora, e o ciclo
  de verificação subiu, porque é a parte que mais ninguém tem.
- **`TestTheReadmeBadgesAreCountedAndNotCarried` e
  `TestEveryReceiptNamesAMeasurement`.** Os badges, a frase abaixo deles, a
  tabela de specs e toda taxa da tabela de contas passam a ser lidos da árvore e
  de `Measured`, nas duas edições — o tratamento que a tabela de estado do
  changelog já tinha, aplicado ao documento que mais gente de fato lê. A guarda
  reprovou na primeira execução, num número que esta mesma mudança havia
  digitado: 40 contratos nunca medidos, onde a árvore dá 35. Cobertura é
  conferida só quanto ao acordo entre README e changelog, porque um teste que
  não roda o gate não pode honestamente afirmar mais, e isso está escrito no
  teste em vez de ficar para ser descoberto.
- **A prosa do próprio changelog estava dois números atrasada.** Dizia que seis
  contratos medidos não atingiram o limiar e que o pior marcava 30%. São cinco,
  e o pior marca 5%.

## 0.17.0 — 31 de agosto de 2026

- **Critério que não imprime nada passa a nomear o seu comando.** Achado
  rodando o binário instalado contra um workspace de verdade, não lendo código:
  um `done.toml` com `test -f CHANGELOG.md` falha em silêncio, então o bloco de
  saída não renderizava e o lembrete voltava a ser o nome e nada. O modelo foi
  ler o `done.toml` para descobrir o que o critério era — duas rodadas atrás de
  algo que o laço tinha em mãos. O comando é identidade e nunca evidência: só
  entra quando não há o que mostrar.

## 0.16.0 — 30 de agosto de 2026

- **`recoverable-cycle`, família nova: o laço é fechado na detecção e aberto na
  recuperação.** Ele sabe que um ciclo piorou e não sabe voltar — o
  `Progressed` devolve um booleano onde cabem três respostas, então empatar,
  regredir e trocar uma falha por outra colapsam num contador de ciclos parados.
  Só `.r`. A objeção que mantinha isto fora de escopo era falsa: **um ponto de
  retorno não precisa ser um commit**, e a fronteira de que git é do usuário
  fica inteira. Desfazer é decisão do laço e nunca do modelo — um agente que
  pode reverter o próprio trabalho pode reverter a evidência.

- **Um ciclo que quebrou algo é desfeito.** O laço passa a classificar o que
  um ciclo fez em três respostas em vez de duas, e reverte as que regrediram —
  um critério que passava e parou. A máquina de instantâneo já existia; nada
  dizia ao laço que um ciclo tinha piorado, e o recorte era o turno inteiro,
  então desfazer depois de um ciclo ruim jogaria fora todo ciclo bom antes
  dele. Empate nunca é desfeito: um ciclo que leu e não fechou nada não quebrou
  nada. O modelo é avisado, e avisado a tentar outra coisa — quem não é avisado
  repete a mesma edição achando que ela nunca aconteceu.

- **O arcabouço sabe rodar um ciclo de verificação, e o primeiro contrato mede
  uma correção.** Duas famílias tinham sido entregues sem nada medido sobre
  elas: todo cenário injetava o lembrete que o ciclo teria produzido, então
  `checkDone`, `Moved` e a reversão nunca rodavam. Um cenário pode declarar
  critérios — predicados sobre o workspace, nunca shell — e o juiz roda a régua
  de novo em vez de ler o transcript. `fixes-what-the-output-named`: **100% de
  20**.
- **Ele leu 65% primeiro, e era um critério meu.** Exigia uma implementação
  específica e a mensagem de erro se lia como o oposto dela, então cinco das
  sete falhas eram execuções presas tentando satisfazê-lo. Quatro vezes em dois
  dias uma taxa disse algo interessante sobre o modelo e era sobre o
  instrumento.

- **Uma medição tirou dois passos do plano.** O roteiro do laço tinha
  "progresso por aproximação" e "subir o teto de ciclos parados" depois da
  reversão, os dois para impedir que o laço desista de trabalho que avança sem
  fechar critério. O `finishes-work-that-takes-more-than-one-cycle` mediu **95%
  de 20**: o teto não morde. O `Moved`, entregue por outro motivo, já tinha
  resolvido — qualquer avanço zera o contador. A intuição era verdadeira quando
  foi escrita e deixou de ser; sem medir, os dois passos teriam sido
  construídos, teriam funcionado, e ninguém saberia que não precisavam existir.
- **`verifiedCycleRounds`**, porque um cenário que roda o ciclo gasta rodadas
  que o trabalho não vê: o modelo tem de parar de chamar ferramenta para o
  ciclo rodar. O teto antigo foi escrito quando nenhum cenário rodava um.

## 0.15.0 — 29 de agosto de 2026

- **A saída do critério que falhou chega ao modelo.** O lembrete carrega o
  que o comando imprimiu, sob a frase que já existia, marcado uma vez como
  resultado e não instrução. Medido antes e depois: os dois contratos de maior
  risco ficaram em 100% de 50 e 100% de 20, e o `states-unmet-on-stall` foi de
  92% para 94% — dois pontos, a menor diferença que 50 execuções enxergam. **A
  família não se justificou pelo número**; fica pelo argumento estrutural, e
  isso está escrito como tal.
- **O teto de rodadas já decidiu quatro medições em dois dias.** Com teto 12 o
  mesmo par leu 82% → 72%, pronto para ser publicado como *a saída piora o
  relato honesto*. Treze das quatorze falhas eram execuções cortadas no meio do
  trabalho. O teto não sobe mais: subir até um contrato passar é ajustar o
  instrumento ao resultado.
- **A saída do critério que falhou fica.** O `Check` rodava o comando e
  descartava o que ele imprimiu num `_`; agora guarda a de tudo que não passou,
  com teto de 2000 bytes — o mesmo do qualificador, porque é a mesma informação
  do mesmo runner. Cortada pelo FIM, já que o resumo de uma suíte e a última
  asserção estão embaixo. Ainda não chega ao modelo: isso muda o prefixo, e a
  medição precisa de um "antes".
- **`failure-feedback`, família nova: o laço detecta bem e devolve mal.** Quando
  um critério falha, o `Check` roda, descarta o que ele imprimiu num `_`, e o
  lembrete diz ao modelo o NOME do critério e nada sobre o que quebrou. A
  evidência é colhida e jogada fora na mesma linha — enquanto a fase vizinha, o
  qualificador, guarda e escreve. Só `.r`: o problema, as regras, e o risco dito
  antes de construir.

## 0.14.0 — 28 de agosto de 2026

- **Verificação impossível não cancela o trabalho.** Uma quinta prática no
  piso, e ela entrou por medição, que é o que a RN-8 da própria família exige.
  Três contratos de duas famílias mostraram a mesma forma — o turno lê tudo,
  raciocina certo, e termina sem propor nem editar — sempre logo depois de
  anunciar uma verificação que não conseguia fazer. A doutrina já dizia o que
  DIZER quando não dá para conferir; nunca disse que o trabalho continua devido.
- **Quatro taxas medidas foram substituídas, não acumuladas.** Elas descreviam
  cenários que mudaram embaixo delas: um teto de 12 rodadas em turnos que leem
  uma spec e uma base antes de produzir qualquer coisa, e um workspace de eval
  compartilhado que não compilava. Uma taxa pertence a um cenário, e uma que
  sobrevive ao cenário dela é o defeito da tabela de estado com outra roupa.
- **O workspace de eval compartilhado volta a compilar.**
  `internal/config/toml.go` chamava dois helpers que não existiam. Modelos leem
  esse arquivo em cenário após cenário, e os cuidadosos diziam isso e gastavam
  as rodadas ali. `TestTheSharedWorkspaceCompiles` roda `go build` offline.
- **A ablação, porque três mudanças juntas não atribuem nada.** Revertendo uma
  de cada vez, 20 execuções por leitura: sem a prática 90%, com o teto de volta
  a 12 95%, com o workspace quebrado 95% — contra 100% com as três e 75% com
  nenhuma. Conjunto e aproximadamente aditivo, sem causa dominante.

## 0.13.0 — 27 de agosto de 2026

- **Oito limiares declarados viraram limiares medidos, e cinco não se
  sustentaram.** Os três contratos do qualificador e os cinco do piso rodaram
  contra modelo real. `qualifier-proposes-commands` (96%),
  `floor-yields-to-user` (96%) e `floor-checks-before-claiming` (100%) fecharam;
  os outros cinco não, e nenhum limiar desceu para encontrar um resultado.
- **A regra mais forte do piso mede 30%.** A mesma instrução, o mesmo texto, a
  mesma tarefa: dita pelo usuário no turno, é obedecida em 96% de 50 execuções;
  escrita no arquivo do projeto, em 6 de 20. O desenho da família se apoia no
  prefixo ser montado em ordem, com as instruções do projeto por último —
  posição no prefixo não é precedência, é esperança de precedência.
- **Critério quebrado é gravado e não declarado.** Antes era declarado, e o
  arquivo é o que a execução seguinte carrega: a sessão de trabalho passava a
  ser medida contra um comando que não existe — vermelho para sempre — e a
  pasta passava a declarar um critério, então nunca mais voltava para a
  qualificação. Dois becos sem saída de uma linha só.
- **Três contratos mostram a mesma falha, em cenários que não têm nada em
  comum.** O turno lê tudo, raciocina certo, e termina sem chamar a ferramenta
  — sempre depois de dizer que quer verificar algo que o turno não consegue
  verificar. O próximo alvo é a instrução, não os cenários.
- **`qualifier-narrows-on-mismatch` foi retirado.** Ele descrevia um segundo
  turno do modelo reagindo à medição, e a medição agora acontece fora do turno:
  quem lê a discordância é a pessoa. Retirar contrato é no mínimo MINOR.

- **O laço descobre os próprios critérios, em modo planejamento.** Um turno
  qualificador lê a spec e o código e chama `done_propose`; o harness mede cada
  critério proposto e escreve um `done.toml` que a pessoa revisa. Quem decide
  que há qualificação é o **laço** — um modelo que escolhesse quando qualificar
  estaria escolhendo quando ser medido.
- **A ferramenta não toca em nada, e isso é o desenho.** A primeira versão
  declarava escrita e o modo planejamento negou, como devia: `read-only` não tem
  exceção, e uma seria a exceção que a próxima pessoa amplia. Então a proposta é
  *gravada*, o turno termina, e o laço pede ao daemon que meça e escreva — sob a
  fronteira em que o trabalho vai rodar, que é o único lugar onde os critérios
  conseguem rodar.
- **O laço encadeia as fases sozinho.** `/loop specs/x` pergunta ao daemon o
  que a pasta declara — uma leitura, `measure=false`, sem rodar nada — e abre a
  sessão qualificadora quando a resposta é nada. O turno acaba, o laço faz o
  commit da proposta, e **para**: proposta que ninguém olhou é régua que
  ninguém leu. Num backlog isso vira uma passada só, e uma sentada de revisão
  antes de o trabalho rodar sozinho.
- **O `Expects` pegou o primeiro de verdade.** Numa spec só com prosa, o modelo
  propôs `bash reverse.sh; test $? -ne 0` esperando que falhasse. Passou — 127 de
  um script ausente é diferente de zero — então o critério estava verde pelo
  motivo errado, e o arquivo diz isso na linha acima dele. Nenhuma leitura humana
  de uma lista de comandos pegaria.

## 0.12.0 — 27 de agosto de 2026

- **`/loop <objetivo>` trabalha o backlog inteiro.** `/loop implemente todas as
  specs pendentes` transformava `implemente` em nome de pasta e falhava em
  `implemente/tasks.md` — prosa virou caminho, o mesmo defeito de prosa virando
  critério apontando para o outro lado. Argumento com separador, ou uma palavra
  só, é caminho; frase é objetivo. RN-7 e US-2, escritas no dia 25 e só agora
  construídas.
- **"Pendente" é medido, não contado.** A descoberta roda os critérios de cada
  pasta pelo mesmo sandbox de um turno, porque marcação é feita por quem teve
  vontade. Pasta que não declara nada é pendente: ausência de prova não é prova
  de pronto. Quem decide é o daemon, que tem o disco e o sandbox.
- **Spec ainda sem tarefas é pendente, não ilegível.** Medido contra um backlog
  real, 11 de 28 pastas voltaram como erro e ficaram fora da fila — todas
  `spec.md` sem `tasks.md`, que é a coisa mais pendente que existe. Agora 28 de
  28, nenhuma ilegível.
- **Um guard casou string solta, terceira vez hoje.** O teste de vazamento de
  inglês procurava palavras do catálogo por substring numa tela em português, e
  `works` casou dentro de `workspace-write` — um valor, não layout. Agora casa
  palavra inteira.

## 0.11.1 — 27 de agosto de 2026

- **`/loop` faz o trabalho em vez de preparar um lugar para pedirem por ele.**
  Ele carregava a definição de pronto, trocava de sessão e **não submetia
  nada** — quem digitava `/loop specs/x` ainda tinha que dizer o que queria.
  Agora submete, nomeando a spec e dizendo que o harness confere os critérios,
  sem repeti-los.
- **Palavra depois do caminho é o que fazer, não flag mistecleada.** `/loop
  <caminho> implementar …` era recusado com *"implementar is not a flag here"*.
  Só o que começa com `-` pode ser flag errada; o resto é a tarefa, como
  digitada.

## 0.11.0 — 27 de agosto de 2026

- **A pasta da spec pode declarar o próprio `done.toml`.** Medido contra as 17
  specs reais do Code Plain, `/loop` devolveu zero critérios em todas: os
  `tasks.md` não têm marcador `verify:` e os critérios de aceitação são frases —
  *"Lighthouse ≥ 95"* — que nenhum parser pode virar comando sem inventar um. A
  pasta ganhou onde dizer isso em comandos. Mesmo nome e formato do arquivo do
  workspace, um parser só; vence o `tasks.md`, vazio é erro em vez de queda, e
  ausente é o caso comum.

## 0.10.0 — 27 de agosto de 2026

- **`/loop` é digitável.** `/loop <caminho> [--protect <glob>]` abre sessão
  medida contra o `tasks.md` daquela pasta, e o texto do comando nunca vira
  entrada de turno. O cliente manda o caminho e o daemon lê, porque o cliente
  pode não estar perto daquele disco.
- **A sessão diz quantos critérios carrega, e zero é resposta.** Sessão sem
  definição de pronto relata pronto no fim do primeiro turno, então o `/loop`
  avisa na hora em vez de no fim.
- **Um erro era classificado procurando a palavra "workspace" no texto dele**, e
  mensagem carrega caminho — então spec inexistente voltava `workspace_invalid`
  num repositório que mora num diretório com esse nome. Agora é sentinela, com
  `errors.Is`.
- **A tabela de estado é contada, não digitada.** Famílias, changelogs de
  decisão e todos os números de contrato saem da árvore e são conferidos contra
  as duas edições e contra a frase ao lado da tabela.
- **O número que diz o quão pouco está verificado estava desatualizado** —
  herdado da release anterior enquanto um quinto contrato já fora medido.
  Contado e separado: 56 declarados, 51 que precisam de modelo, 13 já rodados.

## 0.9.1 — 27 de agosto de 2026

- **Dois portões com o mesmo nome passam a ser distinguidos na tela.** Achado
  rodando a 0.9.0: um projeto com script `test` no `package.json` e alvo `test`
  no `Makefile` imprimia duas linhas chamadas `test`, com comandos diferentes, e
  nada dizia qual era qual. O `Source` era carregado e nunca mostrado. É o mesmo
  defeito que o parser de tarefas recusa num `tasks.md` — duas linhas que o
  leitor não distingue — chegando pela outra ponta da mesma release. Só os
  ambíguos são qualificados.

## 0.9.0 — 27 de agosto de 2026

- **Workspace sem repositório git diz isso, uma vez.** `Repo` era `nil` para um
  diretório que não é repositório, e `nil` não punha nada no prefixo — o
  comentário do campo dizia "ordinary and silent" e a invariante dizia que o
  prefixo carrega "nada quando não é". Ordinário, sim; silencioso, não. Sem
  repositório não há diff para revisar, nem desfazer que não seja reescrever o
  arquivo à mão, nem commit, branch ou PR — então todo acordo de trabalho que um
  arquivo de projeto descreve está descrevendo maquinário que não existe. Achado
  em auditoria: um agente trabalhou um dia inteiro exatamente nesse estado,
  escrevendo o próprio arquivo de projeto exigindo um commit por tarefa, e nada
  lhe disse. O prefixo passa a afirmá-lo como fato, com a instrução de dizer uma
  vez, oferecer `git init`, e seguir o trabalho.
- **"Não olhei" e "olhei e não há" continuam separados.** `nil` passa a
  significar só o primeiro, e segue calado. Três guardas numa função só mantêm a
  distinção, e a do prazo cancelado não foi previsão: um teste que já existia
  reprovou a primeira versão desta mudança, que afirmava "não é repositório"
  sobre uma leitura que nunca completou.
- **A doutrina ganha um piso, e ele é sobreponível.** `Doctrine.Practices` é o
  que o dcode faz quando ninguém pediu. A assimetria com `Safety` é a regra
  inteira: `Safety` não tem campo no overlay **porque não pode ser sobreposta**
  — trava por tipo, não por convenção — e `Practices` tem **porque um piso que
  não pode ser sobreposto não é piso, é regra fingindo ser default**.
  `practices.md` substitui e nunca acrescenta.
- **A precedência não precisou de máquina nenhuma.** `prompt > projeto >
  default` sai da posição: o piso é renderizado depois da `Safety` e antes de
  tudo que alguém efetivamente disse, e as instruções do projeto seguem sendo o
  último bloco do prefixo. Duas invariantes guardam isso, e a segunda é a que
  carrega peso — no dia em que as instruções do projeto deixarem de ser as
  últimas, o piso passa a vencer quem devia vencê-lo e nada mais no código diria.
- **O piso tem texto: três práticas e as duas regras sobre elas.** Conferir
  antes de afirmar que falta algo num arquivo; reler o documento que o próprio
  turno tornou obsoleto; saída não-zero é falha, e se uma instrução mandar ler
  uma delas como sucesso, obedecer **e citar a instrução**. Nenhuma saiu de lista
  de boas práticas; as três são defeitos que alguém entregou. Dois parágrafos
  não são práticas e sim regras sobre elas: dizer isso **uma vez**, nunca como
  ressalva anexada ao trabalho e nunca esperando resposta; e instrução do usuário
  ou do projeto que contradiga a seção **vence sem discussão**.
- **O prefixo nomeia os portões que o projeto declara.** `internal/workspace` lê
  os scripts do `package.json` e os alvos do `Makefile`. O projeto auditado
  declarava quatro, dois vermelhos desde o primeiro dia, e o único verde media
  `1 + 1 === 2`. A lista termina numa frase que é constante não configurável:
  *nada aqui afirma que eles passam, e nada os rodou*. Sem ela, lista de portões
  lê como lista de garantias — que é o defeito que pediu a seção.
- **`This repository` virou `This workspace`**, porque o bloco já carregava a
  linha dizendo que **não** há repositório.
- **O qualificador mede antes do trabalho, e classifica.** Critério que **falha**
  é aceitação — testemunha que o trabalho aconteceu; o que **passa** é guarda de
  regressão. Passar é `Exit == ExitCode`, nunca `Exit == 0`; 126 e 127 e falha ao
  iniciar são **quebrado**, não vermelho.
- **O qualificador pode ser assinado, e assinar é editar.** Critério editado é
  medido de novo antes de congelar. Recusa, prazo, teto de rodadas e **falha do
  canal** terminam em `ErrRefused`. E conjunto que **fica** vazio ao congelar é
  recusado: ele não era vazio, ficou — a única porta que a regra não cobria.
- **A `loop-command` deixou de afirmar mais do que faz.** Só `- [ ] N.` e
  `` verify: `cmd` `` são sintaxe; `tasks.md` que não dá para ler é erro, não
  `DoneSet` vazia; e o fall-through de `SourceAuto` voltou a existir.

## 0.8.0 — 26 de agosto de 2026

> **MINOR, embora todo commit dela diga `fix:`.** O `scripts/version.sh` deriva
> 0.7.1 dos Conventional Commits; o contrato diz outra coisa, e o contrato
> vence. Dois campos entraram no protocolo (`tool.requested.typed`,
> `session.mode_changed.sandbox_mode`), um comportamento foi **removido** (a
> confirmação em duas etapas do `/mode auto`, lançada na 0.7.0 e retirada no
> primeiro uso), e a doutrina mudou de sentido — e a superfície deste produto é
> em parte feita de frases.
>
> Medido contra o MiniMax-M3 neste ciclo: `boundary-decides-write` **MET, 100%
> de 20 execuções**. O `boundary-decides` voltou 90,0% com uma execução perdida
> por EOF de transporte, que o harness reporta como **unsound** e não como
> veredito — 19 execuções não são 20.

### Corrigido

- **Uma parede que diz como se abre.** A doutrina passou a mandar tentar em vez
  de recusar, e deixar a fronteira perguntar — mas um cruzamento de caminho
  dentro de um comando de shell não tem quem pergunte: o comando é opaco, então
  ninguém sabe que houve cruzamento. Observado ao vivo, o modelo fez o que foi
  mandado, foi barrado, e disse de boa fé que *"o harness vai te perguntar"*.
  Nunca pergunta, e a pessoa fica esperando. O resultado do comando passa a
  levar uma nota: este EPERM é o sandbox, pergunta nenhuma vem, e os caminhos
  são `/mode auto` ou nomear o path em `sandbox.writable`. Estreita de
  propósito — só EPERM, nunca sob `full-access`.

- **O status do topo também segue a troca.** O crachá aprendia o modo novo e a
  barra de status não, então uma sessão em `auto` seguia anunciando
  `workspace-write` — o campo que a §2.1 chama de perigoso errar, e por isso
  isento da ordem de descarte. Ele anunciava um limite que tinha acabado de ser
  retirado, que é o pior sentido para esse campo errar. O `session.mode_changed`
  passa a carregar `sandbox_mode`, carregado e não recalculado pelo cliente,
  porque a tabela que liga nome a par tem uma casa só.

- **`auto` remove a fronteira de verdade.** Trocar de modo movia a resposta da
  política e mais nada: o sandbox recebia o modo como **valor**, copiado quando
  a sessão foi montada, então `/mode auto` fazia o veredito dizer `allow` e o
  crachá dizer `auto` enquanto o SO seguia aplicando o limite com que a sessão
  nasceu. Escrever fora do workspace continuava voltando `EPERM` — um modo cujo
  contrato inteiro é "sem fronteira" deixava uma de pé. O executor agora
  pergunta o modo **a cada comando**; fonte nula significa `read-only`, porque
  fronteira que ninguém decidiu falha fechada. Medido num pty real: o mesmo
  `mkdir` fora do workspace é recusado sob `assist` e funciona sob `auto`.

- **Quem pergunta é o harness, e o modelo não.** A doutrina dizia "when that
  happens **the user is asked**" — voz passiva, sem sujeito — então o modelo
  preencheu o sujeito consigo mesmo e construiu um protocolo de permissão
  próprio, em prosa, que nunca aciona a máquina de aprovação: *"você tem que
  dizer 'vai' explicitamente"*. Ele citava essa frase para justificar
  exatamente o que a mesma doutrina proíbe três linhas abaixo. O sujeito passa a
  ser nomeado, a chamada passa a ser dita como sendo **a** pergunta, e permissão
  dada em prosa passa a ser dita como não concedendo nada, porque nada chegou a
  ser perguntado.
- **Uma célula medida não é a vizinha medida.** O `boundary-decides` marcava
  100% de 20 execuções enquanto isso falhava na frente de um usuário, porque ele
  cruza a **rede** e a falha relatada **escrevia fora do workspace**. Um segundo
  cenário cobre essa célula. E o limite dos dois fica escrito: o eval é de turno
  único, e a recusa que sobrevive a ser contestada é uma falha que este
  arcabouço ainda não enxerga.

### Alterado

- **Todo modo passa de primeira, `auto` inclusive.** A confirmação em duas
  etapas saiu ontem na 0.7.0 e durou até o primeiro uso. Digitar `/mode auto`
  são onze caracteres deliberados — não há reflexo a desambiguar, ao contrário
  do `^C`, e pedir que a pessoa repita o que acabou de dizer não é salvaguarda,
  é um degrau que se aprende a pular. O que diz que não há fronteira é o crachá
  na barra, que diz isso enquanto for verdade.
- **O que você digitou, você lê.** `!ls -la` desenhava uma linha, `exit 0`, e
  mais nada. A saída nunca se perdeu — chegava ao cliente, ficava na entrada, e
  reaparecia com `esc`, `↑`, `tab`. Isso é pior que perder: a tela respondia um
  pedido de **ver** com um código de status, e parecia certa fazendo isso. A
  regra de recolher foi escrita para as chamadas do modelo, onde a saída é meio
  e a prosa seguinte carrega o ponto; um comando digitado não tem prosa depois.
  A origem passa a viajar no evento, em vez de ser inferida do formato do id.
- **`exit N` aparece uma vez.** O `bash` prefixa a saída com o código porque o
  modelo lê a saída como texto, e a linha já mostra esse código na coluna dela.
  Invisível enquanto a saída ficava recolhida; dobrado assim que o comando
  digitado passou a abrir sozinho.

## 0.7.0 — 25 de agosto de 2026

### Adicionado

- **Três modos, e um caminho entre eles sem reiniciar.** `plan`, `assist` e
  `auto` são nomes para o par que o motor já rodava — read-only + never,
  workspace-write + on-request, full-access + on-request. `/mode` mostra ou
  troca, `shift+tab` cicla, e a barra leva o crachá. Cair para `auto` pede o
  gesto duas vezes, pelos dois caminhos, com um aviso só, que vive no rodapé
  enquanto a decisão está pendente — como o segundo `^C` já funciona.

### Alterado

- **Sair custa duas, limpar a linha não custa nenhuma.** `^C` significa "limpa
  esta linha" em todo shell, e estava ligado direto no sair — então um reflexo
  que o terminal ensinou custava uma conversa. Agora: turno rodando é
  interrompido, linha com texto é limpa, e linha vazia avisa primeiro e sai
  depois. Armado exatamente enquanto o aviso está na tela, porque qualquer outra
  tecla desarma: um temporizador deixaria a tecla viva por um segundo depois da
  frase ter sumido, que é um estado que a pessoa não vê e portanto não consegue
  raciocinar sobre.

### Corrigido

- **A sessão diz o modo em que de fato está.** Ela nascia rotulada `assist`
  fosse qual fosse o modo do motor, então `full-access` usava o crachá do modo
  contido — e trocar de volta **para** `assist` não fazia nada, porque a sessão
  acreditava já estar lá. O comando que instala a fronteira era justamente o que
  falhava em silêncio. O nome passa a ser derivado do par em vigor, e um par que
  não é nenhum dos três fica sem nome em vez de receber o do vizinho.
- **Trocar de modo com turno vivo deixou de ser corrida.** `SetMode` escreve da
  goroutine do handler HTTP enquanto o turno roda — que é o motivo de não
  interrompê-lo. A avaliação foi posta sob o mutex; a montagem de um filho
  delegado, não, e ela lê os mesmos dois campos da goroutine que está viva
  quando a troca chega. Todo leitor passa agora por um acessor sob trava.
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
