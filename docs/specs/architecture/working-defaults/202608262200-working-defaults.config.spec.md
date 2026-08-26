# Config: Piso de prática e precedência

> Nenhuma variável de ambiente nova no código sem estar aqui.
> Precedência: flag > variável de ambiente > arquivo de config > default.

## 1. Chaves previstas — **nenhuma declarada ainda**

Este arquivo não declara chave nenhuma. Linha de tabela num `.config.spec.md`
**é** a declaração — `TestEveryKeyTheSpecsDeclareIsReadSomewhere` lê as linhas e
reprova a chave que nenhum código consome — e o que ainda não existe não é
reivindicado como existente. A linha de tabela entra no PR da etapa que lê a
chave.

Mesma forma das "Invariantes previstas" do `.p §7`, pelo mesmo motivo.

### 1.1 A única chave prevista — etapa 3

**`DCODE_WORKSPACE_GATES`**, bool, default `true`. Liga o inventário de portões
declarados no prefixo (`.p §5`).

Nasce **ligada**: a sonda lê dois arquivos, não roda nada, e o custo é uma
leitura na abertura da sessão. Uma sonda barata que nasce desligada é uma sonda
que ninguém liga.

Ela existe por um motivo só, e é honesto declará-lo: um repositório grande com
um `Makefile` de setenta alvos gastaria contexto com uma lista que ninguém lê. O
teto da `.config §3` cobre o caso comum; a chave cobre quem quiser nada.

## 2. O que **não** vira chave, e por quê

**Ligar e desligar cada prática.** É o arquivo do projeto que faz isso, e é a
RN-1 da `.r`. Uma chave de configuração para desligar uma prática seria um
terceiro eixo de precedência ao lado dos dois que já existem, e o terceiro eixo
é sempre o que ninguém lembra de consultar.

**Substituir o texto do piso.** É o `practices.md` no diretório de doutrina, que
é o mecanismo que já existe para `identity.md` e `style.md`. Uma chave que
carrega prosa é um arquivo com passos extras.

**Escolher quais fontes o `Probe` lê.** Duas, fixas. Uma lista configurável de
lugares onde procurar portão é configuração que muda o que o prefixo **afirma**
sobre o projeto, e afirmação do harness não se configura — é a RN-3 da `.r`,
fato não se sobrepõe.

**Um modo silencioso do piso inteiro.** Já existe: `practices.md` vazio. Uma
chave para o mesmo efeito seria a segunda maneira de fazer a mesma coisa, e a
segunda maneira é a que fica desatualizada.

## 3. Constantes não configuráveis

| Constante | Valor | Motivo |
|---|---|---|
| `Practices` vazia não faz `Build` falhar | sempre | Piso vazio é piso desligado, que é escolha legítima. Identidade e segurança vazias não são. |
| `DoctrineOverlay` **sem** campo para `Safety` | sempre | Trava por tipo, não por convenção. É a garantia da `behavior-definition` RN-12 e nada aqui a toca. |
| `practices.md` substitui, nunca acrescenta | sempre | Acrescentar a um piso produz dois pisos, e o segundo nunca é lido junto com o primeiro. |
| Posição da seção: depois de `Safety`, antes de `Using tools` | sempre | A posição **é** a precedência. Instrução do projeto é o último bloco e por isso vence. |
| Fontes do `Probe`: `package.json` e `Makefile` | sempre | §2. Acrescentar fonte é aditivo e barato quando alguém tiver uma; antecipar é lista que ninguém lê inteira. |
| `Probe` não executa portão nenhum | sempre | Rodar é `202608261730-done-qualifier`. Aqui só se diz que existem. |
| O bloco de portões diz que nada ali afirma que eles passam | sempre | Sem essa linha, lista de portões lê como lista de garantias — e a família teria produzido o defeito que a motivou. |
| Instrução do usuário e do projeto vencem o piso, sem discussão | sempre | RN-1 da `.r`. Um default que argumenta com quem o sobrepõe custa mais do que entrega. |
| Sobreposição é dita **uma vez**, e dizer não é perguntar | sempre | RN-2 da `.r`. É onde a RN-1 morre parecendo estar sendo cumprida. |

## 4. Medição de contratos comportamentais

**Nenhum limiar desta família foi medido.** Os cinco alvos da §6 do `.p` são
alvos, não resultados.

Quando a etapa 4 da §8 existir, `DCODE_EVAL_MODEL`, `DCODE_EVAL_RUNS` e
`DCODE_EVAL_ENABLED` passam a valer — declaradas em
`202608072334-provider-adapter.config.spec.md` §4, não redeclaradas aqui — e
cada número medido entra com modelo e data em `changelog/`.

O judge de `floor-checks-before-claiming` provavelmente **não** é mediado: a
pergunta "todo caminho afirmado ausente aparece antes num `read`/`glob`/`grep`?"
é determinística sobre o transcript. Se for, o contrato migra para `Asserted` e
sai da medição, como os quatro da `loop-command` fizeram.

## 5. Changelog

- [202608262200 — o piso de prática e quem pode mudá-lo](changelog/202608262200-piso-de-pratica.md)
- [202608262300 — o contrato do piso](changelog/202608262300-contrato-do-piso.md)
