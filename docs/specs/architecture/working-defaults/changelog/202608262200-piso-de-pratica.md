# O piso de prática e quem pode mudá-lo

**Data:** 2026-08-26
**Specs afetadas:** nova família `202608262200-working-defaults` (só o `.r`).
Sem mudanças em outras famílias.

> **Estado.** Existe o `.r`. Não existe `.p`, `.config`, `.i` nem código —
> exceto F-1 e F-2 do catálogo, que foram entregues antes desta spec, na
> `behavior-definition`, porque eram determinísticos e não dependiam deste
> desenho.

## De onde veio

Da auditoria de um projeto real, em 2026-08-26, depois de um dia de trabalho de
agente. Dois padrões apareceram lá, e os dois já tinham aparecido no próprio
repositório do dcode na mesma semana.

**Afirmação não conferida.** Um documento de análise listava cinco conflitos
"🔴 bloqueantes, resolver antes de implementar". Quatro já estavam resolvidos
nos arquivos citados. Um deles "propunha adicionar" uma lista de caminhos
reservados que estava no arquivo, palavra por palavra, com comentários citando
as specs que o documento dizia estarem em conflito.

Do outro lado, no dcode: um `CHANGELOG.md` afirmando que um comando rodava,
enquanto o `ROADMAP.md`, **no mesmo commit**, dizia que não havia código nenhum.

Direções opostas, mecanismo idêntico: um valor copiado de uma verdade que se
move. O documento foi escrito uma vez e nunca releu o mundo que descrevia.

**Portão não lido.** Quatro portões declarados, dois vermelhos desde o primeiro
dia — `lint` quebrado, cobertura em 0% contra um piso de 80% que o arquivo do
projeto chamava de "mínimo pra merge". O terceiro passava verde medindo
`1 + 1 === 2`.

E por baixo: nenhum repositório git, num projeto cujo arquivo exigia um commit
por tarefa. O harness olhou na abertura e não disse, porque estava escrito no
código que aquilo era "ordinary and silent".

## A decisão de forma

O piso é **pequeno** e cada linha dele veio de um defeito observado, não de uma
lista de boas práticas.

E ele **não** é política de qualidade. Não decide se o projeto deve ter testes,
qual cobertura, qual fluxo de branch — isso é do dono do projeto. O piso é sobre
não afirmar o que não se conferiu e não deixar portão declarado sem leitura, que
valem em qualquer projeto com qualquer política. É por isso que podem ser
default.

## A precedência, que foi o pedido explícito

```
prompt do usuário (este turno)  >  arquivo do projeto  >  default embutido
```

O de cima **substitui** o de baixo. Não negocia, não pondera, não pede
confirmação, não avisa que contraria a boa prática.

Não é tolerância a risco: é a constatação de que o dono do projeto sabe coisas
que o harness não sabe, e de que um default que argumenta com quem o sobrepõe
custa mais do que entrega. A alternativa já foi medida neste produto, e o nome
dela é a sessão que não anda.

## A regra que quase ficou implícita

Sobrepor é **obedecer e dizer uma vez**. E "dizer" precisou ser distinguido de
"perguntar", explicitamente, porque a distância entre os dois é onde a RN-1
morre com aparência de estar sendo cumprida:

| é | não é |
|---|---|
| "o default X está desligado pelo `DCODE.md`, linha 87" | "quer mesmo desligar X?" |
| uma vez, no começo | a cada turno |
| afirmação | ressalva anexada ao trabalho |
| e o trabalho segue | e o trabalho espera |

Um agente que transforma o relato em ressalva, ou em "vou fazer como você pediu,
mas note que…", quebrou a regra parecendo cumpri-la.

A justificativa é do próprio repositório, escrita em `doctrine_overlay.go` para
as seções de doutrina:

> "uma substituição invisível seria pior que a imutabilidade que ela substitui,
> porque o único jeito que o usuário tem de saber o que chegou ao modelo é ler"

`Origin` e `SectionOrigins` já existem para isso. Falta existirem para práticas.

## A distinção que mantém a precedência honesta

**Fato não se sobrepõe; prática sim.**

O `DCODE.md` pode desligar o anúncio da ausência de repositório. Ele não pode
fazer o workspace ser um repositório.

Sem essa linha, "precedência absoluta" viraria uma máquina de produzir
afirmações falsas por ordem de arquivo. O que se sobrepõe é **o que o dcode
faz**; o que ele **observou** não está em disputa.

## O que pode virar fato não vira prosa

É a regra de projeto da família, e vale mais que qualquer das outras.

Prosa é a camada mais fraca que este repositório reconhece — está no `AGENTS.md`
e foi provada três vezes nesta semana. Um piso escrito como exortação vale o que
valer a atenção do modelo naquele turno; o mesmo piso escrito como **fato no
prefixo** é lido antes de qualquer decisão. O `repo.go` já tinha o argumento:

> "uma regra que precisa de consulta primeiro é uma regra seguida por acidente"

Daí o catálogo estar dividido em fatos e práticas, e daí a regra de que toda
prática conversível em fato deve ser convertida. O número de linhas em prosa
deve encolher com o tempo, nunca crescer.

## O que já foi entregue antes desta spec

F-1 e F-2 — o workspace sem repositório é dito uma vez, e o instantâneo não
tomado continua calado. Entregues na `behavior-definition` em 2026-08-26,
antes deste `.r` existir, porque eram determinísticos, pequenos, e não
dependiam de nenhuma decisão que este documento toma.

A ordem invertida é deliberada e vale registrar: o fato pôde ir sozinho
justamente por ser fato. As práticas não podem, porque precisam da precedência
e da visibilidade que só este desenho define.

## O risco, escrito antes de construir

**Um piso é uma superfície nova para o agente ser chato.**

Cada prática é uma chance de anexar ressalva, pedir confirmação, ou reabrir
decisão tomada. O produto já pagou esse preço, na semana em que um agente
recusava rodar um comando que o usuário estava expressamente mandando rodar.

A RN-1 existe por causa disso. A RN-2 é o contrapeso, e é a mais fácil de
aplicar errado: "dizer uma vez" vira "avisar sempre" sem que uma linha de código
mude.

Por isso os contratos comportamentais desta família, quando o `.p` os escrever,
vão medir sobretudo o **silêncio**: que o default sobreposto não aparece duas
vezes, que a menção não vira pergunta, e que o trabalho não espera por ela.

## Impacto previsto

- Um campo novo na doutrina para as práticas, sobreponível — ao contrário de
  `Safety`, que não é, e pelo mesmo motivo de tipo e não de convenção.
- `Origin` estendido às práticas, para o relato da RN-2.
- Sondagem de portões declarados no `vcs` ou vizinho, entrando no prefixo como
  fato.
- Nada em `internal/loop/`. O piso é o que o agente sabe ao começar, não uma
  fase do ciclo.
