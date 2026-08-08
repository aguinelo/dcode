# Marca no estado vazio

**Data:** 2026-08-08
**Specs afetadas:** `202608081250-client-tui` (`.r`, `.p`, `.i`)

## O que mudou

Nova **RN-10 — a marca aparece dentro do produto**: o estado vazio de uma sessão nova
exibe o mascote e o nome. A antiga RN-10 (cliente descartável) passou a **RN-11**.

Nova seção 8.1 no `.p` com o layout do estado vazio, e Passo 10b no `.i`.

## Por que mudou

A identidade visual do projeto foi definida — mascote de três caixas em pixel art,
documentado em `docs/brand/`. A regra que orientou o desenho foi que **marca que não
renderiza na própria ferramenta é decoração externa**, e essa regra só vale se estiver
escrita em algum lugar que a revisão consulte.

O mascote também é a validação da escolha estética: pixel art foi escolhido justamente
porque sobrevive ao terminal e a uma cor só. Se a degradação para ASCII perdesse as três
caixas, a escolha estaria errada — e é isso que o teste do Passo 10b verifica.

## Restrições que a regra impõe

- **Some no primeiro turno e não volta.** Splash persistente rouba altura do fluxo, que é
  o recurso escasso da tela.
- **Nunca em sessão retomada com histórico.** Quem retoma quer ver onde parou, não a marca.
- **Degrada para ASCII e para uma cor**, como todo o resto da interface.

## Detalhe que amarra

O olho do mascote é o mesmo `⏺` que marca cada linha de execução no fluxo. A marca se
repete a cada `⏺ read`, `⏺ edit`, `⏺ bash` — sem ocupar espaço próprio depois do primeiro
turno.
