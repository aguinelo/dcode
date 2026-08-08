# Research: Protocolo Client-Server do Harness

> Fonte da verdade de negócio para a camada de transporte entre o núcleo do harness (servidor) e suas superfícies (TUI, desktop, extensão de IDE).
> Decisão de arquitetura de origem: **ADR-04 — Arquitetura de processo**.

## 1. Contexto

O harness é um agente de codificação de terminal. A ADR-04 decidiu **client-server desde o primeiro commit**: o núcleo roda como daemon local e o TUI é apenas um cliente sobre um protocolo estável.

O motivo não é elegância. É que retrofitar client-server em um monolito de TUI é reescrita, e três capacidades dependem disso: aplicativo desktop, extensão de IDE e execução remota. Nenhuma delas cabe em um monolito de terminal.

Este documento define **o que o protocolo precisa garantir**. O contrato técnico está no `.p.spec.md`.

## 2. Fronteira de determinismo

> Seção presente em todo `.r.spec.md` deste projeto. Ver `docs/conventions/SDD-HARNESS.md`.

Esta camada é **inteiramente determinística**. Não há mediação por modelo em nenhum ponto do protocolo: o servidor recebe entrada, emite eventos e responde a decisões de aprovação — tudo por regra explícita.

**Consequência para a revisão:** todo comportamento aqui é verificável por asserção em `go test`. O `.p.spec.md` desta spec **não** tem seção de contratos comportamentais, e usar limiar estatístico neste escopo seria erro de método.

Isso é deliberado. A ADR-01 estabelece que empurrar comportamento para a camada determinística é objetivo de arquitetura, não acidente. O protocolo é a fronteira: **do lado de cá, asserção; do lado de lá, eval.**

## 3. User stories

| # | Como | Quero | Para |
|---|---|---|---|
| US-1 | desenvolvedor | abrir o TUI e trabalhar sem perceber que existe um servidor | o modelo client-server não pode cobrar imposto de usabilidade |
| US-2 | desenvolvedor | fechar o terminal por engano e reabrir na mesma sessão | trabalho longo não pode depender do processo do cliente sobreviver |
| US-3 | desenvolvedor | anexar um segundo cliente à sessão em andamento | acompanhar do editor o que roda no terminal |
| US-4 | desenvolvedor | responder a um pedido de permissão de qualquer cliente anexado | a aprovação não pode ficar presa à janela que iniciou a sessão |
| US-5 | autor de superfície | escrever um cliente novo sem ler o código do núcleo | o protocolo é o contrato, não a implementação |
| US-6 | operador | rodar dezenas de sessões simultâneas | densidade de sessão é a tese do produto (ADR-01) |

## 4. Regras de negócio

### RN-1 — O servidor é dono da sessão; o cliente é descartável
Nenhum estado de sessão vive no cliente. Matar, reiniciar ou trocar o cliente não afeta a sessão. Um turno em andamento **continua** com zero clientes anexados.

### RN-2 — A sessão é um log de eventos append-only
Todo fato observável vira evento numerado, em sequência monotônica, jamais reescrito. Cliente que anexa informa de qual número quer receber; o servidor reproduz o histórico e emenda no fluxo ao vivo.

Isso não é escolha de implementação — é o que faz US-2, US-3 e US-6 caírem da mesma primitiva. Também é o mesmo princípio da **ADR-03**: append-only no contexto do modelo, append-only no log da sessão. Um prefixo imutável é cacheável, reproduzível e testável por golden file.

### RN-3 — Cliente lento nunca bloqueia o agente
O agente não espera entrega. Um cliente que não consome não pode segurar o turno. Como o cliente controla a própria posição de leitura, ele se reposiciona sozinho após qualquer atraso.

### RN-4 — Aprovação é do usuário, não do cliente
Quando a execução cruza a fronteira do sandbox (**ADR-02**), o servidor emite um pedido de aprovação e **bloqueia o turno**. Qualquer cliente anexado pode responder. O primeiro a responder decide; os demais recebem conflito.

### RN-5 — Aprovação falha fechada
Pedido não respondido dentro do prazo é **negado**, nunca concedido. Sem cliente anexado e sem política que permita decisão automática, a operação é negada e o turno segue com a ferramenta rejeitada — não trava indefinidamente.

### RN-6 — O socket é a fronteira de confiança
Na primeira versão o servidor escuta **apenas** em socket de domínio Unix com permissão restrita ao dono. Não há autenticação no protocolo porque não há superfície de rede. Expor em TCP exige autenticação e é mudança de contrato — spec nova, não extensão desta.

### RN-7 — Compatibilidade é promessa versionada
Todo endpoint e todo tipo de evento carrega nível de estabilidade declarado. Quebrar algo marcado como `stable` exige entrada em `changelog/` e incremento de major.

### RN-8 — Um turno por sessão
Uma sessão processa um turno por vez. Entrada nova durante turno em andamento é rejeitada. Paralelismo se obtém com múltiplas sessões, não com múltiplos turnos na mesma — é o que mantém o log de eventos linearmente ordenado e o contexto append-only coerente.

## 5. Fora de escopo

- Autenticação, autorização e transporte remoto (depende de RN-6; spec futura).
- Protocolo de plugin e o ABI correspondente.
- Formato de persistência do log em disco — aqui só o contrato de fio.
- Protocolo entre harness e provider de modelo.

## 6. Changelog

_Sem alterações desde a criação._
