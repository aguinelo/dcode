# O fluxo que encerrava o cliente

**2026-09-01** — RN-20: fim de fluxo substituído não encerra o `dcode`.

## O sintoma

`/loop oi` fechava o programa.

## O mecanismo

`waitForEvent` captura os canais **quando o comando é construído**:

```go
func (p *program) waitForEvent() tea.Cmd {
	events, errs := p.events, p.errs
	return func() tea.Msg { ... }
}
```

Trocar de sessão chama `attach`, que cancela a assinatura anterior. O leitor que
assistia à sessão velha **continua lendo os canais velhos**, vê o fechamento, e
devolve `streamClosedMsg` — que o `Update` responde com `tea.Quit`.

## Isto nunca foi sobre o `/loop`

`/clear`, `/model` e `/resume` anexam pelo mesmo caminho e teriam o mesmo fim.
O defeito estava lá desde que trocar de sessão existe.

O que mudou foi só a **alcançabilidade**: `/loop <palavra>` falhava no
`CreateSession`, devolvia uma nota, e nunca chegava a trocar de sessão. Corrigido
isso, o comando passou a ter sucesso — e sucesso era o caminho que levava ao
defeito.

Vale registrar do jeito certo: a correção anterior não introduziu o problema, ela
**tornou o problema alcançável**. As duas afirmações têm consertos diferentes, e
tratar a primeira como verdadeira teria levado a reverter a correção certa.

## A regra

Cada assinatura recebe um número, e as três mensagens que ela produz — evento,
erro e fim — o carregam.

A conferência acontece no `Update`, que é de uma linha só de execução. Ler o
número corrente de dentro do comando seria **corrida de dados** com o laço que o
escreve, e a suíte roda sob `-race` justamente para não deixar isso passar.

Mensagem de fluxo substituído é descartada e **não rearma** o leitor: quem anexou
já iniciou o leitor do fluxo novo, e rearmar poria dois leitores nos mesmos
canais.

Fim do fluxo **corrente** continua encerrando. A regra é sobre qual fluxo, não
sobre nunca encerrar.

## Geração zero

Mensagem construída antes de qualquer `attach` — ou por teste que não se importa
— lê como corrente, porque não há contra o que ela seja obsoleta. O `errMsg` do
caminho de `steer` também é geração zero, e de propósito: ele é aquela chamada
falhando, não um fluxo terminando.

## Invariantes

- `TestAReplacedStreamEndingDoesNotQuitTheClient`
- `TestTheCurrentStreamEndingStillQuits`
- `TestAnEventFromAReplacedStreamIsDropped`
