# Workspace sob `/tmp` continua visível dentro do sandbox

**Data:** 2026-08-14

## O que mudou

O bubblewrap aplica montagens **na ordem em que recebe os argumentos**, e o
`/tmp` gravável vinha depois do bind do workspace:

```
--ro-bind / /  --chdir <ws>  --bind <ws> <ws>  --tmpfs /tmp
```

Para um workspace em qualquer lugar sob `/tmp`, o `tmpfs` novo era montado por
cima do bind recém-feito e o diretório **deixava de existir** dentro do sandbox.
Todo comando falhava antes de rodar:

```
bwrap: Can't chdir to /tmp/.../ws: No such file or directory
```

Em `read-only` era pior: nada remontava o workspace, então ele simplesmente
sumia.

A ordem foi invertida — `tmpfs` primeiro, workspace por cima — e o `read-only`
passou a remontar o workspace com `--ro-bind` quando ele está sob `/tmp`.
Manter o workspace visível não pode ser a coisa que o torna gravável.

## Por que passou tanto tempo

Duas cegueiras que se somaram:

- **No macOS o caso não existe.** `t.TempDir()` devolve algo sob `/var/folders`,
  não sob `/tmp`, e o backend é o seatbelt. Nenhum teste de macOS chega perto.
- **No Linux a CI não conseguia criar namespace.** O Ubuntu 24.04 restringe
  namespaces de usuário não privilegiado via AppArmor. O passo de CI instalava o
  `bwrap` e dizia, em comentário, exatamente o que estava em jogo — *"a skipped
  boundary test reads as a passing one"* — mas instalar não bastava. Todo teste
  que precisava de fronteira real **pulava**, na única plataforma cujo backend é
  o bubblewrap.

Resultado: `internal/app` rodava a 78,8% no Linux contra 95%+ no macOS, e ninguém
olhava o número por plataforma.

Encontrado ao subir o gate de cobertura para 95%: 95,0% no macOS e 93,8% no
Linux. Diferença desse tamanho nunca é sobre os testes.

## O que passou a valer

Uma invariante nova em `## 6. Invariantes verificáveis`, cobrada por teste:

- Workspace sob `/tmp` continua visível dentro do sandbox: o `tmpfs` é montado
  antes dele, nunca por cima.

E a CI Linux passou a levantar a restrição do AppArmor e a **provar** que o
sandbox funciona, rodando a mesma sonda que o produto roda na inicialização. Um
passo que falha em silêncio é o que produziu tudo isso.
