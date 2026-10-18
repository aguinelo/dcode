# `dcode orchestrate` abre a orquestradora do projeto

**Data:** 2026-10-18
**Specs afetadas:** `202608081250-client-tui` (`.p`, §2.2, §7 e §10),
`202610132000-orchestration` (`.i`, passo 13; `.p`, §13)

O resto do passo 13 da `.i` da `orchestration`. A lista viva já existia, dobrada do fio
(`202610172345-a-coluna-da-orquestradora.md`); faltava chegar nela, lê-la do daemon ao abrir, e
as duas teclas que agem sobre ela.

## O que entrou

- **`dcode orchestrate`** (`cmd/dcode/orchestrate.go`) resolve o projeto da pasta pelo
  `POST /v1/projects`, que é idempotente e reconhece outra worktree ou um link da mesma árvore, e
  anexa à orquestradora pelo `GET /v1/projects/{id}/orchestrator`. `dcode` sozinho continua a
  conversa comum: é a decisão 4 do dono, e o padrão só troca depois de o log da primeira fatia ser
  lido.
- **Ele não sobe daemon próprio**, ao contrário de `dcode tui`. A orquestradora mora no daemon e
  sobrevive ao terminal (RN-2); um daemon embutido morreria com o terminal e levaria os filhos.
  Sem daemon, o erro diz o socket, o que falhou e que é preciso `dcode serve`. Daemon que não
  conduz projetos responde `orchestrator_unavailable`; o erro repete a mensagem dele e nomeia as
  duas chaves que decidem — `orchestrator.enabled` e `record.enabled` —, porque a mensagem do
  daemon nem sempre nomeia (um servidor sem projetos diz só que não conduz nenhum). Sessão que
  volta sem `role: orchestrator` é recusada: abrir uma conversa comum como se fosse a orquestradora
  daria as teclas dela a quem não as tem.
- **A lista é lida ao abrir, depois do histórico.** `Options.Orchestration` é uma interface
  própria (`Children`, `Questions`, `AnswerQuestion`), e não mais três métodos no `Transport`: só
  `dcode orchestrate` a entrega, e uma conversa comum não tem uso para ela. A leitura espera o
  histórico acabar porque a releitura do log dobra todo `child.*` que ele já teve; uma resposta
  pousada no meio dela teria de volta filhos que o daemon já soltou depois da retenção.
- **`^Q`** abre a pergunta mais antiga como sobreposição, com o cursor na recomendada — Enter
  aceita o conselho, e qualquer outra resposta é um gesto deliberado. A recomendada é marcada por
  palavra e o cursor por caractere, nunca só por cor. Os rótulos que o daemon manda são ingleses
  (a língua do fio); as respostas dos tipos que ele pergunta — `merge`, `return`, `close`,
  `continue`, `stop` — são traduzidas aqui, e as de outro tipo saem como vieram.
- **O primeiro que responde leva.** `question_already_answered` é dito, a pergunta sai da lista e
  a lista é relida. Pergunta respondida em outro lugar — o desktop, outro terminal — com esta
  aberta fecha e diz por quê, em vez de ficar na tela esperando um `POST` que só seria recusado.
  Outra falha é dita e a pergunta fica aberta: nada pode parecer respondido.
- **`^R`, sob a orquestradora, lista os filhos** em vez das conversas gravadas, e escolher um
  **anexa** a ele — onde a aprovação dele é respondida. Só anexa: filho que o daemon não tem é
  dito pelo id, nunca continuado numa sessão nova por palpite, que seria um segundo escritor numa
  worktree que é do harness. Do filho, a orquestradora é a primeira linha, e voltar a ela relê a
  lista. A lista atravessa a troca, para que do filho `^R` ainda mostre os irmãos e `^Q` ainda
  responda à orquestradora.
- **Fora da orquestradora**, `^Q` diz onde funciona — tecla que não faz nada e não diz nada se lê
  como tecla quebrada — e `^R` continua a lista de conversas. O `/help` lista as duas, e a barra
  diz "agentes" no `^r` quando é essa a lista que ele abre.

Dezoito invariantes novas na §10 da `client-tui`, uma linha na §13 da `orchestration`, cada uma
com teste em `internal/tui/orchestrate_test.go` ou `cmd/dcode/orchestrate_test.go`.

## O que ficou de fora, e por quê

- **A §8 não muda.** Ela é dos comandos `/` dentro da TUI; `dcode orchestrate` é subcomando da
  linha de comando, está no texto de uso nas duas línguas e na §2.2.
- **Nenhuma invariante prevista da `orchestration` (§14) sobe aqui.** A §14 já está vazia: as três
  últimas subiram com o passo 12, os cenários roteirizados. A linha nova da §13 é deste passo.
- **A lista vista de um filho não se atualiza.** O cliente assina o log de uma sessão por vez; no
  filho, a lista é a que se tinha ao sair da orquestradora, e voltar a ela relê. Assinar os dois
  logs ao mesmo tempo é mudança do `Transport`, e não há pedido para ela ainda.
- **A nota (`note`) da resposta** não é pedida: a pergunta de merge recusada com o porquê
  continua sendo dita na conversa com a orquestradora, que é onde um porquê longo cabe.
