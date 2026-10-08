# A memória e as skills de um workspace

**Data:** 2026-10-08
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`: duas rotas em
§4, a §5.4, dez linhas novas em §9)
**Fonte:** a visão de equipe (Crew) do cliente desktop, que mostra o que um
workspace lembra e quais skills ele tem, pedida em 2026-10-07.

## O problema

O desktop vai mostrar, por workspace, a memória que as sessões ali leem e as
skills que elas têm. O protocolo não respondia nenhuma das duas. Um cliente que
lesse o `.dcode/memory.md` e as pastas de skills por conta própria teria de
reimplementar a gramática do cabeçalho, o teto de entradas, a marca de memória
velha, a precedência entre a pasta do usuário e a do projeto e a retenção de
skill que pede a fronteira — cinco regras que já existem, cada uma num lugar, e
que a cópia deixaria de seguir no dia em que mudassem.

## O que mudou

- **`GET /v1/memory?workspace=`** responde o que uma sessão no workspace lê
  como memória: cada entrada, com tipo, assunto, corpo e procedência, `stale`
  quando o commit não existe mais e `shown` quando a sessão a mostra; o caminho
  do arquivo relativo ao workspace, se ele existe, se a memória está ligada e o
  teto; e os blocos que não são memória, cada um com o motivo.
- **`GET /v1/skills?workspace=`** responde as skills que uma sessão no
  workspace tem: as do usuário e as do projeto, cada uma com nome,
  `when_to_use`, gatilhos, origem e caminho; as retidas marcadas, com o que
  pedem; e os arquivos que não carregaram, com o motivo.
- **O cliente Go** ganha `ReadMemory` e `ListSkills`.
- **O pacote de memória** ganha `Diagnose`, `Hidden` e `Stale`, e o `Render`
  passa a usar os dois últimos.

## Como, e por quê assim

- **Lido pelas chamadas que montam a sessão.** A memória é o `memory.Read`, o
  velho é o `knownCommits` com o `memory.Stale` — a mesma condição que o
  `Render` imprime — e o que fica de fora é o `memory.Hidden`, o mesmo corte que
  o `Render` declara. As skills saem do `skillDirs`, a lista de pastas que o
  `New` passou a ler também, pelo `behavior.LoadSkills` e com o mesmo teto de
  bytes. Um teste monta a sessão e confere que o que a lista diz que ela mostra
  está no prefixo dela, e o que diz que não mostra não está.
- **O motivo de um bloco malformado mora no pacote de memória.** O pacote só
  guardava a linha. Um motivo escrito na rota seria uma segunda leitura do
  cabeçalho, e no dia em que as duas discordassem um bloco seria acusado de algo
  que não tem; `Diagnose` lê com o mesmo `splitHeader` que o `parse`.
- **Toda entrada é listada, não só as mostradas.** A lista é de quem cuida do
  arquivo: uma entrada além do teto, ou com a memória desligada, continua no
  arquivo, e escondê-la seria a lista guardando menos do que o arquivo tem.
  `shown` diz quais a sessão lê.
- **Desligado é dito, não é lista vazia.** `enabled` vem de `memory.enabled` e
  de `behavior.skills_enabled`. Responder vazio com o arquivo cheio leria
  "nada foi aprendido aqui".
- **Arquivo ilegível não é recusa.** A sessão abre sem a memória e diz por quê
  (`memory_unreadable`); a rota responde o mesmo em `unreadable`. Pasta de
  skills ilegível derruba a sessão, e derruba a rota igual.
- **Skill retida é listada como retida.** Carregá-la é resposta de uma pessoa a
  uma pergunta que a sessão faz; a rota não tem a quem perguntar. Deixá-la de
  fora esconderia a pergunta; mostrá-la carregada a responderia. `claims` leva o
  que ela pede, nas palavras da pergunta.
- **Caminhos relativos à origem.** O de uma skill é relativo à pasta de skills
  de onde veio, e o motivo de um aviso troca o caminho absoluto pelo relativo:
  quem lê já sabe de quem é e onde, e o diretório de configuração do usuário
  não é assunto de uma lista.
- **Workspace é obrigatório.** Diferente de `/models`, não há resposta para "a
  configuração com que o daemon subiu": memória e skills de projeto são de um
  workspace. Ausente ou relativo é recusado na borda; inexistente ou com
  configuração ilegível, com `workspace_invalid` e o motivo do `optionsFor`.
- **Daemon sem o gancho recusa em vez de responder vazio**, como `/models`.

## As invariantes

As dez linhas novas em §9 são reivindicadas em `internal/app`, pelas rotas do
próprio daemon e com arquivos de verdade — memória num repositório git com um
commit que existe e um que não, skills sob `DCODE_HOME` e sob o projeto —, e a
do gancho ausente em `internal/server`:
`TestTheMemoryRouteListsWhatASessionReads`,
`TestAnAbsentMemoryIsAnEmptyListAndNotAFailure`,
`TestAMalformedMemoryIsListedWithTheReason`,
`TestTheSkillsRouteListsWhatASessionWouldHave`,
`TestTheSkillsListedAreTheOnesASessionIndexes`,
`TestAHeldSkillIsListedAsHeld`,
`TestASkillThatCannotLoadIsListedWithTheReason`,
`TestMemoryAndSkillsNeverCarryWhatTheyDoNotDeclare`,
`TestMemoryAndSkillsRefuseAWorkspaceThatCannotBeRead` e
`TestMemoryAndSkillsWithoutTheHookRefuse`.

Rotas e tipos novos, nenhum contrato existente mudado: MINOR.

## O que fica de fora

- **A visão de equipe no desktop**, que é do laço do desktop.
- **Escrever memória ou instalar skill pela rota.** As duas são só leitura; a
  memória é escrita pela ferramenta `remember` e revisada num diff.
- **`memory.enabled` e `memory.max_entries` num `config.toml`.** O `FromEnv` lê
  as duas chaves, mas elas não estão na lista de chaves conhecidas, e um arquivo
  que as escreve é recusado como chave desconhecida. Defeito anterior a estas
  rotas, com branch própria; os testes de desligado e de teto não passam por
  ele.
