# Ferramenta `plan`

**Data:** 2026-08-08
**Specs afetadas:** `202608072337-tool-suite` (`.r`, `.p`, `.i`), `202608072240-client-server-protocol` (`.p`), `202608080016-behavior-definition` (`.r`, `.p`)

## O que mudou

O conjunto mínimo passa de seis para **sete ferramentas**: `read`, `write`, `edit`, `glob`, `grep`, `bash` e agora **`plan`**.

`plan` cria e atualiza a lista de execução da sessão. Não toca o sistema de arquivos e não executa nada — só muda estado de sessão.

## Por que mudou

O planejamento passa a ser **comportamento intrínseco** do produto: toda tarefa tem plano, com profundidade proporcional ao tamanho dela. O plano fica permanentemente visível no cliente.

Isso exige que o plano seja **dado estruturado**, não prosa no meio da resposta. Um cliente que tivesse que interpretar texto para montar o painel obrigaria todo cliente futuro — desktop, IDE — a reimplementar a mesma heurística, e as implementações divergiriam.

Ferramenta é o mecanismo certo porque já existe: passa pelo avaliador de política como qualquer outra, produz evento naturalmente, e o estado vive na sessão, do lado do servidor, onde a ADR-04 manda.

## Alternativas descartadas

**Campo dedicado na resposta do modelo.** Exigiria suporte no adaptador de cada família e no protocolo de cada transporte. `plan` como ferramenta funciona em qualquer família que saiba chamar ferramenta — que é o requisito mínimo do produto de qualquer forma.

**Plano derivado do texto por heurística.** Frágil, não portável entre clientes, e sem forma de saber quando um item foi concluído.

## Impacto

- Sétima entrada no conjunto de ferramentas; nenhuma outra muda.
- Novo evento `plan.updated` no protocolo.
- Estado de plano na sessão, do lado do servidor.
- `plan` nunca cruza fronteira de sandbox: `Declare` não reporta caminho nem rede, e o veredito é sempre `allow`.
- **Aprovar plano não é aprovar fronteira.** Continuam sendo duas altitudes distintas — ver RN-7 de `202608081250-client-tui`.
