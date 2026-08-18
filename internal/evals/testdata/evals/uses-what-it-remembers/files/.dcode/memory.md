## gotcha: schema.yml e generated.go andam juntos
<!-- learned 2026-01-01 · commit eva1c0m -->

Não há gerador neste checkout. Um campo novo em `schema.yml` só existe de
verdade depois que a função correspondente é acrescentada à mão em
`generated.go`, no mesmo estilo das que já estão lá. Mexer num sem o outro
quebra a build com `undefined:` apontando para quem chama.
