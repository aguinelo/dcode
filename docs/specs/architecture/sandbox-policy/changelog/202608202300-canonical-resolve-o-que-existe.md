# `canonical` resolve até onde dá, e a resposta não muda quando o caminho nasce

**Data:** 2026-08-20
**Specs afetadas:** `202608072336-sandbox-policy` (`.p`)

## O que mudou

O `canonical()` devolvia o caminho **cru** quando ele ainda não existia — o
`filepath.EvalSymlinks` só resolve o que existe. Agora ele resolve o ancestral
mais profundo que existe e recoloca o resto.

E as três decisões de `/tmp` no `bubblewrap.args` passam a comparar contra
`tmpRoot()` — `/tmp` como o próprio `canonical()` o reporta — em vez do literal
`"/tmp"`.

## Por que mudou

Porque o mesmo diretório canonicalizava de **dois jeitos**, conforme o momento em
que se perguntava: `/tmp/ws` antes de ser criado, `/private/tmp/ws` depois. A
comparação seguinte era contra o literal `"/tmp"`, então a decisão de remontar o
workspace por cima do `tmpfs` seguia o **sistema de arquivos**, não o modo.

O sintoma foi um teste de fronteira de segurança que passava ou falhava conforme
`/tmp/ws` existisse na máquina de quem rodava. Ele ficou um dia inteiro escondido
atrás do cache de testes do Go: todo `make check` dizia verde até um
`go clean -testcache`. **Teste de fronteira que responde diferente em máquinas
diferentes é pior que teste que falha, porque é acreditado.**

## O que isso conserta além do teste

O comentário em cima do `canonical()` já dizia o perigo, e o próprio `canonical()`
o produzia: *"no macOS `/var` e `/tmp` são symlinks para dentro de `/private`,
então um profile nomeando o caminho não resolvido não concede nada e toda escrita
falha sem explicação."*

O caminho não resolvido era exatamente o que ele devolvia para um workspace ainda
não criado — e o `seatbelt` (que é o backend do macOS, onde o perigo é real) usa a
mesma função. A frase que justificava o fallback, *"um workspace que ainda não
existe ainda produz um profile utilizável"*, continua verdadeira; o que muda é que
"utilizável" passa a significar **resolvido**.

Em produção o `bubblewrap` só roda no Linux, onde as duas grafias já são iguais,
então nenhuma lista de argumentos muda lá. O ganho está no macOS e na
previsibilidade.

## Contrato substituído, não afrouxado

`TestCanonicalFallsBackToTheInput` afirmava o comportamento antigo pelo nome. Foi
**reescrito** para o novo — `TestCanonicalResolvesAsFarAsItCanAndStaysPut` —, que
cobra as duas metades: o ancestral existente é resolvido, e criar o diretório não
muda a resposta.

Três asserções em `sandbox_test.go`, `scratch_test.go` e `tmp_workspace_test.go`
fixavam a grafia crua de caminhos de fixture. Passam a comparar contra
`canonical(...)`, que é o que o `args()` monta. Não é afrouxamento: era asserção
que presumia grafia, e a grafia nunca foi a garantia.

## Alternativa descartada

**Canonicalizar só o lado do `/tmp`**, deixando o `canonical()` como estava. Foi
tentado primeiro e apenas **inverteu** o defeito: com o workspace ausente a
comparação passava a falhar, com ele presente passava a valer. O mesmo bug com
o sinal trocado — porque a causa nunca foi a comparação, foi a função devolver
duas respostas para a mesma pergunta.
