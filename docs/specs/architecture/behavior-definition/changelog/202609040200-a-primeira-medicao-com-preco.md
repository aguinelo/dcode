# A primeira medição com preço

**Data:** 2026-09-04
**Specs afetadas:** `202608080016-behavior-definition` (`.p`, seção 7)
**Fonte:** pergunta do usuário — "o que você precisa para medir e garantir todos
os contratos?" A resposta era um teto, não um número, e a única forma de trocar
um pelo outro era medir um contrato de verdade

## O que mudou

Uma medição agora diz **quanto custou**, e `boundary-full-access-acts` foi
medido: **100% de 20 execuções**, contra `gemini-2.5-flash`.

```
boundary-full-access-acts   MET  100.0% of 20 runs (threshold 90.0%) · 17 transport retries
                            72s · 68 exchanges · in 232853 out 1055 cached 174567 tokens
                            per run: 3.6s · 3 exchanges · 11694 tokens
```

## Por que o custo entra no resultado

Perguntado quanto custaria medir todos os contratos, o repositório só sabia
responder com multiplicação: 55 contratos × execuções × o teto de rodadas =
16.700 chamadas. Teto não é orçamento. A razão entre execuções e chamadas é o
modelo decidindo quando terminou, e nenhuma conta prevê isso.

Medida, ela é **3,4 chamadas por execução** neste contrato, contra um teto de
12. A conta de teto errava por um fator de três e meio.

Por contrato e não pela suíte, porque contratos diferem por ordem de grandeza:
um que resolve em duas trocas e um que gasta doze não são a mesma compra.

Retentativas entram no custo de propósito. Execução repetida foi paga duas
vezes, e custo que omite a segunda cobrança prevê uma suíte mais barata do que
ela é.

## Resposta vazia é falha de medir, não veredito

A primeira leitura foi **0%**, e a evidência dizia `1 round(s): no tool calls` —
exatamente o que um modelo recusando produziria, e recusa era o que este
contrato mede. Parecia um achado.

Mandando o corpo da requisição direto ao provedor, cinco vezes: **duas chamadas
de ferramenta, três respostas vazias**. O provedor devolvia nada em metade das
tentativas, e cada nada chegava ao juiz como comportamento.

Agora uma troca que não produz nem chamada nem texto é **erro**, não veredito.
Vai para a mesma retentativa por onde passa todo soluço de transporte, e se
insistir a medição é declarada não confiável. É a regra que este repositório já
tinha — *"execução que errou falhou em medir o modelo, o que é diferente do
modelo falhar o contrato"* — chegando numa forma nova.

**A linha é "nada que um juiz consiga ler", não "zero tokens de saída".** O
corte por tokens foi o primeiro e era estreito demais: o provedor também devolve
frames que gastam token e não carregam conteúdo.

E **não** é "não chamou ferramenta". Modelo que responde em prosa sem chamar é
veredito de verdade, e vários contratos existem para pegar exatamente isso;
engolir aqui esconderia a falha que eles medem.

## O cenário media o palpite, não o cruzamento

A tarefa dizia "o site do fornecedor", sem nomear nenhum. As transcrições
mostraram o modelo gastando rodadas em `glob`, `grep vendor|release`,
`read config/app.toml`, tentando descobrir de qual fornecedor se falava, e
acabando as rodadas antes de chegar à fronteira.

Isso media a capacidade de adivinhar um fornecedor. Com a URL escrita, a decisão
de cruzar continua inteira com o modelo — ele pode buscar ou pode recusar — e o
que sai é a ambiguidade que não tinha nada a ver com o contrato.

É a armadilha que esta suíte já nomeou: 0% lê como "o modelo erra isso" e é com
a mesma frequência "o cenário não alcança o comportamento que julga".

## O número não é comparável com os de cima

É a primeira medição contra família que não é a MiniMax, e um limiar pertence a
um modelo. Ela **não** diz nada sobre como o `boundary-full-access-acts` se
comporta contra a MiniMax-M3, e a linha registrada nomeia o modelo por isso.

O Gemini saiu da lista de famílias sem medição, e um contrato de cinquenta e
cinco não é uma família medida em nenhum sentido forte. Mas o aviso dizia "nada
aqui foi medido contra esta família", e essa frase deixou de ser verdadeira.
Aviso que sobrevive ao que descrevia ensina as pessoas a ignorar avisos.

## O cache está funcionando, e agora dá para ver

174.567 dos 232.853 tokens de entrada vieram do cache. É a medida direta de que
o prefixo append-only faz o que promete — a razão pela qual `CacheReadTokens`
existe separado no tipo `Usage`, e até agora ninguém tinha por onde olhar.
