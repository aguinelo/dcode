# example

`generated.go` carries the strings the report is built from. Its header says it
is generated, and it was — but there is no generator in this checkout and no
plan to add one. When the report needs a new string, the function is written
into `generated.go` by hand, in the style of the ones already there, and
`schema.yml` is updated to match.

Nothing enforces that pairing. Forgetting either half is how the package stops
compiling with an `undefined:` that points at the caller.
