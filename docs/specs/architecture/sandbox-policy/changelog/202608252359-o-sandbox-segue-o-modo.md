# O sandbox segue o modo, ou o modo é mentira

`/mode auto` trocava metade da fronteira. A política passava a responder
`allow`, o crachá na barra passava a dizer `auto`, e a escrita fora do
workspace continuava voltando `EPERM` — porque o sistema operacional seguia
aplicando o limite com que a **sessão nasceu**.

Medido no binário de verdade, num pty, antes e depois:

```
[assist] mkdir fora do workspace ... bloqueado
/mode auto                          crachá vira auto
[auto]   mkdir fora do workspace ... bloqueado   ← antes
[auto]   mkdir fora do workspace ... funciona    ← depois
```

A causa é uma linha. `sandbox.Runner` recebia `Mode policy.SandboxMode` — um
**valor**, copiado quando a sessão foi montada, em `app.New`. `Engine.SetMode`
mudava `cfg.Mode`, que é o que a política lê, e nada mais. Os dois eixos que a
§2.1 promete mover juntos moviam-se em lugares diferentes, e um deles nunca
soube da troca.

É a terceira vez, neste dia, que o mesmo defeito aparece com roupa diferente: um
estado guardado ao lado da verdade em vez de derivado dela. O nome do modo
guardado ao lado do par. O par lido fora da trava que o escreve. E agora a
fronteira do SO copiada de um par que ainda ia mudar. **O que se copia, diverge.**

`Runner.Mode` passa a ser `func() policy.SandboxMode`, perguntada **uma vez por
comando**. Quem não é sessão — o critério de "done", que é o daemon conferindo a
própria definição, não a pessoa trabalhando — declara `sandbox.Fixed(...)` e
segue com o modo configurado. Dizer qual dos dois se é vale mais que um default,
porque o default conveniente aqui é justamente o que para de seguir em silêncio.

Fonte nula é `read-only`. Um executor a que ninguém deu modo é um executor cuja
fronteira ninguém decidiu, e a leitura que este repositório sustenta para
"ninguém disse" nunca é `allow` (RN-3).

O fio entre o motor e o sandbox é atado depois de os dois existirem, como o
`explore.Delegator` ao lado, e é guardado num `atomic.Pointer` em vez de um
campo simples: escrito na goroutine que monta a sessão, lido na de cada turno —
exatamente a forma da corrida que o `SetMode` teve de corrigir horas antes. Um
campo simples funcionaria na prática e estaria errado sob `-race`, e "funciona
na prática" foi o que aquela corrida também disse.

Uma guarda de fiação caiu junto, e merece nota. `TestNewWiresEveryLoopConfigField`
procurava `Mode:` no corpo inteiro de `app.New` e conferia o primeiro que
achasse. Passava porque o primeiro `Mode:` do arquivo por acaso era o certo —
ela verificava **ordem no código**, não a atribuição que nomeia. Foi ao vermelho
no dia em que outro `Mode:` foi escrito acima. Agora recorta o literal
`loop.Config{…}` antes de procurar.
