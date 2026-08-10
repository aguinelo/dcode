# Research: Configuração e Descoberta de Arquivos

> Fonte da verdade de negócio para **onde os arquivos moram na máquina do usuário**, em que formato, e em que ordem são resolvidos.
> Desbloqueia as pendências de caminho deixadas em `202608080016-behavior-definition`.

## 1. Contexto

Toda `.config.spec.md` deste projeto abre com a mesma linha de precedência — *config travada por administrador > flag > variável de ambiente > arquivo de config > default*. Essa linha nunca foi definida em lugar nenhum. Esta spec é o dono dela.

Também resolve o que ficou aberto na spec de comportamento: onde vivem arquivos de instrução e skills, e como são descobertos.

O produto é um binário estático único (ADR-01). O usuário não instala estrutura de diretório: o dcode cria o que precisa, na primeira vez, no lugar certo do sistema.

## 2. Fronteira de determinismo

**Regime: determinístico.**

Resolução de caminho, parsing de config, descoberta de arquivo e aplicação de precedência são regra explícita. Nenhuma mediação por modelo.

**Consequência para a revisão:** tudo é verificável por asserção contra sistema de arquivos real em diretório temporário. O `.p.spec.md` não tem seção de contratos comportamentais.

## 3. User stories

| # | Como | Quero | Para |
|---|---|---|---|
| US-1 | usuário | versionar minha config no meu repositório de dotfiles | levar preferências entre máquinas |
| US-2 | usuário | que log de sessão não polua o que eu versiono | log cresce; config não |
| US-3 | usuário | apagar cache sem perder configuração | recuperar disco sem reconfigurar |
| US-4 | usuário de monorepo | convenção diferente por pacote | um repositório, várias equipes |
| US-5 | usuário | reaproveitar instruções que já escrevi para outra ferramenta | não manter dois arquivos com a mesma regra |
| US-6 | usuário | dar instrução só ao dcode, sem afetar outras ferramentas | ajuste específico deste agente |
| US-7 | administrador | travar config para a equipe | política organizacional |
| US-8 | usuário | entender de onde veio um valor efetivo | depurar config sem adivinhação |
| US-9 | usuário | criar atalho para instrução que repito | reduzir digitação de tarefa recorrente |

## 4. Regras de negócio

### RN-1 — Quatro raízes, com propósitos distintos
Configuração, dados, estado e cache têm ciclos de vida diferentes e **não** compartilham diretório.

| Raiz | Contém | Versionável? | Descartável? |
|---|---|---|---|
| config | config do usuário, instruções globais, skills, comandos | **sim** (US-1) | não |
| dados | o que o usuário criou e quer manter | sim | não |
| estado | log de sessão, perfis de sandbox, socket | **não** (US-2) | sim, com perda de histórico |
| cache | consultas de versão, temporários | não | **sim, sem perda** (US-3) |

Juntar tudo num diretório só impede US-1, US-2 e US-3 ao mesmo tempo: ou você versiona 500 MB de log, ou não versiona a config.

Para quem prefere simplicidade, **uma única variável colapsa as quatro sob uma raiz** — a separação é o default, não uma imposição.

### RN-2 — Config é TOML
Tipada, comentável e sem sensibilidade a indentação. JSON não aceita comentário, o que é inaceitável em arquivo editado à mão. YAML tem armadilhas de indentação e de coerção de tipo que não pagam o ganho.

### RN-3 — Segredo não mora em arquivo de config
Chave de API vem de variável de ambiente ou do chaveiro do sistema. **Nunca** do arquivo de config.

Arquivo de config é feito para ser versionado (US-1) e sincronizado. Aceitar segredo ali é convidar o vazamento mais comum que existe.

Config que contenha campo com aparência de credencial é recusada na inicialização, com erro explicando de onde a credencial deve vir.

### RN-4 — `AGENTS.md` é o formato compartilhado; `DCODE.md` é o específico
`AGENTS.md` virou convenção entre ferramentas de agente. Ler esse arquivo significa aproveitar instrução que o usuário já escreveu (US-5).

`DCODE.md` existe para instrução que deve valer **só** para o dcode (US-6). No mesmo diretório, `DCODE.md` tem precedência — é o mais específico.

Escrever um formato próprio e ignorar o compartilhado seria custo para o usuário sem ganho para o produto.

### RN-5 — Descoberta é a cadeia da raiz do workspace até o diretório da sessão
Cada nível da cadeia contribui; o mais profundo tem mais peso (US-4).

A cadeia é resolvida **na criação da sessão** e congelada, porque o prefixo é imutável durante a sessão. Instrução que aparecesse depois invalidaria o prefixo inteiro.

### RN-6 — Instrução fora da cadeia vira lembrete, não é ignorada
O agente pode acabar trabalhando em diretório cujo arquivo de instrução não entrou na cadeia congelada.

Descartar em silêncio faria o agente violar convenção que o usuário escreveu. Injetar no prefixo quebraria a imutabilidade.

A saída é o terceiro canal: **lembrete anexado**. Direciona o comportamento sem tocar o prefixo — exatamente o problema para o qual aquele canal existe.

### RN-6.1 — Instrução compartilhada é traduzida no setup, nunca filtrada em execução
A RN-4 é certa e tem um efeito colateral: nome de arquivo compartilhado significa que **quem escreveu primeiro vence**, inclusive tendo escrito para outra ferramenta. O resultado observado é um prompt dominado por instrução que descreve máquina inexistente.

Filtrar por assunto em tempo de execução exigiria julgamento semântico, a cada turno, automático e invisível — e descartaria regra legítima do usuário em silêncio. **Filtro que erra calado é pior que filtro nenhum.**

A tradução acontece **uma vez, no setup**, e produz arquivo que uma pessoa revisa antes de valer. O erro passa a ser visível, revisável e único.

Três obrigações decorrem, e nenhuma é opcional:

- **Ferramenta citada é conferida contra o registro do produto**, e comando citado é conferido por **sonda de arquivo**. Ambos são fato, não julgamento.
- **Comando de arquivo de instrução nunca é executado para verificação.** O arquivo pode ter vindo de repositório clonado, e verificar executando é executar instrução de desconhecido dentro do workspace.
- **O que foi descartado é registrado com o motivo**, no arquivo gerado. Mesma exigência da RN-10 de `behavior-definition`, e pelo mesmo motivo: descarte invisível é descarte que ninguém audita.

### RN-6.2 — Instrução não traduzida avisa; nunca bloqueia
Havendo arquivo de instrução não escrito para o dcode e não havendo o específico, a sessão começa avisando, **com o número medido**.

Exigir setup antes de responder uma pergunta em repositório recém-clonado é a ferramenta burocrática da RN-9 de `behavior-definition`, e portão que trava vira portão que se atravessa no automático. O aviso basta porque o problema não é o produto não saber decidir — é o usuário não saber que está acontecendo.

**Reindex detecta, avisa o que mudou, e propõe.** O arquivo gerado guarda o digest das origens; divergiu, avisa nomeando o arquivo. Regenerar automaticamente destrói a edição manual que o usuário fez depois — e o arquivo específico é, por desenho, dele a partir da geração.

### RN-7 — Uma única cadeia de precedência, para tudo
```
config travada por administrador  ← vence tudo
flag de linha de comando
variável de ambiente
config do projeto
config do usuário
default do produto               ← perde para tudo
```

Vale para toda chave de toda spec, sem exceção nem caso especial.

### RN-8 — Valor efetivo é sempre explicável
O usuário consegue perguntar ao dcode qual é o valor efetivo de qualquer chave **e de onde ele veio** (US-8).

Config que não se explica produz o pior tipo de suporte: "no meu funciona".

### RN-9 — Config travada é visível, nunca silenciosa
Quando administrador trava uma chave, tentar sobrescrevê-la **informa** que está travada e por quem. Ignorar em silêncio faz o usuário achar que a mudança funcionou.

### RN-10 — Comando é expansão determinística de texto
Comando é atalho: expande para instrução, sem executar nada por conta própria (US-9). A expansão é determinística — mesma entrada, mesmo texto.

Comando que executasse ação direta seria uma segunda superfície de execução, fora do avaliador de política. Isso está proibido pela RN-6 da spec de sandbox e não abre exceção aqui.

## 5. Fora de escopo

- Interface gráfica de configuração.
- Sincronização de config entre máquinas — é papel do gerenciador de dotfiles do usuário.
- Chaveiro do sistema como origem de credencial no MVP; a regra RN-3 já reserva o lugar.
- Quais comandos embutidos existem: decisão de produto do cliente, não de configuração. Aqui só o **mecanismo**.

## 6. Changelog

- [202608101900 — Tradução de instruções de terceiros](changelog/202608101900-traducao-de-instrucoes-de-terceiros.md)
