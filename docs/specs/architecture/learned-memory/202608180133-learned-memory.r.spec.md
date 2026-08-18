# Research: Memória aprendida

> Fonte da verdade de negócio para o que o agente descobre e precisa lembrar
> entre sessões.

## 1. Contexto

Uma sessão termina e tudo que ela descobriu morre com ela. Na sessão seguinte o
agente redescobre que `make test` falha sem `go generate` antes, que o linter
reclama de uma coisa que ninguém documentou, que a decisão de manter `-race` na
CI já foi tomada e discutida.

A gravação de sessão (`internal/session/record.go`) preserva **o que aconteceu**
e responde "o que essa sessão fez". Não responde "o que este repositório ensina",
e não deveria: são perguntas diferentes com prazos de validade diferentes.

A cadeia de instruções (`AGENTS.md`, `DCODE.md`) preserva **o que alguém
decidiu**. Também não responde: quem escreve ali é uma pessoa, deliberadamente, e
o que o agente descobre no meio de um turno não passou por essa deliberação.

Falta a terceira: **o que foi descoberto e vale guardar**.

### Por que não copiar a solução conhecida

O desenho de referência nesse espaço monta um wiki paralelo: daemon próprio,
captura automática por hooks de ciclo de vida, consulta por MCP, consolidador
LLM ao fim de cada sessão, decaimento por desuso.

Três coisas nesse desenho colidem com decisões já tomadas aqui:

- **MCP** ficou fora por ter superfície grande com ciclo de vida e modos de falha
  próprios.
- **Hooks de projeto** ficaram fora porque é como configuração vira mato — e o
  próprio autor daquele sistema relata **seis semanas de perda silenciosa de
  dados** por uma chave de hook errada, que é exatamente o que acontece quando o
  produto depende de um acoplamento que nenhum dos dois lados verifica.
- **Memória fora do repositório** é memória que o time não lê e o code review não
  vê. O autor a descreve como "LLM-optimized, not human-curated".

A parte que **vale copiar** é a disciplina técnica: armazenamento simples, índice
simples, sem banco vetorial. Isso já é a doutrina deste projeto.

## 2. Fronteira de determinismo

**Regime: misto.**

- **Determinístico:** onde a memória mora, como é lida, como entra no prefixo,
  qual autoridade tem, como é limitada, e como obsolescência é detectada. Tudo
  isso é asserção comum.
- **Mediado por modelo:** *se* o agente escreve uma memória, e se o que ele
  escreve vale a pena. Isso é contrato comportamental e vive na suíte de eval.

**Consequência para a revisão:** a metade determinística é a que carrega as
garantias. A metade mediada não pode enfraquecer nenhuma delas — um modelo que
nunca chama `remember` produz um produto que funciona exatamente como antes.

## 3. User stories

**Como pessoa que volta ao projeto depois de duas semanas**, quero que o agente
já saiba o que a sessão anterior descobriu, para não pagar de novo o custo de
descobrir.

**Como pessoa revisando um PR**, quero ver no diff o que o agente aprendeu, para
poder discordar antes que aquilo vire base de decisão.

**Como pessoa que escreveu as regras do projeto**, quero que nada que o agente
ensinou a si mesmo passe por cima do que eu escrevi.

**Como pessoa que herda o repositório**, quero ler a memória sem ferramenta
nenhuma — é markdown no git, como o resto.

## 4. Regras de negócio

**RN-1. A memória mora no workspace e é versionada pelo usuário.**
Não em diretório de estado, não em banco, não em serviço. A revisão é o portão de
qualidade, e revisão acontece no diff.

**RN-2. Nada aprendido vence nada escrito por pessoa.**
A fonte aprendida ordena **abaixo** de `user`, que é a mais fraca das fontes
humanas hoje. Não há configuração que inverta. Sem isso a memória vira o caminho
pelo qual o agente reescreve devagar as próprias restrições.

**RN-3. A procedência é visível no prefixo.**
O modelo tem de conseguir distinguir o que uma pessoa exigiu do que ele próprio
anotou. Instrução sem procedência é instrução que ninguém pode contestar.

**RN-4. Memória é tipada, e a lista de tipos é fechada.**
`gotcha`, `decision`, `convention`. Memória sem tipo vira diário, e diário vira
ruído em duas semanas.

**RN-5. Registro de atividade não é memória.**
"O que fiz na sessão passada" é trabalho da gravação. Uma memória que registra
atividade em vez de conhecimento é a forma como esse mecanismo se envenena.

**RN-6. Escrever memória é ato explícito do agente.**
Sem consolidador ao fim da sessão, sem chamada de modelo que ninguém pediu, sem
processo rodando depois que a pessoa parou de olhar. O que é escrito aparece na
transcrição enquanto acontece.

**RN-7. Toda memória carrega quando foi aprendida e em que commit era verdade.**
É o que torna obsolescência **conferível** em vez de adivinhada.

**RN-8. Obsolescência é reportada, nunca aplicada.**
Uma memória sobre arquivo que não existe mais é suspeita, não lixo. Apagar por
heurística é como uma regra real desaparece sem ninguém notar.

**RN-9. A memória é limitada, e o corte é declarado.**
Nada neste código corta saída em silêncio.

**RN-10. Não há decaimento por acesso.**
Frequência mede o que o agente por acaso precisou, não o que é verdade. A gotcha
que dispara uma vez por ano é justamente a que vale guardar.

**RN-11. Um produto sem nenhuma memória escrita é o produto de hoje.**
Arquivo ausente é o caso comum e é silencioso. Nenhum caminho novo pode falhar
por não haver memória.

## 5. Fora de escopo

**Memória de usuário, atravessando repositórios.** A cadeia de configuração já
suportaria, e é justamente onde uma lição errada faz mais estrago: uma gotcha de
um projeto aplicada a outro é pior que nenhuma. Fica para depois de haver
evidência de uso no escopo de projeto.

**Busca, índice, embeddings.** Enquanto a memória couber no prefixo, procurar
nela é resolver um problema que não existe. Se um dia não couber, o limite de
RN-9 é o sinal, e aí é decisão nova.

**Consolidação automática por LLM.** RN-6 fecha isso por ora. Se a medição
mostrar que o modelo raramente chama `remember`, a resposta é lembrete de Camada
2, não um segundo modelo.

**Compartilhar memória entre máquinas.** É git. Já está compartilhada.

## 6. Changelog

- [202608180133 — Memória aprendida](changelog/202608180133-memoria-aprendida.md)
