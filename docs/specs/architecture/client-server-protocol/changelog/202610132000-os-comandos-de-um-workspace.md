# Os comandos de um workspace

**Data:** 2026-10-13
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`: uma rota em §4,
a §5.6, seis linhas novas em §9)
**Fonte:** o menu de `/` do cliente desktop (D48 do desktop), que oferece os
comandos que o campo atende e não tinha como oferecer os que a pessoa escreveu.

## O problema

A TUI oferece os comandos de barra que o usuário e o projeto escrevem em
`commands/*.md` (`config.DiscoverCommands`), e os expande antes de mandar. Ela
lê os arquivos por conta própria, porque roda na máquina deles. Nenhum outro
cliente pode: o desktop fala com o daemon por HTTP, não lê a pasta de
configuração do usuário — que é do daemon descobrir, por `DCODE_HOME` e pelo
resto das raízes — e uma cópia da descoberta em TypeScript teria de seguir a
precedência entre usuário e projeto, o nome tirado do arquivo, o teto de bytes e
a recusa de comando sem corpo, quatro regras que já existem num lugar só.

## O que mudou

- **`GET /v1/commands?workspace=`** responde os comandos de barra que um cliente
  no workspace oferece: os do usuário e os do projeto, por nome, cada um com a
  descrição, a origem, o arquivo sob a pasta `commands` da origem e o corpo; e,
  em `overrides`, os nomes que o projeto tomou do usuário.
- **O cliente Go** ganha `ListCommands`.
- **O pacote de configuração** ganha `CommandMaxBytes` (os 64 KiB que a TUI já
  usava, agora um número só para todo leitor) e `CommandSet.Overridden`, os
  mesmos casos de `Collisions` pelo nome, sem os caminhos.

## Como, e por quê assim

- **A mesma descoberta da TUI.** A rota chama o `config.DiscoverCommands` com as
  raízes do ambiente do daemon e o teto da TUI. Um teste resolve os comandos
  pelas duas portas no mesmo workspace e compara: um comando não significa uma
  coisa num cliente e outra no outro.
- **O corpo viaja.** Uma skill não manda o corpo: ele entra num turno quando um
  gatilho dispara, e a lista não tem nada com isso. Um comando é o contrário:
  invocá-lo é mandar o texto dele, e o cliente precisa do texto para mandar. Um
  comando nunca roda nada (`config.Command`), então o corpo não dá a um cliente
  nada que a pessoa não pudesse digitar.
- **A expansão fica no cliente.** `$ARGUMENTS` trocado, ou o resto acrescentado
  depois de uma linha em branco, é o que o `config.Expand` faz e a §5.6 escreve.
  Uma rota que expandisse precisaria do texto inteiro digitado, e o cliente já
  manda o resultado como mensagem pela rota de turno, onde o resto das garantias
  mora.
- **Os embutidos são do cliente.** A TUI ignora um comando do usuário com nome
  de embutido (`ShadowedBuiltins`), e o desktop tem os dele. O daemon não sabe
  quais são e não adivinha: lista todos, e cada cliente decide.
- **Arquivo que não carrega recusa a lista.** É o que acontece na partida da
  TUI. Uma lista sem ele ofereceria o resto como se o quebrado não existisse, e
  a pessoa procuraria o comando que escreveu sem saber por que sumiu.
- **O caminho é o nome do arquivo.** As pastas de comandos são planas; o
  caminho absoluto levaria a pasta de configuração do usuário a uma lista que
  não precisa dela.
- **Workspace é obrigatório**, como em `/memory` e `/skills`: os comandos do
  projeto são de um workspace. Ausente ou relativo é recusado na borda;
  inexistente ou com configuração ilegível, com `workspace_invalid` e o motivo
  do `optionsFor`.
- **Daemon sem o gancho recusa em vez de responder vazio**, como `/skills`.

## As invariantes

As seis linhas novas em §9 são reivindicadas em `internal/app`, pelas rotas do
próprio daemon e com arquivos de verdade sob `DCODE_HOME` e sob o projeto, e a
do gancho ausente em `internal/server`:
`TestTheCommandsRouteListsWhatTheTUIOffers`,
`TestTheCommandsListedAreTheOnesTheTUIResolves`,
`TestACommandThatCannotLoadRefusesWithTheReason`,
`TestCommandsNeverCarryWhatTheyDoNotDeclare`,
`TestCommandsRefuseAWorkspaceThatCannotBeRead` e
`TestCommandsWithoutTheHookRefuse`.

Rota e tipos novos, nenhum contrato existente mudado: MINOR.

## O que fica de fora

- **O menu de `/` no desktop**, que é do laço do desktop e vem num PR próprio,
  em cima deste.
- **Descobrir a cada pedido contra congelar na partida.** A TUI congela os
  comandos ao abrir, para que uma barra não mude de sentido no meio da sessão;
  a rota responde o que está no disco quando perguntada, e congelar é do
  cliente que pergunta.
- **Criar ou editar um comando pela rota.** Só leitura.
