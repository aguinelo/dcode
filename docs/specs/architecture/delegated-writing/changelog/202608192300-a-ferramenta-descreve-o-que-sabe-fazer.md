# A ferramenta descreve o que sabe fazer

Encontrado medindo, não revisando.

## O que aconteceu

A rodada 6 do experimento deu ao dcode o caso **mais fácil possível** de trabalho
divisível: cinco notas de arquitetura, uma por pacote, cada uma no seu arquivo,
nenhuma dependendo das outras, e dito na tarefa que não compartilham arquivo.

Ele escreveu as cinco. **Sozinho, em série, sem delegar nenhuma.**

```
bash 69 · read 23 · plan 8 · write 5 · explore 0
```

## Por quê

Porque a descrição da ferramenta dizia, em voz alta:

> "It can only read: no editing, no commands, and it cannot delegate further."

O schema oferecia `owns` havia três PRs. A descrição negava. **Quando as duas
discordam, quem ganha é a descrição** — é a frase sobre a qual o modelo raciocina,
e o schema ele lê depois de já ter decidido não usar a ferramenta.

É a forma que este repositório não para de encontrar em si mesmo, invertida:
não algo declarado que ninguém lê, mas algo **construído que as palavras negam**.

Nenhuma guarda pegou, porque descrição de ferramenta não tinha nenhuma. Três PRs
de teste, invariante e revisão passaram por cima disso sem ver.

## A guarda que entrou, e a que não entrou

Entrou uma estreita: `explore` não pode oferecer `owns` e dizer que só lê.

**Não entrou** a que eu tentei primeiro — "toda descrição menciona todo campo do
seu schema". Ela acusou catorze campos em quase todas as ferramentas, e estava
errada: o modelo lê também as descrições **por campo** dentro do schema, então
campo não mencionado não é campo invisível. O que faz mal é a contradição, e só
o `explore` faz afirmação sobre os próprios limites.

Descartada em vez de forçada. Guarda que acusa o que não é defeito é guarda que
alguém desliga.

## A descrição nova

Diz o que a ferramenta faz, quando usá-la, e — o que importa mais — **quando não**:

> "Do not use it ... for changes that have to stay consistent with each other:
> work that must agree with itself belongs in one head."

Essa última frase é o contrato `keeps-writing-that-must-be-coherent` da §6 do
`.p`, escrito onde o modelo lê antes de decidir, em vez de só medido depois.
