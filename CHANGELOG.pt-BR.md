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

## Estado atual — 20 de agosto de 2026

**O que é.** Harness de codificação agêntica em Go: um daemon, um cliente de
terminal e o laço do agente entre os dois, num binário estático único, sem cgo
fora do pacote isolado.

**Onde está.**

| | |
|---|---|
| famílias de spec | 13, com 68 changelogs de decisão |
| contratos comportamentais | 42 declarados |
| **contratos medidos contra modelo** | **3** |
| cobertura | 95,0%, com gate em 90% |
| CI | matriz macOS + Linux, gate sobre a **união** dos perfis |
| versão publicada | **0.1.0** |

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

### Cliente TUI

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
