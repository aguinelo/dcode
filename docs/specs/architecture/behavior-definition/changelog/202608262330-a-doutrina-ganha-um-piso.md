# A doutrina ganha um piso, e ele é sobreponível

**Data:** 2026-08-26
**Specs afetadas:** `202608080016-behavior-definition` — uma invariante da §8
muda de "as quatro seções" para "cada seção", e seis nascem. Implementa a etapa
1 da `202608262200-working-defaults.p §8`.

## O que passou a existir

`Doctrine` ganha `Practices`: o piso, o que o dcode faz quando ninguém pediu.

A seção nasce **vazia**. Este changelog registra a estrutura; o texto das
práticas é a etapa 2 e vai sozinha, de propósito — é a primeira mudança que um
usuário sente, e é a que mais provavelmente será reescrita depois de vista.
Separá-la é o que permite revertê-la sem levar a estrutura junto.

## A assimetria com `Safety` é a regra inteira

| | campo no `DoctrineOverlay` | por quê |
|---|---|---|
| `Safety` | **não tem** | porque não pode ser sobreposta. Trava por tipo, não por convenção — RN-12. |
| `Practices` | **tem** | porque um piso que não pode ser sobreposto não é piso: é regra fingindo ser default. |

E `Practices` vazia **não** faz `Build` falhar, ao contrário de `Identity` e
`Safety`. Piso vazio é piso desligado, e desligar é escolha legítima; agente sem
identidade não é agente degradado, é agente imprevisível.

## A descoberta que encurtou a implementação

**A precedência não precisou de máquina nenhuma.**

O `Build` monta o prefixo em ordem, e o comentário sobre o bloco do repositório
já dizia o porquê: o que vem antes é contexto para ler o que vem depois, não
regra que compete com ele. As instruções do projeto são o **último** bloco.

Então basta o piso ser renderizado depois de `Safety` e antes de tudo que
alguém efetivamente disse, e `prompt > projeto > default` sai de graça — de
posição, não de resolvedor. A tentação era desenhar um terceiro eixo de
precedência ao lado dos dois que já existem, e ele teria sido a terceira maneira
de ordenar as mesmas coisas.

Duas invariantes guardam isso, e a segunda é a que importa: as instruções do
projeto continuam sendo o último bloco. No dia em que deixarem de ser, o piso
passa a vencer quem devia vencê-lo, e nada mais no código diria isso.

## Substituir, nunca acrescentar

`practices.md` no diretório de doutrina **substitui** o texto embutido, como
`identity.md` e `style.md` já fazem. Não há variante que acrescenta — ao
contrário de `tools.md`, que acrescenta e nunca substitui.

Acrescentar a um piso produz dois pisos, e o segundo nunca é lido junto com o
primeiro. Quem quer desligar **uma** prática não usa o overlay: escreve uma linha
no arquivo do projeto, que é renderizado depois e por isso vence. O overlay é
para quem quer **outro** piso.

## Um teste que quebrou por bom motivo

`TestOriginsReportAllFourSectionsAndSafetyIsAlwaysBuiltin` montava
`SectionOrigins` por posição, e acrescentar a quinta seção quebrou os cinco
casos de uma vez.

A pressão funcionou — mas ela só precisou ser sentida porque a ordem dos campos
era carga num teste sobre **qual seção veio de onde**, que é justamente a coisa
que a posição não deveria decidir. Passou a ser literal com chave, e o teste
mudou de nome junto: `TestOriginsReportEverySectionAndSafetyIsAlwaysBuiltin`.

O `specguard` pegou o rename no mesmo instante, como devia: a invariante nomeava
o teste antigo.

## Invariantes

Uma muda — "as quatro seções" vira "cada seção", que é o que ela sempre quis
dizer e agora não envelhece na próxima. Seis nascem, cobrindo: piso vazio não
reprova o `Build`; a posição da seção; as instruções do projeto continuarem por
último; `practices.md` substituir; a sobreposição alcançar `Practices` e nunca
`Safety`; e o piso substituído ser reportado como tal.

A última é a RN-2 da `working-defaults` no único lugar onde ela é
determinística. O resto dela — dizer uma vez, não virar pergunta — é
comportamento, e é contrato medido, não asserção.

## O que isto não faz

Não muda o prefixo. Com a seção vazia, a saída de `Build` é byte a byte a de
antes, e há teste para isso.
