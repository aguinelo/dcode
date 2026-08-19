# Uma escrita recusada diz que era escrita

PR 3 da §9 do `.p`, e o último. Fecha as quatro invariantes que faltavam.

## O que mudou de verdade

Uma só coisa, e ela existia porque a delegação nunca tinha escrito.

Consentimento dado ao pai não vale para o filho (ADR-02), então um turno
delegado nunca pergunta: o que seria pergunta vira recusa, reportada em vez de
engolida. Isso já era verdade — e o relatório chamava **toda** recusa de
`could not read`, porque era o único tipo que existia.

Com o filho escrevendo, o nome fica errado. Escrita recusada reportada como
caminho não lido diz ao pai que o filho **não olhou**, quando o que aconteceu é
que ele olhou, decidiu e foi impedido. São fatos diferentes e levam a decisões
diferentes: um buraco no que o filho sabe, contra trabalho que o pai pediu e não
aconteceu.

`denyAll` passa a separar as duas listas, e decide pela **fronteira** que a
política cruzou, nunca pelo nome da ferramenta — a fronteira é o que a política
decidiu, o nome é só como o modelo chamou.

## As outras três já eram verdade

E é por isso que elas estavam na lista.

Tokens do filho debitados do pai, teto de concorrência da sessão, e filho sem
critério de pronto: os três vinham herdados da delegação somente-leitura e
continuaram valendo com o filho escrevendo. O que faltava não era código, era o
**teste que os reivindicasse nessa condição**.

Reivindicar sem testar é o que este repositório não faz, e uma invariante que
vale "por herança" é exatamente a que quebra quando a herança muda.

## O caso que fecha o desenho

`TestAChildCarriesNoDefinitionOfDone` verifica as duas metades da regra que a
§5 do `.p` estabelece: o filho não carrega critério nenhum, **e** as instruções
dele dizem, com todas as letras, que conferir a árvore não é trabalho dele.

A segunda metade importa tanto quanto a primeira. Sem critério, um modelo
prestativo roda a suíte por conta própria — sobre uma árvore que ainda vai
mudar, produzindo um verde que ninguém confere de novo.
