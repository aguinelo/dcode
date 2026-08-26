# Research: Piso de prática e precedência

> Fonte da verdade de negócio para **o mínimo que o dcode faz sem que ninguém
> peça**, e para quem tem autoridade de mudá-lo. Depende de:
> **`202608080016-behavior-definition`** (doutrina, `Repo`, `DoctrineOverlay`,
> `Origin`, e a tabela `authority` das fontes de instrução),
> **`202608072337-tool-suite`** (as ferramentas cujo resultado vira afirmação),
> **`202608261730-done-qualifier`** (que mede os portões; aqui eles só são
> inventariados).

## 1. Contexto

O dcode não tem piso. Ele tem doutrina — identidade, política de ferramenta,
segurança, estilo — e tem a cadeia de instruções do usuário e do projeto. Entre
as duas não há nada dizendo **o que ele faz por padrão quando ninguém mandou**.

O resultado disso foi auditado num projeto real em 2026-08-26, depois de um dia
inteiro de trabalho de agente. Dois padrões apareceram, e os dois já haviam
aparecido no próprio repositório do dcode na mesma semana:

**Afirmação não conferida.** Um documento de análise listava cinco conflitos
"🔴 bloqueantes, resolver antes de implementar". Quatro deles já estavam
resolvidos nos arquivos que ele citava — incluindo uma lista de caminhos
reservados que o documento "propunha adicionar" e que estava no arquivo,
palavra por palavra, com comentários citando as specs que o documento dizia
estarem em conflito. Do outro lado, no repositório do dcode, um `CHANGELOG.md`
afirmava que um comando rodava enquanto o `ROADMAP.md`, no mesmo commit, dizia
que não havia código nenhum.

Direções opostas, mecanismo idêntico: **um valor copiado de uma verdade que se
move**. O documento foi escrito uma vez e nunca releu o mundo que descrevia.

**Portão não lido.** O projeto declarava quatro portões. Dois estavam vermelhos
desde o primeiro dia — `lint` quebrado por ferramenta depreciada, cobertura em
0% contra um piso de 80% que o próprio arquivo do projeto chamava de "mínimo
pra merge". O terceiro, `pnpm test`, passava verde medindo `1 + 1 === 2`.
Ninguém rodou nenhum deles, e o verde que existia não media nada.

E por baixo dos dois: **não havia repositório git**, num projeto cujo arquivo de
instruções exigia um commit por tarefa e um PR por spec. O harness olhou, na
abertura da sessão, e não disse — porque estava escrito no código que isso era
"ordinary and silent".

## 2. O que esta família é, e o que ela não é

**É** um piso pequeno: um punhado de coisas que o dcode faz sem que ninguém
peça, cada uma com um default declarado, e uma regra de precedência que diz
quem pode mudá-lo.

**Não é** uma política de qualidade. Não decide se o projeto deve ter testes,
qual cobertura, qual fluxo de branch. Essas decisões são do dono do projeto, e o
arquivo dele manda. O piso é sobre **não afirmar o que não se conferiu** e
**não deixar portão declarado sem leitura** — que valem em qualquer projeto,
com qualquer política, e que é justamente por isso que podem ser default.

## 3. Fronteira de determinismo

**Regime: misto**, e a divisão importa mais aqui do que em qualquer outra
família desta base.

| Parte | Regime | Verificação |
|---|---|---|
| Sondar o workspace (é repositório? que portões declara?) | determinístico | asserção |
| Colocar o resultado da sondagem no prefixo **como fato** | determinístico | asserção |
| Resolver a precedência entre default, projeto e prompt | determinístico | asserção |
| Relatar qual default foi sobreposto e por qual linha | determinístico | asserção |
| Conferir uma afirmação sobre um caminho antes de escrevê-la | **mediado** | limiar |
| Reler um documento antes de reafirmá-lo | **mediado** | limiar |
| Não descontar um exit code sem ordem | **mediado** | limiar |

**A regra de projeto que decorre disto, e é a mais importante da família:
o que pode virar fato não vira prosa.**

Prosa é a camada mais fraca que este repositório reconhece — está escrito no
`AGENTS.md` e foi provado três vezes esta semana. Um piso de prática escrito
como exortação é um piso que vale o que valer a atenção do modelo naquele turno.
O mesmo piso escrito como **fato no prefixo** é lido antes de qualquer decisão,
e o `repo.go` já tinha escrito o porquê:

> "uma regra que precisa de consulta primeiro é uma regra seguida por acidente"

Por isso a tabela acima tem quatro linhas determinísticas antes das três
mediadas, e por isso a §5 separa o catálogo em **fatos** e **práticas**. Toda
prática que alguém conseguir converter em fato deve ser convertida, e o número
de práticas em prosa deve encolher com o tempo, nunca crescer.

## 4. Regras de negócio

### RN-1 — A precedência é absoluta, e quem está acima não discute

```
prompt do usuário (este turno)  >  arquivo do projeto  >  default embutido
```

O de cima **substitui** o de baixo. Não negocia, não pondera, não pede
confirmação, não avisa que "isto contraria a boa prática".

O arquivo do projeto é `DCODE.md`, `AGENTS.md`, `CLAUDE.md` — o que a família
`behavior-definition` já resolve como `SourceProject`. Se ele disser o
contrário do default, o default **cai**, sem uma linha de resistência.

Isto não é tolerância a risco. É a constatação de que o dono do projeto sabe
coisas sobre o projeto que o harness não sabe, e de que um default que
argumenta com quem o sobrepõe é um default que custa mais do que entrega. A
alternativa — um agente que reabre a mesma discussão a cada turno — já foi
medida neste produto e o nome dela é a sessão que não anda.

### RN-2 — Sobrepor é obedecer **e** dizer uma vez. Dizer não é perguntar

Quando um default cai, o dcode diz **uma vez**, em uma linha: qual default
caiu e o que o derrubou.

E aí está a distinção que a RN-1 exige que seja escrita explicitamente:

| é | não é |
|---|---|
| "o default X está desligado pelo `DCODE.md`, linha 87" | "quer mesmo desligar X?" |
| dito uma vez, no começo | repetido a cada turno |
| afirmação | ressalva anexada ao trabalho |
| e o trabalho segue | e o trabalho espera |

Relatar não é questionar. Um agente que transforma o relato em ressalva, em
pedido de confirmação, ou em "vou fazer como você pediu, mas note que…" está
quebrando a RN-1 com a aparência de cumprir a RN-2.

A justificativa é do próprio repositório, escrita para as seções de doutrina em
`doctrine_overlay.go`, e vale igual aqui:

> "uma substituição invisível seria pior que a imutabilidade que ela substitui,
> porque o único jeito que o usuário tem de saber o que chegou ao modelo é ler"

Existe mecanismo pronto para isso: `Origin` e `SectionOrigins`. Falta ele
existir para práticas.

### RN-3 — Fato não se sobrepõe; prática sim

O `DCODE.md` pode desligar a prática de anunciar a ausência de repositório. Ele
**não** pode fazer o workspace ser um repositório.

A distinção é o que mantém a RN-1 absoluta sem transformar a precedência numa
máquina de mentir. O que se sobrepõe é **o que o dcode faz**; o que ele
**observou** não está em disputa, e nenhuma instrução de nenhuma fonte faz uma
sondagem devolver outro resultado.

Consequência prática: um default que é fato ("este workspace não é
repositório") continua verdadeiro depois de sobreposto — o que muda é o dcode
parar de mencioná-lo.

### RN-4 — Afirmação sobre um caminho é conferida antes de ser escrita

"O arquivo X não tem Y", "falta Z", "não está declarado em lugar nenhum" — toda
frase que nomeia um caminho e afirma o que há ou não há nele é conferida contra
aquele caminho, **no mesmo turno em que é escrita**.

Não é rigor acadêmico: é o defeito auditado. Quatro dos cinco "bloqueantes" de
um documento de análise eram afirmações sobre arquivos que qualquer `grep`
teria desmentido, e o documento inteiro pedia para ser aprovado antes de
qualquer implementação.

O custo de conferir é uma leitura. O custo de não conferir é a próxima pessoa
refazendo trabalho feito, com o documento por cima dizendo que era bloqueante.

### RN-5 — Documento que descreve estado é relido antes de ser reafirmado

Um documento que descreve o mundo envelhece no instante em que o mundo se move
— e quem mais move o mundo é o próprio turno que escreveu o documento.

Se o turno editou os arquivos que o documento descreve, o documento é relido e
corrigido **antes de o turno terminar**. Um `CHANGELOG` que diz que o comando
roda, escrito no commit que não construiu o comando, é isto.

### RN-6 — Portão declarado é inventariado; portão vermelho de nascença é dito

O que o projeto declara como portão — scripts de `package.json`, alvos de
`Makefile`, o que o arquivo do projeto chamar de mínimo — é **nomeado no
prefixo**, como fato, do mesmo jeito que o branch é.

**Inventariar não é medir.** Rodar cada portão e classificar o resultado é a
família `202608261730-done-qualifier`, e a regra do vermelho inicial mora lá.
Aqui só se garante que o agente **sabe que os portões existem** sem ter que
descobrir por leitura, e que um portão que já estava vermelho quando a sessão
abriu é dito uma vez em vez de descoberto no fim.

Um portão que ninguém lê não é portão — é decoração, e ensina todo mundo que o
lê a ignorá-lo.

### RN-7 — Exit code não é descontado sem ordem, e a ordem é citada

Vermelho é vermelho. Se o arquivo do projeto manda ler um vermelho específico
como verde — e um deles mandava, com nome de sinal e de syscall —, o dcode
**obedece** (RN-1) e **cita a linha** ao fazê-lo (RN-2).

O que não pode acontecer é o desconto virar hábito: a licença vale para o caso
que a instrução descreve, e não para o próximo vermelho que se pareça com ele.

### RN-8 — O piso é pequeno, e encolhe

Uma prática que ninguém consegue medir é candidata a não existir.

O catálogo da §5 tem o tamanho que tem porque cada linha veio de um defeito
observado, não de uma lista de boas práticas. Acrescentar uma prática exige o
defeito que a motiva; converter uma prática em fato é sempre preferível a
mantê-la; e uma prática que nunca reprovou nenhum contrato em três medições é
uma linha que o modelo já faz sozinho, e sai.

## 5. O catálogo

### 5.1 Fatos — sondados, postos no prefixo, não sobreponíveis (RN-3)

| # | Fato | Estado |
|---|---|---|
| F-1 | O workspace não é um repositório git | **entregue** (`behavior-definition`, 2026-08-26) |
| F-2 | Instantâneo não tomado não vira afirmação | **entregue** junto de F-1 |
| F-3 | Os portões que o projeto declara, nomeados | a fazer |
| F-4 | Quais deles já estavam vermelhos na abertura | a fazer — depende de `done-qualifier` |

### 5.2 Práticas — default embutido, sobreponível (RN-1)

| # | Prática | Default | Vira fato? |
|---|---|---|---|
| P-1 | Anunciar uma vez a ausência de repositório e oferecer `git init` | ligado | parcialmente: F-1 é o fato, o anúncio é a prática |
| P-2 | Conferir afirmação sobre caminho antes de escrevê-la (RN-4) | ligado | não — mediado |
| P-3 | Reler documento de estado que o próprio turno tornou obsoleto (RN-5) | ligado | não — mediado |
| P-4 | Não descontar exit code sem ordem citada (RN-7) | ligado | não — mediado |

Quatro práticas. É pouco de propósito, e a RN-8 diz por quê.

## 6. O que fica de fora, e por quê

- **Política de qualidade.** Cobertura mínima, fluxo de branch, obrigatoriedade
  de teste: é do dono do projeto. O piso não opina.
- **Rodar os portões.** É `done-qualifier`. Aqui eles só são nomeados.
- **Fazer `git init` sozinho.** Oferecer é prática; executar é ato do usuário
  sobre o repositório dele. O `vcs` deste produto lê e não escreve, e é uma
  garantia que não se troca por conveniência.
- **Impedir o agente de escrever um plano impossível.** Ele pode ler que não há
  repositório e ainda assim escrever "um commit por tarefa". A linha diz que não
  dá; obedecer é mediado. Impedir exigiria o harness julgar planos, que é uma
  máquina que este produto não quer.
- **Detectar prosa desatualizada em geral.** Só o caso da RN-5 — o documento que
  o **próprio turno** tornou obsoleto — está no piso. O caso geral é revisão de
  documentação, e não tem fim.
- **Um segundo modelo conferindo o primeiro.** Mesma recusa da
  `done-qualifier`: troca uma decisão não verificada por duas.

## 7. O risco desta família, dito antes de construí-la

**Um piso é uma superfície nova para o agente ser chato.**

Cada prática é uma chance de o modelo anexar uma ressalva, pedir uma
confirmação, ou reabrir uma decisão já tomada — e o produto já pagou esse preço
antes, na semana em que um agente recusava rodar um comando que o usuário
estava expressamente mandando rodar.

A RN-1 existe por causa disso e é a regra mais forte da família. A RN-2 é o
contrapeso e é a mais fácil de aplicar errado: "dizer uma vez" vira "avisar
sempre" sem que nenhuma linha de código mude.

Por isso os contratos comportamentais desta família, quando o `.p` os escrever,
medem sobretudo o **silêncio**: que o default sobreposto não é mencionado duas
vezes, que a menção não vira pergunta, e que o trabalho não espera por ela.

## 8. Changelog

- [202608262200 — o piso de prática e quem pode mudá-lo](changelog/202608262200-piso-de-pratica.md)
