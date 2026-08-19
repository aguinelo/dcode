# Um invariante diz o que a regra faz sob `never`

A §6 do `.p` dizia **"Regra nunca é avaliada sob política `never`"**, e a linha
logo abaixo dizia **"`never` nega o que uma regra escalonaria"**. As duas não
podem ser verdade ao mesmo tempo: negar o que uma regra escalonaria exige
avaliar a regra.

A primeira ficou para trás quando o comportamento mudou. Até então
`evaluateRules` era pulado sob `never` de fato — e era esse pulo que deixava
`rm -rf /` passar, porque sem regra não havia nada para negar. A correção
removeu a guarda e fez a regra rodar sob toda política, com `never` respondendo
não em vez de perguntar.

O invariante passa a ser **"Regra avaliada sob política `never` nunca vira
pergunta"**, que é o que `TestTheApprovalPolicyStillGovernsRules` já assertava o
tempo todo: sob `never` a decisão não é `escalate`, e sob `on-request` é.

Nada muda no produto. O que muda é a spec parar de afirmar duas coisas
incompatíveis sobre o mesmo caminho — e o teste que ela nomeia passar a
descrever o invariante que carrega, em vez de contradizê-lo.
