# Quem pergunta é o harness, e a doutrina não dizia isso

A seção `Safety` abria com:

> Some actions cross a boundary the operating system enforces: reading or
> writing outside the workspace, and reaching the network. **When that happens
> the user is asked**, and a refusal is final.

Voz passiva, sem sujeito. O modelo preencheu o sujeito com **"eu"**, e a partir
daí construiu um protocolo de permissão próprio, em prosa, paralelo à máquina de
aprovação e que nunca a aciona. Na tela de um usuário:

> Eu tenho que te perguntar, e você tem que dizer "vai" explicitamente — não
> inferir do contexto, não deduzir do tom.

Três linhas abaixo, a mesma doutrina já dizia *"Do the work and let the boundary
ask"* e *"announcing that you will not run something (…) is a refusal the user
never gave"*. O modelo **citava a primeira frase para justificar** exatamente o
que a terceira proíbe. Não foi desobediência: a primeira instrução chegou antes,
não tinha sujeito, e o resto virou nuance a ser racionalizada.

A correção nomeia o sujeito e diz onde a pergunta acontece:

> When the work needs one of them, CALL THE TOOL. The harness asks the user —
> you do not. The call IS how the question gets asked, and there is no other way
> to ask it. Asking in prose does not reach the machinery, it replaces it:
> permission granted in prose changes nothing, because nothing was ever asked.

Três acréscimos vêm junto, cada um fechando uma saída que o relato mostrou em
uso. **Nunca inventar uma frase que o usuário tenha de dizer de volta** — foi
literalmente o que aconteceu, e uma permissão dada assim não chega a lugar
nenhum. **Ser mandado fazer o trabalho já é a instrução de tentar** — o usuário
tinha mandado, e o modelo pedia que mandasse de novo, de um jeito específico.
E **discordar é dizer uma linha e tentar mesmo assim**: a recusa vinha embrulhada
numa discussão técnica (Node 24 contra Node 22) que o modelo já tinha perdido,
usando a fronteira como argumento.

A ordem também mudou. A instrução ativa — chame a ferramenta — vem agora antes
da regra sobre recusa. A regra sobre recusa continua, e continua absoluta, mas
depois de estar dito quem recusa: *"a refusal is final — but the refusal has to
have been GIVEN"*.

Nada disso afrouxa a fronteira. `safety-not-overridable` segue a 100% e a
garantia continua estrutural: a política do sandbox não consulta o prompt. O que
muda é o modelo parar de responder no lugar do usuário — que é a metade cara,
porque uma fronteira que ninguém chega a cruzar não chega nunca a perguntar.
