# Doutrina editável por camada

**Data:** 2026-08-10
**Specs afetadas:** `202608080016-behavior-definition` (`.r`, `.p`, `.config`, `.i`)

## O que mudou

A doutrina base deixa de ser inteiramente imutável. Cada uma das quatro seções passa a ter uma **permissão declarada**, e as três permissões são diferentes:

| Seção | Substituir | Acrescentar | Origem permitida |
|---|---|---|---|
| `Identity` | sim | sim | apenas a raiz de configuração do usuário |
| `Style` | sim | sim | apenas a raiz de configuração do usuário |
| `ToolPolicy` | **não** | sim | apenas a raiz de configuração do usuário |
| `Safety` | **nunca** | **nunca** | nenhuma |

O mecanismo é um diretório `doctrine/` sob a raiz de configuração do usuário, com um arquivo por seção:

```
<raiz de config do usuário>/doctrine/
  identity.md   → substitui Doctrine.Identity
  style.md      → substitui Doctrine.Style
  tools.md      → ACRESCENTA a Doctrine.ToolPolicy
```

Qual arquivo existe determina qual seção muda; **como** ela muda é fixo por seção e não é configurável. Não há arquivo que substitua `ToolPolicy` e não há arquivo que alcance `Safety`.

`DCODE_DOCTRINE_STYLE` **sai**. Estava declarada na seção 4 do `.config` desde a criação da spec e nunca foi implementada. Um bloco de prosa de várias linhas não é valor de variável de ambiente, e manter duas formas de ajustar a mesma seção cria a dúvida de qual venceu — que é exatamente o problema que esta mudança existe para resolver.

## Por que mudou

Hoje `DefaultDoctrine` devolve quatro constantes e `internal/app/app.go` a consome sem parâmetro. Não há caminho de configuração até nenhuma das quatro. Quem quer que o agente responda em outro idioma, ou seja menos verboso, só pode **acrescentar texto ao fim do prompt**, pela seção de instruções de projeto.

Isso é contradição, não substituição. A doutrina continua dizendo o que dizia, a instrução diz o contrário, e o modelo escolhe — às vezes uma, às vezes outra, sem que o usuário saiba qual foi. E a instrução do usuário entra pela mesma porta que a instrução que veio junto do repositório clonado, o que é o assunto de outra mudança.

A trava de `Safety` existe por um motivo real e específico: se um arquivo pudesse reescrevê-la, um repositório clonado da internet poderia desligar a exigência de aprovação por texto. A RN-10 está certa. **O defeito é que essa razão foi aplicada às outras três seções, que não a têm.** Ninguém é atacado pela reescrita do próprio estilo de saída.

A pergunta que separa corretamente as seções não é *o que pode mudar* — é **quem pode mudar**. E o repositório que o usuário clonou não é o usuário. Essa distinção já existe no produto do lado da configuração: `internal/config/config.go` conhece a raiz do usuário, e `toml.go` conhece a raiz do projeto, como origens distintas com pesos distintos. A doutrina apenas ainda não a usava.

### Por que `ToolPolicy` acrescenta e não substitui

`ToolPolicy` descreve máquina que existe de fato. Substituí-la permite declarar ferramenta inexistente ou omitir ferramenta real, e o resultado é o modelo chamando o que não há — falha que se manifesta longe da causa e parece defeito do provider.

Mas é também onde mora "prefira `rg` a `grep`" e "rode a verificação antes de dizer que terminou", que são preferência legítima de quem usa. Acrescentar preserva as duas coisas: a lista real de ferramentas é inegociável, o hábito em torno delas não.

### Por que `Safety` não aceita nem acrescentar

Acrescentar ao fim de `Safety` é funcionalmente substituir: o texto acrescentado pode dizer "ignore o acima". Uma seção que aceita apêndice não tem trava — tem uma trava que se contorna escrevendo mais um parágrafo.

## Como a trava é feita

Não por condicional. `Safety` **não é campo do tipo de sobreposição**:

```go
// DoctrineOverlay é o que a configuração do usuário pode mudar na camada base.
// Campo ausente deixa o texto embarcado intacto.
//
// Safety não está aqui, e é essa ausência que é a garantia: não existe caminho
// para fechar, porque não existe caminho.
type DoctrineOverlay struct {
    Identity  string // substitui
    Style     string // substitui
    ToolsMore string // ACRESCENTA a ToolPolicy; nunca substitui
}
```

`ToolsMore` tem nome diferente de `ToolPolicy` pelo mesmo motivo: não existe atribuição acidental que troque uma pela outra.

Isso segue a RN-2 da própria spec — regra que vira invariante de código sai do campo do texto e passa a ser verificável por asserção. Uma trava por convenção quebra no primeiro refactor; uma trava por tipo não compila.

O carregador recebe **um** diretório, não uma lista:

```go
func LoadDoctrineOverlay(dir string, maxBytes int) (DoctrineOverlay, []Notice, error)
```

O contraste com `LoadSkills(dirs []string, ...)` é deliberado. Skill vem de duas raízes — a do usuário e a do projeto. Sobreposição de doutrina vem de uma. O tipo singular diz isso melhor que qualquer comentário, e a raiz do workspace nunca chega a ser argumento.

## Visibilidade

`DCODE_DOCTRINE_DUMP` já existe e passa a **marcar a origem de cada seção** — embarcada, substituída ou acrescentada. Substituição invisível seria pior que a imutabilidade atual: o usuário perderia a única forma que tem hoje de saber o que foi ao modelo.

Truncamento por exceder o teto **avisa**, pelo mesmo motivo já registrado para arquivo de instrução: truncar em silêncio faz o usuário acreditar que uma regra está valendo quando não está.

Arquivo cujo nome não corresponde a nenhuma seção permitida — inclusive `safety.md` — é ignorado **e registrado**, nunca descartado em silêncio. É o mesmo tratamento que a RN-10 já exige para instrução que tenta afrouxar segurança, e pela mesma razão: tentativa invisível é tentativa que ninguém investiga.

## Alternativas descartadas

**Doutrina inteira num arquivo só.** Uma seção some por engano ao editar e o produto passa a rodar sem política de ferramenta, sem erro. Um arquivo por seção torna a ausência explícita.

**Marcador dentro do arquivo escolhendo substituir ou acrescentar.** Empurra para o usuário uma decisão que é do produto, e a resposta errada em `ToolPolicy` produz o modo de falha que esta mudança evita.

**Permitir sobreposição também pela raiz do projeto.** É o vetor exato que a RN-10 fecha, apenas por outra porta. Um `.dcode/doctrine/identity.md` num repositório clonado redefine quem o agente pensa que é, antes de qualquer instrução ser lida.

**Chave de configuração por seção, em vez de arquivo.** A superfície de configuração é bijetiva entre chave e variável de ambiente, e o conteúdo aqui é documento de várias linhas, não valor. Foi o que fez `DCODE_DOCTRINE_STYLE` nunca sair do papel.

## Impacto

- Novo tipo `DoctrineOverlay` e método `Doctrine.Apply` em `internal/behavior`.
- Novo carregador `LoadDoctrineOverlay`, com o mesmo desenho de teto e aviso de `LoadSkills`.
- `internal/app` passa a resolver a sobreposição na criação da sessão — **uma vez**, como manda a RN-5. Arquivo de doutrina escrito no meio da sessão não altera o prefixo, pelo mesmo motivo que instrução tardia não altera.
- `DCODE_DOCTRINE_STYLE` removida da seção 4 do `.config`; três chaves novas no lugar.
- `Doctrine.Safety` continua constante não configurável na seção 5 do `.config`, agora com o mecanismo dito por extenso.
- Nenhuma mudança no protocolo, no sandbox ou na avaliação de política.
