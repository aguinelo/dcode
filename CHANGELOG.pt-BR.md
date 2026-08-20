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
| famílias de spec | 13, com 63 changelogs de decisão |
| contratos comportamentais | 42 declarados |
| **contratos medidos contra modelo** | **3** |
| cobertura | 95,0%, com gate em 90% |
| CI | matriz macOS + Linux, gate sobre a **união** dos perfis |
| versão publicada | **0.0.1** |

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

### Distribuição

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
