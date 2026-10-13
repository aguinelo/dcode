# Um contrato, várias famílias

**2026-10-13** — `Measured` passa a guardar o mesmo contrato medido contra mais
de um modelo; a tabela de estado conta os contratos medidos por família; o eval
honra `model.family` e procura a credencial da família do modelo medido.

## A premissa

*Harness > modelo*: as mesmas premissas valem qualquer que seja o modelo, e o
que o harness não consegue impor é medido **por família**. A régua não deixava
fazer isso, por três motivos.

## O achado

**Um contrato só podia ser medido contra um modelo.**
`TestEveryMeasurementNamesWhatProducedIt` (`internal/evals/changelog_test.go:154`)
reprovava ID repetido em `Measured`: *"a medição mais nova substitui a anterior
em vez de se juntar a ela"*. A regra era certa para uma remedição contra o
**mesmo** modelo. Contra outro, apagava a primeira. Medir
`boundary-decides` no claude exigiria jogar fora a medição do MiniMax, e o
próprio comentário de `Measurement.Model` diz que um limiar medido contra um
modelo não diz nada sobre outro.

**O eval ignorava `DCODE_FAMILY`.** `setup` (`internal/evals/harness_eval_test.go:66`)
resolvia o provider com `reg.Resolve(cfg.Model, …)`, só por prefixo. A sessão usa
`ResolveFamily` quando `model.family` está posto (`internal/app/app.go:821`).
Modelo que nenhum prefixo reivindica — todo modelo local, pela `generic` — não
podia ser medido. A família que o produto mais avisa ser desconhecida era a
única que nenhum eval alcançava.

**A credencial era a de `model.name`.** `app.FromEnv` resolve as opções para
`model.name` (MiniMax-M3 por padrão) e procura a credencial guardada pela família
**desse** modelo (`CredentialName`, `internal/app/app.go:802`). O eval só trocava
o modelo na chamada. Medir `gemini-2.5-flash` numa máquina logada nos dois
mandava a chave do MiniMax para o Google.

## O que muda

- `Measurement.Family`: a família pela qual a execução passou, quando o modelo não
  a diz. Vazio é a família cujo prefixo reivindica o modelo, que é toda medição
  até hoje. É obrigatório quando nenhum prefixo reivindica: a guarda reprova
  medição que não diz a qual família pertence.
- A chave de `Measured` é **contrato + modelo + família**. Dois modelos são duas
  medições; o mesmo duas vezes continua sendo uma leitura que deveria ter
  substituído a outra. A família entra na chave, e não só o modelo, porque com
  `DCODE_FAMILY` o mesmo modelo pode passar por duas famílias — formulação
  diferente, prompt diferente, medição diferente.
- "Contratos de fato já medidos" passa a contar **contratos** com ao menos uma
  medição confiável, não medições. Hoje os dois números coincidem (21).
- Linha nova e contada na tabela de estado: contratos medidos **por família**,
  todas as famílias do build listadas, as de zero inclusive. Hoje:
  `minimax-m3` 20, `gemini` 1, `claude` 0, `generic` 0.
- O eval resolve como a sessão: `ResolveFamily` quando `model.family` está posto,
  prefixo caso contrário; e procura a credencial para o modelo medido.
  `DCODE_API_KEY` continua vencendo, como no produto. A linha de resultado passa a
  dizer a família, para que registrar a medição continue sendo copiar uma linha.
- A guarda de impressão digital (`202610131200-a-impressao-do-que-foi-enviado.md`)
  reconstrói o prompt na família da medição, não na que o prefixo do modelo daria.

Essas decisões saíram de código que só existia com a tag `eval`, que nenhum teste
unitário alcança. Saíram para funções sem tag (`measuredOptions`,
`measuredProvider`), e o eval as chama.

## O que **não** muda: o que "medida" quer dizer para uma família

A RN-11 do `provider-adapter` diz quando uma família avisa que não foi medida:
*"família sem **nenhuma** medição registrada"*. É binário por família, e a
`gemini` saiu da lista em 4 de setembro com **um** contrato de cinquenta e cinco.
O aviso deixou de ser verdadeiro, mas a frase que o substitui — silêncio — diz
mais do que a medição sustenta.

A spec decide isso, então isto não muda aqui. A linha por família torna a
distância visível, e a proposta de regra está no `docs/ROADMAP.md`, §18.

## Invariantes

- `TestOneContractIsMeasuredOncePerFamily`
- `TestAnUnclaimedModelMustNameItsFamily`
- `TestTheEvalHonoursANamedFamily`
- `TestTheEvalLooksUpTheCredentialOfTheModelItMeasures`
- `TestTheEnvironmentKeyStillWins`
- `TestTheResultLineNamesTheFamily`
- `TestTheStateTableIsCountedAndNotCarried` passa a conferir a linha por família,
  nas duas edições.
