# skill-loaded-on-trigger

**Contrato:** `202608080016-behavior-definition.p.spec.md` · limiar **≥ 85%**

Tarefa que casa com o índice de uma skill; carrega e usa o corpo da skill.

## Onde fica a linha entre determinístico e medido

O **casamento** é determinístico e já é asserção: gatilho explícito, ou duas
palavras significativas em comum com o `when_to_use`. Isso não é o que este
limiar mede.

O que se mede é o modelo **usar** o corpo depois de carregado. A skill do
material declara um passo que ninguém adivinharia — atualizar um arquivo de
versão específico antes de marcar a tag — e é a presença desse passo que separa
ter lido de ter recebido.

## Por que esta fixture mantém `bash`

O terceiro passo da skill é **cortar a tag**, e isso é comando. Tirar o shell
daqui mudaria o que a tarefa pede, não como ela é medida — que é a linha entre
limpar artefato e reescrever o cenário.

Dez outras fixtures ofereciam `bash` sem que juiz ou injeção o mencionassem, e
não era de graça: o modelo abre com ele, o harness recusa, e a rodada se foi.
Em alguns casos a conclusão é pior — *"can't actually execute commands here — the
shell returns canned errors, not real results"* — e modelo que desconfia das
próprias ferramentas não mede mais nada dali em diante.
