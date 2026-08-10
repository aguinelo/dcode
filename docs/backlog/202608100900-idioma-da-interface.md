# Idioma da interface e da conversa

Pauta de discussão, não plano. Vira spec RPI quando for decidido.

**Estado medido em 2026-08-10**, sobre `a7187f9`.

## O pedido

O dcode conversa e apresenta a interface em **pt-BR por default**, com troca por
configuração.

## O que já funciona, e o que não

São dois eixos, e só um está aberto.

**Conversa — já resolvido.** A doutrina base carrega
`"Answer in the language the user wrote in."`, e é por isso que o modelo
respondeu em português a sessão inteira sem ninguém configurar nada. Adaptativo
é melhor que default fixo aqui: um repositório com issues em inglês e um usuário
que escreve em português continuam sendo atendidos cada um na sua língua.

> Se ainda assim quisermos forçar, é uma linha de doutrina, não i18n. Vale
> decidir se "default pt-BR" significa *forçar* ou significa *o que já
> acontece*.

**Interface — não existe.** Toda cadeia é literal em inglês no código.

## O tamanho, e por que ele é menor do que parece

Contagem grosseira de cadeias longas por pacote:

| Pacote | ~cadeias |
|---|---|
| `internal/tui` | 381 |
| `internal/tools` | 277 |
| `internal/update` | 137 |
| `internal/policy` | 111 |
| `cmd/dcode` | 76 |
| `internal/credential` | 53 |

Mas **nem toda cadeia é interface**, e a distinção decide o trabalho:

| Categoria | Quem lê | Traduzir? |
|---|---|---|
| Texto do cliente — TUI, `/help`, saída de CLI | a pessoa | **sim** |
| Erro de ferramenta — `read` antes de `edit`, caminho fora do workspace | **o modelo**, para se corrigir | **não** |
| Descrição de ferramenta e doutrina | o modelo | **não** |
| Mensagem de erro de configuração | a pessoa | sim |

Erro de ferramenta é o caso difícil: **o modelo lê para se corrigir e a pessoa
lê na tela**. Traduzir muda o prompt e invalida os limiares comportamentais
medidos — `tool-over-shell`, `reminder-acted-upon` e companhia foram aferidos em
inglês. A ADR-05 já diz que a formulação pertence à família; o idioma é a mesma
natureza de decisão.

Minha leitura: **o modelo continua em inglês, a pessoa vê o idioma dela.** Onde
o mesmo texto serve aos dois, quem manda é o modelo, e o cliente traduz apenas
o que ele mesmo compõe em volta.

## As perguntas a decidir

**1. Default pt-BR ou default do sistema?**

`LANG`/`LC_ALL` já dizem o idioma do usuário, e o dcode já os lê para decidir
Unicode. Default pelo sistema acerta sem configuração para todo mundo; default
pt-BR fixo é uma escolha de produto que precisa ser dita, não deduzida.

**2. Onde as cadeias moram?**

Mapa em Go embutido é a opção sem dependência e sem arquivo externo — cabe bem
num binário que se orgulha de não ter cgo. Arquivo por idioma facilita
contribuição de tradução e cria um caminho de carregamento em runtime, com o
custo de mais uma coisa que pode faltar.

**3. Chave ausente: cai para inglês ou aparece a chave?**

Cair é mais bonito e esconde tradução faltando para sempre. Mostrar a chave é
feio e é o que faz alguém corrigir. Vale um teste que exige cobertura total das
chaves em todo idioma declarado, e aí a pergunta some.

**4. Isso colide com a `LANGUAGE.md`?**

Não, mas parece. A convenção diz artefato canônico em inglês — comentário,
commit, spec em português. Ela fala de **quem contribui**. Interface fala de
**quem usa**, e são públicos diferentes. Vale escrever essa distinção na própria
`LANGUAGE.md` para ninguém ler uma como contradizendo a outra.

**5. O que fazer com largura?**

O renderizador já mede em células de exibição, então acento e CJK não quebram
layout. Mas texto traduzido é mais longo — alemão facilmente 30% —, e há linhas
com coluna fixa: a de ferramenta usa 6 células para o nome e 26 para o alvo. Um
teste de largura por idioma evita descobrir isso em captura de tela.

## Ordem sugerida

1. Decidir 1 e 4, que são de produto e não custam código.
2. Extrair só `internal/tui` e `cmd/dcode` — é a interface de verdade, e é o
   que o usuário vê.
3. Deixar erro de ferramenta e doutrina em inglês, explicitamente e com o motivo
   escrito na spec, para que a próxima pessoa não "conserte" isso.
