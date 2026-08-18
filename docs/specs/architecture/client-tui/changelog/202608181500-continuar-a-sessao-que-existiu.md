# Continuar a sessão que existiu

**Data:** 2026-08-18

## O que mudou

`dcode -r` passa a retomar a sessão mais recente **em que algo foi perguntado**.
Registro sem turno é pulado.

Antes retomava a mais recente, ponto — e a mais recente era, muitas vezes, uma
sessão vazia.

## Por que o defeito era garantido, não ocasional

O registro é aberto **antes** da primeira pergunta. Toda interface que abre e
fecha sem perguntar nada — e toda que falha ao abrir — deixa um registro de uma
linha para trás.

Isso fecha um ciclo em si mesmo: rodar `-r` cria um registro vazio, e é esse
registro que o próximo `-r` escolhe. **Tomada ao pé da letra, a opção destruía o
próprio alvo a cada uso.** Quem usa `-r` duas vezes seguidas nunca continua nada
na segunda.

Encontrado depois de um `-r` que morreu por não haver TTY. Ele falhou, não
continuou coisa alguma, e mesmo assim deixou o registro que envenenaria o
seguinte — a falha custou mais que o erro que reportou.

## O que não mudou, e por quê

**A sessão vazia continua sendo gravada.** A tentação era não gravar antes do
primeiro turno, e é a decisão errada: `dcode sessions` precisa listar a sessão em
que a pessoa está sentada agora, que ainda não perguntou nada. Já existe
formulação para isso — `(nothing asked yet)`. O defeito não era gravar; era
escolher sem olhar.

## As duas ausências ditas separadamente

Não haver sessão nenhuma neste workspace e só haver sessões vazias são situações
diferentes, com conserto diferente: uma pede começar aqui, a outra pede escolher
outra pelo id. Passam a ter mensagens distintas.

## Invariantes que entraram

- `--continue` retoma a sessão mais recente **em que algo foi perguntado**:
  registro sem turno é pulado.
- Só haver registro vazio é dito como tal, distinto de não haver sessão alguma.
