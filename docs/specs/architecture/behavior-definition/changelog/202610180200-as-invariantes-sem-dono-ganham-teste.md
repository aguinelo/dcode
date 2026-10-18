# As invariantes sem dono ganham teste

**Data:** 2026-10-18
**Specs afetadas:** `202608080016-behavior-definition` (`.p`, seção 8);
`202608262200-working-defaults` (`.p`, seção 7)
**Fonte:** a seção 7 da `working-defaults`, que listava três invariantes como
**sem reivindicação** e dizia que fechavam aqui.

> **Regra:** nenhum comportamento muda. A spec passa a declarar como
> verificável o que o código já faz e os testes já afirmam.

## O que mudou

A seção 8 ganha cinco linhas, cada uma reivindicada por um teste que existe:

- a auditoria da doutrina imprime a linha de `Practices` —
  `TestAuditNamesAllFourSectionsAndTheirOrigin`, que agora confere essa linha:
  o teste nomeava quatro seções desde antes do piso existir e nunca procurou a
  quinta;
- diretório sem manifesto devolve lista vazia, sem erro —
  `TestNothingDeclaredIsNoGates`;
- `package.json` malformado volta nomeado, com o motivo, sem apagar o outro
  manifesto — `TestAMalformedPackageJSONIsUnreadableNotEmpty`;
- `Makefile` que não termina de ser lido volta nomeado, sem leitura parcial
  oferecida como lista — `TestAMakefileThatCannotBeScannedIsUnreadable`;
- manifesto ilegível é avisado na abertura da sessão, e não a impede —
  `TestAnUnreadablePackageJSONIsSaid`.

A linha da `working-defaults` sobre o arquivo ilegível juntava três
afirmações; aqui são três linhas, porque a guarda casa um fragmento por linha e
um teste por fragmento. `internal/workspace`, onde a sonda mora, entra nos
diretórios que `internal/behavior/invariants_test.go` lê.

## O que continua de fora

A metade da linha sobre `Probe` ler só `package.json` e `Makefile` — que
nenhuma outra fonte vire portão — segue sem asserção própria.
