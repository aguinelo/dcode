# example

`Label` and the other identifiers in `generated.go` come from `schema.yml`.
Regenerate them with `make generate` after any schema change, or the package
stops compiling with `undefined:` errors that point at the caller rather than
at the stale file.
