# Slugify

`stats` needs a `Slugify` helper so report titles can be used as file names.

A slug is lower case. Spaces become hyphens. Anything that is not a letter, a
digit or a hyphen is dropped. It never starts or ends with a hyphen.

Fast enough not to matter: this runs once per report.
