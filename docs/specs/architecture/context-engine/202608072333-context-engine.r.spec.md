# Research: Motor de Contexto

> Fonte da verdade de negócio para a montagem do contexto enviado ao modelo.
> Decisão de arquitetura de origem: **ADR-03 — Contexto append-only**.

## 1. Contexto

Toda chamada ao modelo carrega um prefixo — instruções de sistema, definições de ferramenta e o histórico da conversa. Esse prefixo é a maior fatia do custo de cada turno, e provedores cobram muito menos por prefixo que já está em cache.

A ADR-03 decidiu **append-only**: o prefixo nunca é mutado entre turnos. Qualquer edição no início invalida o cache KV inteiro e recobra o prompt completo, em latência e em dinheiro.

Este é o componente onde essa decisão vive ou morre. Se ele estiver certo, o resto do sistema herda a propriedade de graça. Se estiver errado, nenhuma otimização em outro lugar compensa.

## 2. Fronteira de determinismo

**Regime: determinístico.**

A montagem do contexto é função pura do estado da sessão. Não há mediação por modelo: dado o mesmo histórico, a saída é byte-a-byte idêntica.

**Consequência para a revisão:** todo comportamento aqui é verificável por asserção e por golden file. O `.p.spec.md` não tem seção de contratos comportamentais.

Isso é o caso exemplar do objetivo descrito em `docs/conventions/SDD-HARNESS.pt-BR.md`: comportamento que *poderia* ser deixado para o modelo resolver foi deliberadamente empurrado para o lado determinístico, onde é exato e barato de testar.

## 3. User stories

| # | Como | Quero | Para |
|---|---|---|---|
| US-1 | usuário | pagar preço de cache na maioria dos turnos | custo e latência de sessão longa serem viáveis |
| US-2 | usuário | continuar uma tarefa longa sem perder o fio | a janela de contexto não é o limite da tarefa |
| US-3 | desenvolvedor do harness | reproduzir exatamente o contexto de um turno passado | depurar comportamento do agente sem adivinhação |
| US-4 | desenvolvedor do harness | adicionar uma ferramenta sem invalidar cache de sessão viva | extensibilidade não pode custar performance |

## 4. Regras de negócio

### RN-1 — O prefixo é imutável
Uma vez enviada ao modelo, nenhuma mensagem é editada, reordenada ou removida. Correção se faz **anexando**, nunca reescrevendo.

Isto é a regra de ouro do componente. Toda outra regra aqui existe para sustentá-la.

### RN-2 — Nada volátil no prefixo
Não entram no prefixo: timestamp, contador de tokens restantes, número de iteração, estado de conexão, ou qualquer valor que mude entre turnos sem que o usuário tenha feito algo.

Um único timestamp no system prompt invalida o cache em todos os turnos e anula o componente inteiro.

### RN-3 — Definições de ferramenta são fixadas no início da sessão
O conjunto de ferramentas é resolvido na criação da sessão, a partir de cache, e não muda enquanto ela viver. Servidor de ferramenta externo que conecte depois **não** injeta definição na sessão em andamento.

Sem isso, US-4 quebra US-1: cada conexão tardia custaria o prefixo inteiro de todas as sessões vivas.

### RN-4 — Ordem de montagem é fixa
As seções do contexto sempre aparecem na mesma ordem, do mais estável para o mais volátil. Parte estável primeiro é o que permite ao provedor casar o prefixo mais longo possível com o cache.

### RN-5 — Compactação é rara, em bloco e explícita
Quando o contexto se aproxima do limite da janela, um trecho contíguo do histórico antigo é substituído por **um** resumo, num único ponto de corte.

Compactar um pouco a cada turno é o pior comportamento possível: invalida o cache sempre e ainda perde informação continuamente.

A compactação é o **único** momento em que o prefixo muda, e por isso é evento observável — quem acompanha a sessão precisa saber que aconteceu.

### RN-6 — Compactação preserva a tarefa corrente
Um resumo que perde o que o usuário pediu torna o agente inútil no exato momento em que a tarefa ficou longa o bastante para importar. A tarefa em andamento e as decisões já tomadas sobrevivem ao corte; detalhe de exploração descartada, não.

### RN-7 — O contexto é reconstruível
Dado o histórico da sessão, a montagem produz o mesmo resultado em qualquer momento e em qualquer máquina. Não há estado escondido, nem dependência de relógio, nem de ordem de map.

É o que sustenta US-3 e o que permite golden test exato.

## 5. Fora de escopo

- Recuperação semântica e memória de longo prazo entre sessões.
- Persistência do histórico em disco — aqui só a montagem em memória.
- Formato de fio de cada provedor: responsabilidade do adaptador de provider.
- Decisão de *quando* chamar o modelo: responsabilidade do loop do agente.

## 6. Changelog

_Sem alterações desde a criação._
