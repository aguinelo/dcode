## gotcha: make build falha sem make generate antes
<!-- learned 2026-01-01 · commit eva1c0m -->

`make build` quebra com `undefined: Label` quando os arquivos gerados estão
velhos. Rode `make generate` primeiro, sempre.
