# Research: Adaptador de Provider

> Fonte da verdade de negócio para a camada entre o dcode e os modelos de linguagem.
> Decisão de arquitetura de origem: **ADR-05 — Agnóstico de provider, com camada de adaptação real**.

## 1. Contexto

A ADR-05 decidiu neutralidade de provider **com camada de adaptação por família de modelo** — não simples troca de endpoint.

O motivo está registrado no estudo comparativo: harnesses agnósticos perdem para os afinados rodando o mesmo modelo, porque system prompt, schema de ferramenta e estratégia de edição precisam ser adaptados por família. Trocar só a URL entrega qualidade inconsistente de tool-calling, que é a falha mais cara de um agente de codificação: uma ferramenta mal invocada corrompe arquivo.

Este componente é onde essa adaptação vive. No MVP existe **uma** implementação; a interface existe desde o dia 1 para que a segunda não exija reescrita.

## 2. Fronteira de determinismo

**Regime: misto.**

| Parte | Regime | Verificação |
|---|---|---|
| Tradução de `[]Message` para o formato de fio | determinístico | asserção e golden file |
| Parsing do stream de resposta em eventos | determinístico | asserção sobre transcript gravado |
| Mapeamento de erro do provedor | determinístico | asserção |
| **Fidelidade de tool-calling da família de modelo** | **mediado por modelo** | limiar sobre fixtures |

A linha é nítida: **dada uma resposta, o adaptador é totalmente determinístico.** O que não é determinístico é *se a família emite tool call bem formada contra o nosso schema* — e isso é propriedade do modelo, não do código.

O `.p.spec.md` tem seção de contratos comportamentais cobrindo apenas a quarta linha.

## 3. User stories

| # | Como | Quero | Para |
|---|---|---|---|
| US-1 | usuário | usar o modelo que já pago | não trocar de assinatura para experimentar o harness |
| US-2 | usuário | ver a resposta aparecendo enquanto é gerada | turno longo não parecer travado |
| US-3 | desenvolvedor do dcode | rodar todo o teste sem tocar a rede | CI determinística, rápida e sem custo |
| US-4 | desenvolvedor do dcode | adicionar uma família nova sem tocar o loop | a adaptação ficar contida |
| US-5 | usuário | entender por que uma chamada falhou | distinguir cota, rede, credencial e erro de schema |

## 4. Regras de negócio

### RN-1 — Adaptação é por família, não por endpoint
Uma família de modelo é o conjunto que compartilha formato de tool call, comportamento de streaming e convenções de prompt. Trocar `base_url` não cria família nova; trocar o formato de tool call, sim.

Cada família tem seu adaptador. Um adaptador que precise de `if modelo == X` interno é sinal de que são duas famílias.

### RN-2 — O núcleo não conhece provedores
O loop e o motor de contexto trabalham com os tipos neutros do dcode. Nenhum tipo específico de provedor vaza para fora deste pacote. Se vazar, a ADR-05 já foi perdida.

### RN-3 — Streaming é obrigatório
Toda chamada é em streaming. US-2 não é ajuste de interface: sem streaming não há como interromper um turno no meio, que é requisito do loop.

### RN-4 — Todo teste roda sem rede
O comportamento é verificado contra transcript gravado. Chamada real fica atrás de build tag, nunca na suíte padrão.

Isso não é conveniência de CI: transcript gravado é o que torna o parsing testável de forma determinística e é a base do gate de cobertura.

### RN-5 — Erro é classificado, não repassado cru
O loop precisa decidir entre repetir, esperar, trocar de abordagem ou desistir. Uma string de erro não permite essa decisão. Todo erro de provedor é traduzido para uma classe estável do harness.

Isto sustenta diretamente a recuperação de erro definida no loop do agente.

### RN-6 — Credencial nunca aparece em log, evento ou erro
Chave de API não é registrada, não entra em mensagem de erro, não vai para o log de eventos. Violação disto é blocker imediato.

### RN-7 — A janela do modelo é dado do adaptador
O motor de contexto dispara compactação em fração da janela, mas não conhece modelos. Quem informa o tamanho da janela é o adaptador.

### RN-8 — Tool call malformada é erro de turno, nunca execução
Se a resposta contém tool call que não valida contra o schema declarado, ela **não** é executada. Vira resultado de erro devolvido ao modelo, que reanalisa.

Executar tool call malformada com input adivinhado é como um agente corrompe arquivo. Não há caso em que valha o risco.

## 5. Fora de escopo

- Roteamento entre múltiplos provedores por custo ou capacidade.
- Cache de resposta do lado do dcode — o cache que importa é o de prefixo, do lado do provedor.
- Fine-tuning, embeddings e qualquer chamada que não seja completude de chat.
- Contabilidade de custo em dinheiro.

## 6. Changelog

_Sem alterações desde a criação._
