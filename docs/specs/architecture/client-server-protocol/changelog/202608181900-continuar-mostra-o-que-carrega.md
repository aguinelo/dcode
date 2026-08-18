# Continuar mostra o que carrega

**Data:** 2026-08-18

## O que mudou

A conversa que uma sessão continua entra no **log** dela, atrás de um evento
novo, `session.resumed`, que nomeia a sessão de origem e quantos turnos vieram.

Antes ela ia só para o motor de contexto.

## Os dois defeitos, e a raiz única

**A conversa chegava ao modelo e nunca à pessoa.** A tela é montada a partir de
eventos; continuar não emitia nenhum. Não havia tipo de evento para isso nem
campo em `protocol.Session` dizendo que a sessão continua outra — por
construção, não existia canal por onde aquilo chegasse à tela. Quem rodava `-r`
via uma janela em branco, e a leitura disponível era que o trabalho tinha
sumido.

**A corrente arrebentava no segundo elo.** O registro da sessão nova guardava só
os turnos dela. Continuar algo que já era continuação carregava apenas o último
trecho — e esse perdia o modelo também, em silêncio. O único teste que existia
verificava que a sessão nova não era a antiga, e nada além disso.

A raiz é uma só: **o carregado não entrava no registro da sessão nova.** Pôr no
log resolve os três leitores de uma vez — a tela desenha de eventos, o registro
é escrito de eventos, e a próxima sessão a continuar esta reconstrói do
registro. Qualquer outro lugar atende no máximo um.

## Por que duas leituras do mesmo arquivo

`Rebuild` responde o que o modelo recebe; `Carry` responde o que a pessoa vê.
São de propósito diferentes: mensagem não é assistível — não tem travessia de
ferramenta, nem aprovação, nem raciocínio, nada do que alguém precisa ver para
saber que o trabalho sobreviveu. Carregar só o que o modelo precisa é
exatamente o que deixava a tela vazia.

## O que não é reproduzido

`session.created` da origem, porque descreveria workspace, modelo e sandbox que
não são os que valem agora — e porque um registro que abre com turno alheio não
é registro que se consiga descrever.

`tool.approval_required` da origem, porque é travessia já decidida: reproduzi-la
poria um modal na frente de alguém por uma pergunta respondida ontem.

## Sequência e identidade

Os eventos carregados são **reanexados**, não copiados: recebem a sequência e o
id da sessão nova. Dois eventos com o mesmo `seq` são uma repetição que o
cliente não tem como distinguir.

## Invariantes que entraram

- A conversa continuada entra no log da sessão nova atrás de `session.resumed`,
  com a sequência e o id dela.
- Continuar uma continuação carrega a conversa inteira, não só o último trecho.
- Conversa continuada é **exibida**, atrás de marca dizendo de qual sessão veio.
