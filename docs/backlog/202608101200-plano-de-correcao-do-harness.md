# Plano de correção do harness

Levantado depois do primeiro exercício de auto-hospedagem, com números medidos
sobre `a7187f9`. Cada item vira branch e PR próprios.

## O diagnóstico em dois números

```
prompt total:        14.286 bytes
doutrina do dcode:    1.376  (10%)
instruções do repo:  12.910  (90%)
```

Noventa por cento do que o modelo recebe é o `AGENTS.md`/`CLAUDE.md` deste
repositório, e ele descreve **outra ferramenta**: 17 menções a `claude-flow`,
11 a `swarm`, 4 a `ruflo`, 2 a `agent_spawn`. As ferramentas listadas ali são
papéis de agente do ruflo, não as sete do dcode.

E a doutrina **nunca manda verificar o próprio trabalho** — procurei por `test`,
`verify`, `build` e `check` em `DefaultDoctrine`: zero ocorrências. No exercício
ele rodou `make check` porque o enunciado mandou.

Os dois juntos explicam a maior parte da distância observada: ele navega com
instrução majoritariamente irrelevante e sem obrigação de conferir o resultado.

---

# Tema 0 — Onde se edita o comportamento

**Pré-requisito dos demais**, porque decide se as correções são código ou
configuração.

## Estado

| Camada | Editável | Onde |
|---|---|---|
| `Doctrine.Identity` | **não** | constante Go |
| `Doctrine.ToolPolicy` | **não** | constante Go |
| `Doctrine.Safety` | **não** | constante Go |
| `Doctrine.Style` | **não** | constante Go |
| Instruções de projeto | sim | `AGENTS.md`, `DCODE.md` |
| Comandos | sim | `commands/*.md` |
| Skills | sim | `skills/*.md` |
| Chaves declaradas | sim | `config.toml` |

`doctrine.dump` apenas **exibe**; não existe chave que altere.

## A questão

A RN-10 diz que `Safety` não é sobrescrevível por instrução de usuário, e está
certa: um `AGENTS.md` hostil desligaria a aprovação. Mas a decisão foi tomada
para segurança e acabou valendo para as quatro partes — inclusive `Style` e
`Identity`, que não têm essa razão.

**Perguntas a decidir:**

1. `Style` e `Identity` viram configuráveis? É onde mora "responda no idioma do
   usuário", "prefira passos pequenos", "seja conciso" — preferência legítima de
   quem usa.
2. `ToolPolicy` é o caso intermediário: descreve *como* usar ferramenta. Ampliar
   é perigoso, restringir não.
3. `Safety` continua imutável — e isso deve estar escrito como invariante, não
   como consequência de tudo ser constante.
4. Se virar editável, **como se lê o que está valendo?** Um `--dump-prompt` já
   responde, mas a procedência de cada trecho não.

---

# Tema 1 — A doutrina não manda verificar

**Maior impacto, menor custo.** Faço primeiro.

Um agente que roda o gate antes de dizer "pronto" erra menos porque *descobre*
que errou. Hoje ele só verifica se for mandado, tarefa a tarefa.

**Proposta:** uma linha de doutrina, mais uma chave com o comando de verificação
do projeto (`verify.command`, ex. `make check`). A doutrina diz *que* verifique;
a chave diz *como*, porque o comando é do projeto e não do produto.

**Cuidado:** não pode virar "rode testes sempre", que gasta iteração em tarefa de
leitura. A formulação precisa amarrar a verificação a *ter mudado algo*.

**Medida de sucesso:** repetir o exercício de auto-hospedagem **sem** dizer "rode
`make check`" e ele rodar assim mesmo.

---

# Tema 2 — Instrução de terceiros entrando como se fosse dele

O dcode lê `AGENTS.md` inteiro e trata como instrução própria. Não é peculiar a
este repositório: qualquer projeto que use Claude Code, Cursor ou Codex tem
esses arquivos.

**Não dá para filtrar por semântica** — e tentar seria pior, porque erraria
silenciosamente. As saídas honestas:

1. **Avisar por tamanho.** Bloco de instrução acima de um limiar emite aviso com
   o número. Barato, e faz a pessoa olhar.
2. **Precedência que filtra.** `DCODE.md` presente faz `AGENTS.md` virar
   secundário — ou ser ignorado. A spec já define a precedência; ela só não é
   usada para *excluir*.
3. **Mostrar a conta.** `dcode config` já diz de onde vem cada valor; podia dizer
   quantos bytes de instrução vêm de cada arquivo.

Minha inclinação: **1 e 3 agora**, 2 depois de decidido — 2 muda comportamento de
quem já tem os dois arquivos.

---

# Tema 3 — Sem delegação

Um contexto, um laço. O M3 tem 1M de janela e isso mascara o custo; com Claude
(200k) a mesma tarefa não cabe.

Sem um jeito de dizer "explore este subdiretório e reporte", todo trabalho grande
polui um contexto só, e a compactação a 80% de 1M chega tarde demais para ajudar.

**Escopo mínimo que resolve a maior parte:** um sub-turno de leitura, sem
escrita, cujo resultado volta como resumo. Não é multi-agente; é uma chamada
aninhada com orçamento próprio.

**A decidir:** herda a mesma fronteira de sandbox (deve), pode chamar
ferramenta de escrita (não), conta no mesmo teto de iteração (sim).

---

# Tema 4 — Busca é `grep` linear

Sem índice de símbolo, achar coisa em repositório grande queima iteração. O teto
de 200 iterações do M3 esconde isso até não esconder.

**Menor passo com valor real:** um `grep` que entende símbolo — definição de
função, tipo, método — em vez de casar texto. Não precisa de embedding nem de
índice persistente; precisa entender a linguagem do arquivo.

**Nota:** isso interage com o Tema 3. Delegar uma exploração barata pode valer
mais que indexar, e é mais simples. Decidir a ordem depois de medir quantas
iterações de uma tarefa real são busca.

---

# Tema 5 — O modelo não sabe quanto contexto gastou

O medidor existe na tela, para a pessoa. O modelo não recebe.

Sem isso ele não pode decidir resumir antes de estourar, nem escolher entre ler o
arquivo inteiro e ler um trecho.

**Cuidado que torna isso não-trivial:** número que muda a cada turno **não pode
entrar no prefixo** — invalidaria o cache em todo turno, que é a ADR-03. Tem que
ir pelo canal de lembretes, anexado, e provavelmente por faixa (`~60%`) e não por
valor exato, para não mudar a cada fragmento.

---

# Tema 6 — Ele não relê o que escreveu

Escreve o arquivo e segue. Uma passada de revisão sobre o próprio diff é barata e
pega muito — foi exatamente o que faltou no exercício, onde a tabela ficou
incompleta e nada olhou de novo.

**A decidir:** doutrina ("antes de terminar, releia o que mudou") ou mecânico (o
loop injeta o diff acumulado antes do último turno). Doutrina é mais barata e
menos confiável; mecânico é o inverso.

---

# Ordem

| # | Tema | Custo | Impacto |
|---|---|---|---|
| 1 | Doutrina manda verificar | baixo | **alto** |
| 2 | Instrução de terceiros | baixo | **alto** |
| 0 | Onde se edita comportamento | médio | alto (destrava o resto) |
| 6 | Reler o próprio diff | baixo | médio |
| 5 | Contexto realimentado | médio | médio |
| 3 | Delegação | alto | alto em tarefa grande |
| 4 | Busca por símbolo | médio | alto em repo grande |

**Faço 1 e 2 juntos, num PR só** — são pequenos, se reforçam e mudam a qualidade
de saída mais que os outros somados. Depois repito o exercício de
auto-hospedagem antes de decidir a ordem do resto: suspeito que boa parte da
distância observada vem desses dois.
