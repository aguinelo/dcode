# Sai o passo que publicava num tap que nunca existiu

**Data:** 2026-08-20
**Specs afetadas:** `202608072352-distribution` (`.p`, `.i`, `.r`)

## O que mudou

O `scripts/publish-tap.sh`, os testes dele e o passo do workflow saíram. O
`brew install` saiu da lista de canais do `.p`. O tap virou item de roadmap.

A **fórmula continua sendo gerada** e anexada ao release, derivada do
`checksums.txt` assinado.

## Por que mudou

`aguinelo/homebrew-dcode` nunca foi criado, e `TAP_TOKEN` nunca foi configurado.

O passo estava **certo no desenho**: sem o segredo, saía com zero e avisava, para
não reprovar um release que já tinha dado certo. O efeito de estar certo assim foi
o v0.0.1 reportar sucesso com um canal que nunca existiu — e ninguém percebeu até
o README estar prestes a documentá-lo.

Maquinário que roda e não entrega nada é pior que ausência: ele **ocupa o lugar da
decisão que ninguém tomou** e faz o release parecer completo. O `.i` já dizia que
criar o tap é decisão do dono da conta; o que faltava era o pipeline parar de
fingir que a decisão podia ser adiada indefinidamente sem custo.

## A armadilha de nome, que estava lá desde o começo

O `.p` anunciava `brew install aguinelo/tap/dcode`. O script empurrava para
`aguinelo/homebrew-dcode`, cujo atalho no brew é `aguinelo/dcode/dcode`.

**Os dois nunca concordaram**, e nada os segurava juntos. Se o tap tivesse sido
criado, o comando documentado teria falhado — e o modo de falha seria idêntico ao
de não ter tap nenhum, o que é a pior forma de um erro se esconder.

Quando o tap for criado, os dois nomes precisam concordar e um teste precisa
segurá-los. Está anotado no `docs/ROADMAP.md`.

## Por que a fórmula fica

Ela é o artefato que o tap consome, é derivada e não digitada, e o `.i` registra
que é instalável por URL mesmo sem tap. Removê-la transformaria o trabalho futuro
de "criar repositório, segredo, e restaurar um passo" em "reconstruir a
derivação e o teste dela".

É uma escolha discutível — asset sem consumidor documentado é exatamente o padrão
que este repositório persegue. A diferença que a sustenta: o passo removido
**alcançava algo que não existe**; a geração produz um arquivo real, com sucesso.
