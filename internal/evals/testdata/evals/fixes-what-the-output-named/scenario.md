# fixes-what-the-output-named

**Contrato:** `202608281900-failure-feedback.p.spec.md` · limiar **85%**

O contrato que faltava. Todos os outros desta suíte medem o que o modelo faz
depois de um lembrete **injetado**; este roda o ciclo de verificação de verdade
— critérios reais, `loop.Check`, `loop.Moved`, o lembrete montado pelo produto —
e mede se a saída do critério que falhou serve para alguma coisa.

## Por que ele precisou existir

Duas famílias inteiras foram entregues sem medição possível. A `failure-feedback`
faz a saída do erro chegar ao modelo; a `recoverable-cycle` desfaz o ciclo que
regride. Nenhum contrato existente percorre esse caminho: o arcabouço injetava a
frase que o ciclo teria produzido e nunca rodava o ciclo.

O resultado foi uma medição que não media nada — 94% e 90% sobre um caminho de
código que a execução não visita — e a `.p` da `recoverable-cycle` registra isso
como etapa riscada.

## O material

`Slugify` devolve a entrada intacta. Três critérios, todos vermelhos no começo,
e cada um nomeia uma parte diferente do que falta. A saída de cada um diz
literalmente o que o arquivo não tem.

Os critérios são **predicados sobre o workspace**, não comandos de shell. O
arcabouço não executa o que um modelo escreveu e não executa o que um fixture
escreveu tampouco: um cenário que saísse para o shell dependeria do que estivesse
instalado naquela tarde.

## O que se mede

Que os três critérios fiquem verdes dentro do turno. É a medida mais direta de
"a correção foi boa" que esta suíte tem — não a frase final, não a intenção
declarada, mas o estado do arquivo contra a régua.

## O que este cenário ainda NÃO pega

**Se a correção é boa por dentro.** `contains: ToLower` fica verde com
`strings.ToLower` chamado em qualquer lugar, inclusive num lugar que não serve.
O critério mede presença, não sentido — e é a mesma limitação que qualquer
critério baseado em comando tem quando o comando é fraco.

O que ele pega, e nada mais pegava, é o ciclo inteiro acontecendo.
