# Uma família para o Gemini

**2026-09-01** — `provider.Gemini` sobre o transporte `openai`; RN-11 passa a
existir, e a guarda dela achou que a `claude` já estava calada havia meses.

## O que foi feito, e o que não foi

Uma **família**, não um transporte. O Gemini fala o dialeto OpenAI pela
superfície de compatibilidade do Google, então a codificação e o decodificador
são os que já existem — a `Gemini` embute a `MiniMaxM3` pelo mesmo motivo que a
`Generic` embute: formato de fio é formato de fio, e uma segunda cópia é uma
segunda coisa para manter em dia.

O que ela sobrescreve é exatamente o que o eixo **família** existe para carregar:
nome, prefixos de modelo, janela, limites e o que ela lê.

A superfície nativa (`:streamGenerateContent`) ficou de fora de propósito. Ela
põe o modelo na URL, autentica com cabeçalho próprio, enquadra o stream de outro
jeito, e codifica chamada como `functionCall`/`functionResponse` com papéis
`user`/`model` e `systemInstruction` separado. Isso é **transporte** — e escrever
um antes de alguém ter rodado esta família contra uma chave real seria construir
a metade difícil primeiro, em cima de um palpite.

## Os números, e por que não são cópias

**Janela: 1.000.000**, abaixo do 1.048.576 documentado. Errar para baixo compacta
cedo e custa um resumo; errar para cima estoura a janela e perde o turno. A
assimetria decide de que lado errar. Tabela por modelo seria lista de números que
ninguém aqui confere, envelhecendo no calendário do Google em vez de no nosso.

**Limites: 50 iterações**, e explicitamente **não** as 2000 da MiniMax. Aquele
número é justificado por uma execução de horizonte longo citada e por uma sessão
real que o teto truncou. Nada disso existe para o Gemini, e copiar teto entre
famílias é como um limite deixa de significar alguma coisa. Há teste afirmando
que os dois diferem.

**Imagens: sim.** Declarado, não tentado — que é a regra do campo — e declarado
sem ter rodado contra uma chave aqui, que é a ressalva honesta. Dizer não seria o
erro maior: tornaria indisponível, em silêncio, a propriedade pela qual esta
família é mais conhecida.

## `Encode` recusa o dialeto que a família não declara

Herdada da `MiniMaxM3`, ela serializaria Anthropic alegremente, para uma família
cujo `Transports()` nomeia um só. O registro não compõe esse par, então hoje isso
não é alcançável — e desacordo inalcançável entre dois métodos do mesmo tipo é
exatamente o que vira alcançável depois sem ninguém notar.

## RN-11, e o que a guarda dela achou

Nome de família, aqui, lê como família **medida**. O `Measurement.Model` existe
porque limiar pertence a um modelo e não fala nada de outro. Acrescentar a
`gemini` e não dizer nada poria um nome onde a verificação não está.

Então família sem medição avisa. E a lista de quem avisa **não é digitada**: uma
guarda a confere contra as medições que existem, nos dois sentidos — quem ganha
medição e continua avisando reprova, quem sai da lista sem medição reprova
também.

Na primeira execução ela reprovou por **`claude`**. A família existe desde o
começo, "para provar que os eixos são ortogonais", e nunca teve uma única
medição — vinha rodando sem dizer isso a ninguém. Não foi o que eu fui procurar;
foi o que a guarda encontrou por existir.

O aviso é um texto só para as duas, e não uma constante para cada: três cópias da
mesma frase divergem, e a frase é o produto inteiro daquela função. A `generic`
mantém o dela, mais largo — lá o endpoint em si é desconhecido, e a janela e as
imagens também são chute.

## O que falta para dizer "o dcode suporta Gemini"

Medir. São 53 contratos que precisam de modelo, e o que existe hoje foi medido
contra MiniMax-M3. Até lá, o produto diz o que sabe e o que não sabe, que é a
única versão honesta disponível sem gastar as chamadas.

## Como apontar

`model.base_url` para `https://generativelanguage.googleapis.com/v1beta/openai`.
O `defaultBaseURL` responde por **transporte**, não por família, e por isso não
nomeia o Gemini: transporte decidindo coisa a partir de família é o colapso dos
eixos que a documentação da própria interface proíbe.

## Invariantes

- `TestGeminiClaimsItsOwnModelsAndNoOthers`
- `TestGeminiEncodesOpenAIAndRefusesTheOther`
- `TestGeminiWindowErrsOnTheCheapSide`
- `TestGeminiDoesNotInheritMiniMaxsHorizon`
- `TestEveryUnmeasuredFamilySaysSo` e `TestAnUnmeasuredWarningNamesItsFamily`,
  em `internal/evals` porque comparam com `Measured` e este pacote não pode
  importar aquele.
