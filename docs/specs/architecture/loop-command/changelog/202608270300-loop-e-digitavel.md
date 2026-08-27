# `/loop` é digitável

**Data:** 2026-08-27
**Specs afetadas:** `202608252000-loop-command` — dez invariantes novas na §8 e
os Passos 3 e 4 da `.i` fechados. Toca `client-server-protocol` de forma
aditiva.

## O que passou a existir

```
/loop specs/2026-08-25-home-page
/loop specs/x --protect '**/*_test.go'
```

O cliente reconhece, **o texto não vira entrada de turno** (RN-3), e uma sessão
nova nasce medida contra o `tasks.md` daquela pasta.

## Quem lê o disco é o daemon

O cliente manda o **caminho**, não os critérios. Ele pode não estar perto do
sistema de arquivos do daemon, e um cliente que lesse a spec estaria afirmando
o que um arquivo que ele não enxerga contém.

`CreateSessionRequest` ganha `loop_spec` e `protect`; o daemon resolve o
caminho **sob o workspace** e recusa o que sobe. As duas metades importam:
resolver contra o workspace é o que faz `/loop specs/x` significar a mesma coisa
de qualquer cliente; recusar a subida é o que impede isso de virar leitura
arbitrária do disco do daemon.

## Spec nomeada não cai no `done.toml`

Pedir uma spec e receber em silêncio o `done.toml` do workspace seria o pior dos
dois: o turno medido contra algo que a pessoa não nomeou, e nada na tela
dizendo. Spec ilegível **encerra** a criação da sessão.

## A sessão diz quantos critérios carrega

`protocol.Session` ganha `done_criteria`, e **zero é resposta, não ausência**.

É a linha que o Code Plain precisa. Uma sessão sem definição de pronto relata
pronto no fim do primeiro turno; quem digitou `/loop` esperando uma tem que ser
avisado **ali**, não no fim. Por isso o cliente troca a mensagem quando o número
é zero, e diz o que faz uma tarefa virar critério.

## Um defeito achado rodando, e ele te atinge

Sondando o daemon de verdade, um `/loop` para spec inexistente voltava com o
código `workspace_invalid`. O workspace estava ótimo.

O servidor classificava o erro procurando a palavra `workspace` **no texto da
mensagem** — e mensagem carrega caminho. Este repositório mora em
`/Users/aguinelo/workspace/dreibox/dcode`: qualquer erro cujo texto citasse um
caminho daqui era rotulado como problema de workspace.

É a mesma forma do teste que varria o prefixo pela palavra "repository", que
quebrou hoje mais cedo quando o piso passou a usar essa palavra numa frase sobre
resumos. **Classificar por string solta é classificar errado assim que o mundo
anda.**

Agora há `policy.ErrWorkspace`, um sentinela, e a classificação é por
`errors.Is`. O erro da spec é codificado na origem. O teste que guardava isso
fabricava a string e passava; agora ele usa o sentinela **e** afirma que um
caminho contendo a palavra **não** é problema de workspace.

## O Passo 4 fechou por outro caminho

A `.p §5` desenhou `SessionConfig` montando o `loop.Config` no cliente. A sessão
acabou nascendo no daemon a partir do caminho — pela mesma razão acima. O
`SessionConfig` segue sem chamador e a isenção do `specguard` continua dizendo
isso, agora pelo motivo certo.

## O que isto não resolve

**Os `tasks.md` reais do Code Plain não têm marcador `verify:`.** Medido: as 17
specs devolvem zero critérios. `/loop` numa delas hoje abre sessão e diz, com
todas as letras, que não há definição de pronto.

É o Passo 5 da `.i` — congelar o formato — e ele deixou de ser hipótese. As duas
saídas são o Code Plain acrescentar os marcadores, ou o parser aprender o
formato que existe. A segunda é decisão de contrato entre dois repositórios e
não se toma sozinha.
