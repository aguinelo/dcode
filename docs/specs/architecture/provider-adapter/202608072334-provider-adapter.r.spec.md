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
| US-6 | desenvolvedor do dcode | reusar um transporte já pronto ao adicionar família | modelo novo em fio conhecido não custar reimplementação |
| US-7 | usuário | escolher o dialeto quando meu modelo fala mais de um | contornar bug de um lado do provedor sem trocar de modelo |
| US-5 | usuário | entender por que uma chamada falhou | distinguir cota, rede, credencial e erro de schema |

## 4. Regras de negócio

### RN-1 — Transporte e família são eixos ortogonais
São duas coisas diferentes, e confundi-las é o erro que faz um harness entregar tool-calling não medido fingindo que foi validado.

| | Transporte | Família |
|---|---|---|
| O que é | formato de fio da requisição | prompt, schema de tool, estratégia de edição |
| Exemplos | `openai`, `anthropic` | `minimax-m3`, `claude` |
| Reusável entre? | sim, entre famílias | não |
| Carrega limiar de contrato comportamental? | não | **sim** |
| Define limites padrão do turno? | não | **sim** (RN-9) |

**"OpenAI-compatible" é transporte, nunca família.** Um modelo desconhecido atrás de endpoint OpenAI-compatible herda apenas o formato de fio — jamais a adaptação nem os limiares medidos para outro modelo.

O caso que prova a necessidade dos dois eixos é o MiniMax M3, que fala **os dois dialetos**, Anthropic-compatible e OpenAI-compatible. Com um eixo só, suportar ambos exigiria duplicar a família inteira.

Uma família declara os transportes com que é compatível, em ordem de preferência. Um adaptador que precise de `if modelo == X` interno é sinal de que são duas famílias.

### RN-11 — Família sem medição diz que não tem

Nome de família, neste repositório, lê como família **medida**. O `Measurement.Model` existe exatamente porque limiar pertence a um modelo e não fala nada de outro, então acrescentar uma família e não dizer nada põe um nome onde a verificação não está.

Uma família sem nenhuma medição registrada **avisa isso na sessão**. O aviso nomeia a família, diz que o formato de fio e a janela são conhecidos, e diz contra o que os limiares *foram* medidos — que é o fato que torna a ressalva legível. É mais estreito que o aviso da `generic`, onde o endpoint em si é desconhecido e a janela e as imagens também são chute.

A lista de quem avisa **não é digitada**: uma guarda a confere contra as medições que existem, nos dois sentidos. Família que ganha medição e continua avisando reprova; família que sai da lista sem medição reprova também.

Isso foi escrito quando a `gemini` entrou — e a guarda, na primeira execução, achou que a `claude` estava nessa condição desde sempre, calada.

### RN-9 — Limites padrão do turno são da família
Modelo treinado para horizonte longo precisa de teto de iteração maior que modelo afinado para tarefas curtas. Um número global serve mal aos dois.

Cada família declara seus limites padrão; a configuração do usuário sobrescreve. O loop consome esse default em vez de carregar um número fixo.

Sem isso, o teto dimensionado para um modelo trunca trabalho legítimo de outro — e teto que trunca trabalho legítimo vira teto que o usuário desliga.

### RN-2 — O núcleo não conhece provedores
O loop e o motor de contexto trabalham com os tipos neutros do dcode. Nenhum tipo específico de provedor — nem de transporte, nem de família — vaza para fora deste pacote. Se vazar, a ADR-05 já foi perdida.

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

**Uma call parcial é malformada.** Frame não é unidade de significado: os argumentos chegam partidos, e uma call emitida antes do fim do stream chega à ferramenta sem input. Isso não é erro do modelo, e tratá-lo como erro do modelo esconde o defeito no adaptador — o sintoma é a ferramenta respondendo *"campo obrigatório ausente"* para toda chamada.

### RN-10 — Raciocínio não é resposta
O pensamento do modelo viaja em canal próprio e **nunca** entra no histórico.

Um modelo que lê o próprio raciocínio de volta como algo que disse em voz alta passa a defendê-lo em vez de reconsiderar. E o texto seria pago em todo turno seguinte da sessão, o que contraria a ADR-03 diretamente.

Família que manda o raciocínio duplicado no campo de conteúdo é o caso comum, não a exceção: quem monta a resposta precisa distinguir os dois, e na dúvida o texto é raciocínio.

## 5. Fora de escopo

- Roteamento entre múltiplos provedores por custo ou capacidade.
- Cache de resposta do lado do dcode — o cache que importa é o de prefixo, do lado do provedor.
- Fine-tuning, embeddings e qualquer chamada que não seja completude de chat.
- Contabilidade de custo em dinheiro.

## 6. Changelog

- [202608072352 — Transporte e família como eixos ortogonais](changelog/202608072352-transporte-familia-ortogonais.md)
- [202608082230 — Decode passa a ter estado por stream](changelog/202608082230-decode-com-estado-por-stream.md) — nova RN-10.
