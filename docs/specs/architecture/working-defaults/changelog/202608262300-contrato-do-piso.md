# O contrato do piso

**Data:** 2026-08-26
**Specs afetadas:** `202608262200-working-defaults` ganha `.p` e `.config`.
Sem mudanças em outras famílias.

> **Estado.** F-1 e F-2 do catálogo existem em código, entregues antes desta
> spec na `behavior-definition`. O resto é desenho aprovado, não implementado.
> Sem `.i`, invariantes **previstas**, `.config` sem linha de tabela.

## A descoberta que encurtou o desenho

**A precedência que a `.r` pede já existe.** Não precisa de máquina nova.

O `Build` monta o prefixo em ordem, e o comentário dele sobre o bloco do
repositório já dizia o porquê: o que vem antes é **contexto para ler** o que vem
depois, não regra que compete com ele. As instruções do projeto são o **último**
bloco — a posição de maior peso — e a tabela `authority` já ordena as fontes
entre si.

Então a RN-1 (`prompt > projeto > default`) não é mecanismo a construir: é
consequência de o piso existir na posição de menor peso entre as regras. O que
falta é a camada de default **ter onde morar**.

Isso derrubou metade do que eu esperava especificar. Vale registrar, porque a
tentação era desenhar um resolvedor de precedência e ele teria sido a terceira
maneira de ordenar as mesmas coisas.

## Práticas são doutrina, e a diferença com `Safety` é o ponto

`Doctrine` ganha `Practices`. `DoctrineOverlay` ganha `Practices` também — e
**continua sem** `Safety`, que é a garantia da `behavior-definition` RN-12: uma
trava por tipo, não por convenção.

A assimetria é deliberada e é a regra inteira:

- `Safety` não tem campo no overlay **porque não pode ser sobreposta**.
- `Practices` tem, **porque um piso que não pode ser sobreposto não é piso** —
  é regra fingindo ser default.

E `Practices` vazia não faz `Build` falhar, ao contrário de `Identity` e
`Safety`. Piso vazio é piso desligado, e desligar é escolha legítima.

## Substituir, nunca acrescentar

`practices.md` **substitui** o texto embutido, como `identity.md` e `style.md`.
Não há variante que acrescenta.

Acrescentar a um piso produz dois pisos, e o segundo nunca é lido junto com o
primeiro. Quem quer desligar **uma** prática não usa o overlay: escreve uma
linha no arquivo do projeto, que é renderizado depois e por isso vence. O
overlay é para quem quer **outro** piso, não para quem quer ajustar este.

## A frase obrigatória do texto embutido

O conteúdo das quatro práticas é do PR que as implementa. Uma frase, não:

> Uma instrução do usuário ou do projeto que contradiga qualquer coisa desta
> seção **vence, sem discussão**. Diga uma vez qual foi, e siga.

Ela está no contrato porque é o que impede a família de virar o próprio risco
que a `.r §7` declarou. Um piso sem essa linha é uma superfície nova de
ressalva, e este produto já pagou esse preço.

## O inventário de portões, e a linha que o salva

`Probe` lê `package.json` e `Makefile`, produz `[]Gate`, e **não roda nada**.
Rodar é `done-qualifier`.

O bloco no prefixo termina com uma frase que é constante não configurável:

> These are what the project measures itself by. Nothing here says they pass.

Sem ela, uma lista de portões no prefixo lê como lista de **garantias** — e a
família teria produzido exatamente o defeito que a motivou, que foi um projeto
com quatro portões declarados, dois vermelhos desde o primeiro dia, e ninguém
sabendo.

## A lição do F-2, aplicada de novo

Duas invariantes previstas dizem a mesma coisa em contextos diferentes:

- projeto **sem portão declarado** não renderiza bloco nenhum;
- `Workspace` **nulo** também não renderiza nada, e nada no prefixo afirma que o
  projeto não declara portões.

"Não sondei" e "sondei e não há" não podem ler igual. É a distinção que custou
três guardas numa função só quando F-1 foi entregue, e é a que a implementação
desta etapa vai ter que refazer.

A diferença com F-1: ali, "não tem repositório" **muda o que terminar
significa** e por isso vale uma linha. Aqui, "não declarou portão" é comum e sem
consequência, e por isso não vale. Nem toda ausência é digna de nota — o que não
pode é a ausência não conferida virar afirmação.

## O que os contratos vão medir: silêncio

Cinco cenários, e os dois com limiar mais alto são `floor-does-not-ask` (≥ 95%)
e `floor-yields-to-user` (≥ 95%).

Se esta família tiver um modo de falhar, é virar superfície de ressalva. Os
outros três medem se o piso funciona; esses dois medem se ele **não atrapalha**,
e é por isso que são os mais exigentes.

`floor-checks-before-claiming` provavelmente nem é mediado: "todo caminho
afirmado ausente aparece antes num `read`/`glob`/`grep`?" é pergunta
determinística sobre o transcript. Se for, migra para `Asserted` e sai da
medição, como os quatro da `loop-command` fizeram. Fica escrito como direção.

## Ordem de entrega

1. A seção de práticas, a sobreposição e a origem — a seção nasce **vazia** e o
   prefixo não muda.
2. O texto embutido das quatro práticas.
3. O inventário de portões.
4. Os contratos.

**2 vai sozinha de propósito.** É a primeira mudança que um usuário sente, e é a
que mais provavelmente será reescrita depois de vista. Separá-la de 1 é o que
permite revertê-la sem levar a estrutura junto.

1 e 3 são independentes e podem ir em paralelo.

## Impacto previsto

- Um campo novo em `Doctrine`, um em `DoctrineOverlay`, um em `SectionOrigins`,
  e uma linha em `DoctrineAudit`.
- Um pacote novo `internal/workspace/` e um tipo novo no `behavior/`.
- Uma chave de configuração, ligada por default.
- Nada em `internal/loop/`. O piso é o que o agente sabe ao começar, não uma
  fase do ciclo.
