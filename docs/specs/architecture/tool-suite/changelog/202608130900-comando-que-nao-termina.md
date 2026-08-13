# Um comando que não termina

**Data:** 2026-08-13
**Specs afetadas:** `202608072337-tool-suite` (`.r`, `.p`)
**Origem:** uso real — "roda esse projeto", `npm start`, a sessão travou

> **Regra:** comando que não termina não pode ser esperado; e o que se inicia fora do turno pertence à sessão, que é quem o encerra.

## O problema

Pedido para subir o servidor da aplicação. O modelo chamou `bash npm start` — a coisa certa a chamar — e a sessão parou de responder até o comando ser cortado.

Não é defeito de implementação. É o desenho encontrando um caso que ele não tem:

```go
runCtx, cancel := context.WithTimeout(ctx, timeout)   // 120s
out, code, err := b.Runner.Run(runCtx, b.Workdir, in.Command)
```

`Run` é síncrono. Servidor não termina — é essa a definição de servidor — então o turno fica preso até o teto. **O teto não é o problema; é o único motivo pelo qual a sessão volta.**

## Por que não é só uma flag

Um `background: true` resolve a chamada e cria quatro problemas que o produto não tinha. Três já foram decididos em `docs/backlog/202608130300`, e a decisão de não virar ADE é o que os fecha: **nada sobrevive ao que criou**. Sem sessão reencontrável, processo sobrevivente não teria a quem responder.

**Quem é dono.** O processo morre com a sessão. A tabela vive no estado de sessão, e estado que acaba leva os processos junto — posse, não faxina. Não existe handler para alguém esquecer de registrar.

**O que a política diz.** Aprovar comando longo é o mesmo ato que aprovar comando. É defensável exatamente porque o processo não sobrevive à sessão: autorização e processo têm o mesmo tempo de vida, então não existe janela em que alguém consentiu com um instante e concedeu uma era.

**Onde a saída vai.** O modelo consulta; não recebe. Receber exigiria um caminho do servidor para dentro do turno que o protocolo não tem, e servidor tagarela comeria a janela sem ninguém ter pedido.

**Como o modelo sabe que subiu.** Resolvido aqui, e é a parte que estava em aberto.

## A décima ferramenta, e o critério

A RN-1 fixa um conjunto mínimo, e acrescentar exige justificativa. Para `symbol` foi a distinção entre definição e uso. Aqui **não é contagem de ferramenta — é veredito de política.**

`bash` declara rede e escrita no workspace, porque comando de shell é opaco e o pior caso é o que se declara. Ler um buffer que este processo já possui não cruza fronteira nenhuma.

Juntas numa ferramenta só, `Declare` teria de ramificar por modo — e toda leitura de log enfileiraria aprovação para uma fronteira que não é tocada. **Pergunta feita sem motivo é como se aprende a responder sem ler**, e é o oposto do que a ADR-02 chama de consentimento.

```go
type ProcessInput struct {
    ID   string `json:"id,omitempty"`   // sem ID, lista
    Stop bool   `json:"stop,omitempty"`
}
```

`stop` está aqui e não em outra ferramenta porque tem o mesmo veredito: encerrar um processo que esta sessão iniciou não cruza fronteira alguma. E é necessário — quem subiu o servidor errado precisa da porta de volta antes de subir o certo.

## "Subiu" é pergunta diferente de "terminou bem"

Processo em segundo plano não tem código de saída até morrer. As duas respostas óbvias foram descartadas:

- **Sondar porta** exige uma porta que ninguém nomeou. `npm start` não diz qual é.
- **Ler prontidão no log** exige uma convenção que dois programas não compartilham.

O que se faz é observar uma **janela de assentamento** curta e reportar o que houver. Isso responde a pergunta que de fato morde — *quebrou no boot?* — que é o caso comum: porta ocupada, dependência faltando, variável de ambiente ausente. Morreu na janela, o resultado informa a saída em vez de "iniciado", porque relatar sucesso que o comando não teve é a ferramenta mentindo.

O resto é `process`, quando o modelo quiser.

## O órfão, que era o risco de verdade

Duas coisas no caminho existente iam contra a decisão de tempo de vida, e as duas são silenciosas:

**`Wrap` monta com `CommandContext(ctx)`.** Passando o contexto do turno, o processo morre quando o turno acaba — a flag existiria e não serviria para nada. O comando é montado com contexto próprio, e quem o encerra é a sessão.

**Ninguém definia grupo de processo.** `npm start` é um shell que executa npm que gera node. Matar o invólucro deixa vivo quem segura a porta, sem nome e sem ninguém que o alcance. O comando passa a nascer em grupo próprio e o encerramento alcança o grupo.

E `dcode "..."` de tiro único não encerrava nada: nada no unix mata o filho quando o pai sai — ele é reparentado e continua. Era o caminho mais curto para estranhar um servidor na máquina de alguém.

> Órfão é pior que sessão travada, **porque travar é visível**.

## Saída: mantém o fim, não o começo

O teto de saída vale como em toda ferramenta (RN-5), mas o corte é pelo outro lado. As primeiras linhas de um servidor são banner; as últimas são por que ele morreu. Cortar o fim joga fora a única parte que se lê um log para ver.

## Configuração

Nenhuma chave nova. A janela de assentamento é campo da ferramenta, com default no código.

Deliberado: chave declarada e não lida é promessa de controle que não existe — a lição do `202608110900`. Quando alguém precisar ajustar a janela, a chave entra com o código que a lê.

## Invariantes

- `bash` com `background` devolve identificador enquanto o comando ainda roda.
- Comando que morre na janela de assentamento reporta a saída, nunca "iniciado".
- Identificador é sequência (`bg1`, `bg2`), nunca relógio — mesma razão da RN-7.
- `process` sem identificador lista; identificador desconhecido nomeia os que existem.
- `process.Declare` não declara caminho nem rede; veredito sempre `allow`.
- `background` declara exatamente o que a chamada de primeiro plano declara.
- `background` sem executor capaz é recusado, nunca rebaixado a primeiro plano.
- Fechar o estado da sessão encerra todo processo iniciado nela.
- Encerrar alcança o grupo — verificado com neto que segura o recurso.
- Comando em segundo plano sobrevive ao turno que o iniciou.
- Saída acima do teto mantém o fim, e declara o corte.

## Impacto

- Décima ferramenta; nenhuma outra muda de comportamento.
- `bash` ganha um campo e duas frases de descrição, dizendo **quando** usá-lo.
- Cadeia de posse explícita: sessão → engine → estado de ferramentas → processos.
- Comandos passam a nascer em grupo próprio, o que também torna o corte por teto no primeiro plano mais completo do que era.

## O que continua fora

- **Contrato de eval.** *"Sobe um servidor e relata o endereço, sem travar"* é comportamento, não asserção, e não existe hoje. Entra quando a suíte em andamento fechar — acrescentar cenário a uma medição rodando invalida a própria medição.
- **Sinal a processo alheio.** Continua negado pelo Seatbelt, e continua certo: nada fica órfão, então o modelo não precisa de autoridade de sinal sobre a máquina.
- **Receber saída sem pedir.** Exigiria caminho do servidor para dentro do turno que o protocolo não tem.
