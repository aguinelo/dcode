# Os modelos que uma sessão pode pedir

**Data:** 2026-10-07
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`: uma rota em §4,
a §5.3, seis linhas novas em §9)
**Fonte:** o menu de modelo por sessão do cliente desktop, que mostra se cada
modelo tem medição por trás (`desktop/docs/DECISIONS.md`, A5: trocar de modelo
ainda avisa que não existe), pedido em 2026-10-06.

## O problema

O desktop vai ganhar um menu de modelo por sessão, e o menu precisa dizer se
cada modelo tem medição por trás. O protocolo não respondia nenhuma das duas
perguntas. Quais modelos existem só se sabia lendo o `models.toml` por conta
própria — o do usuário e o do projeto, com a regra de quem vence —, e se a
família de um modelo foi medida é `provider.Unmeasured`, dentro do daemon. Um
cliente que reimplementasse as duas coisas acoplaria o desktop ao formato do
arquivo e teria uma segunda cópia da lista de famílias medidas, justamente a
lista que um guarda confere contra as medições que existem.

## O que mudou

- **`GET /v1/models?workspace=`** responde o que uma sessão naquele workspace
  pode pedir: `default`, o modelo que ela recebe sem pedir nenhum, e `profiles`,
  cada perfil do `models.toml` — o do usuário com o do projeto por cima, pelo
  nome —, em ordem de nome. Sem `workspace`, vale a configuração com que o
  daemon subiu.
- **Cada escolha** (`ModelChoice`) traz o nome a pedir, o modelo, a família e o
  transporte resolvidos, o endpoint configurado, a janela que a sessão teria,
  `measured` e, quando a família não tem medição, `notice` com a admissão dela.
- **O cliente Go** ganha `ListModels`.

## Como, e por quê assim

- **Resolvido pelo caminho que monta a sessão.** A cadeia do workspace é a do
  `optionsFor`, o nome é aplicado pelo `applyModelRequest` — o mesmo de um
  `CreateSessionRequest.Model` — e família, transporte e janela vêm do provider
  que o `buildProvider` compõe, com a conta de janela do `New`. Um menu que
  resolvesse por conta própria ofereceria uma coisa e abriria outra no dia em
  que um dos lados mudasse; um teste monta a sessão de cada escolha e confere.
- **Família e transporte resolvidos, não os configurados.** `measured` precisa
  da família que a sessão de fato usa, e um transporte vazio não diz nada a quem
  escolhe. `base_url` vai como configurado — vazio é o do transporte —, o mesmo
  valor que `Session.BaseURL` já carrega.
- **`measured` só vale para uma escolha que monta sessão.** `provider.Unmeasured`
  responde pelas famílias que existem, e para um nome que nenhuma família
  reivindica devolveria `""`: ler isso como medido afirmaria medição de uma
  família que nada aqui resolve. Um perfil assim — modelo sem família, família
  ou transporte que esta build não tem — continua no menu, com `measured` falso
  e, em `notice`, a recusa que a sessão daria. Tirá-lo deixaria a pessoa
  procurando o perfil que escreveu; derrubar o menu inteiro por ele levaria
  junto as escolhas que funcionam.
- **Nunca uma credencial.** A descrição é montada sem a chave — compor um
  provider não precisa dela —, e o teste procura no corpo da resposta a chave, a
  máscara, a impressão digital e de onde ela veio, e recusa campo que a rota não
  declare.
- **Workspace ruim é recusado, como uma sessão ali seria.** Relativo, na borda;
  inexistente ou com configuração ilegível, com `workspace_invalid` e o motivo
  que o `optionsFor` já dá. Responder com a configuração do daemon no lugar
  ofereceria modelos que o projeto não escolheu.
- **Daemon sem o gancho recusa em vez de responder vazio.** Um menu sem nada
  leria "nenhum modelo pode ser pedido", o que nunca é o caso.

## As invariantes

As seis linhas novas em §9 são reivindicadas em `internal/app`, pelas rotas do
próprio daemon e com configuração de verdade — perfis num `models.toml` sob
`DCODE_HOME`, outro no projeto, e a família `generic`, que não tem medição:
`TestTheMenuListsTheDefaultAndTheProfilesOfTheWorkspace`,
`TestAChoiceDescribesTheSessionItOpens`, `TestAModelWithNoMeasurementSaysSo`,
`TestAProfileNoSessionCanBuildIsListedWithTheReason`,
`TestTheListOfModelsNeverCarriesACredential` e
`TestAWorkspaceThatCannotBeReadIsRefusedWithTheReason`.

Rota e tipos novos, nenhum contrato existente mudado: MINOR.

## O que fica de fora

- **O menu no desktop**, que é do laço do desktop.
- **A TUI ler esta rota** para completar o `/model`, em vez dos nomes de perfil
  que ela lê do `models.toml` no próprio processo: mudança da TUI, com PR
  próprio.
- **A sessão dizer o aviso ao abrir.** O `New` monta o texto de
  `provider.Unmeasured` em `Session.Notice`, mas nada o lê, e com
  `instruction.notice` ligado — o padrão — ele é sobrescrito pelo aviso das
  instruções. Defeito anterior a esta rota, com branch própria; até lá, este
  menu é o único lugar em que o aviso chega a alguém.
- **Credencial escrita dentro de `base_url`** (`https://usuario:senha@host`): vai
  como configurada, como já vai no `Session` e no registro. Se um dia precisar
  ser escondida, é no protocolo inteiro, não só nesta rota.
