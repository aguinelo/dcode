# A skill que alcança a fronteira

**2026-08-31** — `SkillSafetyClaims` passa a existir; skill que tenta afrouxar
segurança é **retida e perguntada**, com o trecho citado.

## O furo

`SafetyClaims` roda sobre **instruções** desde que a RN-10 pediu que a tentativa
fosse registrada. Nada rodava sobre **skill**.

E skill é o texto menos confiável que este produto carrega. Ela chega por
`git clone` em `.dcode/skills/`, ou é baixada do repositório de um estranho —
que foi literalmente o que aconteceu no teste de campo desta mesma tarde. O
corpo dela vai direto para o turno dentro de um bloco `<skill>`, sem ninguém
ler antes.

A RN-11 já tinha dito o essencial sobre esse caminho, para doutrina: *"arquivo
dentro do workspace veio junto com código que pode ter sido clonado de qualquer
lugar, e **não é o usuário**"*. Skill vinha pela mesma porta, sem a mesma
desconfiança.

## Por que este pergunta, e a RN-10 só reporta

O comentário do `safetyClaims` explica por que ele **não** filtra, e a explicação
está certa — para instrução:

> a false positive costs a line of output rather than a lost rule

Instrução é do usuário ou do projeto dele. Descartar um arquivo inteiro por uma
frase custaria a ele uma regra que ele mesmo escreveu.

Em skill a assimetria vira. Falso positivo custa **uma pergunta**, que a pessoa
responde vendo o trecho citado. Falso negativo carrega texto de terceiro dentro
do contexto do modelo, sem pergunta nenhuma. É a mesma lista de padrões,
consumida de outro jeito porque a procedência é outra — e a procedência é a
coisa que a RN-11 já usa para decidir exatamente este tipo de pergunta.

**Perguntar, e não recusar.** Recusar de saída seria o produto decidindo o que é
da pessoa, e fronteira e autorização são eixos separados desde a ADR-02: o
sandbox é a fronteira, aprovação é a autorização, e esta é a segunda. Aprovada, a
skill carrega inteira — reter é pergunta, não deleção, e quem diz sim recebe a
skill que instalou.

**Sem ninguém para perguntar, não carrega.** É a mesma regra que o laço já aplica
a toda travessia, com a mesma frase por trás: com ninguém a quem perguntar, a
única alternativa a recusar é conceder em silêncio.

**Os três desfechos deixam linha na auditoria, o concedido inclusive.**
Consentimento que não deixa rastro é indistinguível de pergunta que nunca foi
feita.

## As duas metades

Corpo **e** linha de índice. O corpo é onde a carga estaria; a linha é paga em
todo turno. Corpo inofensivo sob uma linha que pede a fronteira é a versão mais
barata do ataque, e filtrar só o corpo a deixaria passar.

## O filtro tem de ser estreito

Guarda que pergunta sobre tudo é guarda que a pessoa aprova sem ler, e aprovação
sem leitura não protege nada.

Medido contra a `web-design-engineer` — 35.012 bytes de orientação real de
terceiro, a mesma que o teste de campo usou: **zero casamentos**. É evidência de
uma amostra só, e está dito aqui como uma amostra só.

## Pergunta, não parada

A skill espera; o produto não para. Fazer o processo morrer daria a qualquer
repositório clonado o poder de impedir o `dcode` de rodar — que é exatamente o
defeito corrigido na entrada anterior deste changelog, e recriá-lo em nome da
segurança seria trocar um problema por ele mesmo.

## Invariantes

- `TestASkillThatReachesForTheBoundaryIsRefused` — retida, com o trecho.
- `TestTheIndexLineIsScreenedAsWellAsTheBody` — a linha conta como o corpo.
- `TestAnOrdinarySkillIsNotRefused` — skill comum não vira pergunta.
- `TestAHeldSkillIsPutToThePerson` — aprovada carrega, negada não, sem ninguém
  para perguntar não carrega, e os três deixam linha.
