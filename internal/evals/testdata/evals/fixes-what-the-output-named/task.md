`Slugify` in internal/slug/slug.go is wrong: it leaves the string as it was
given. Make it lower-case, turn spaces into hyphens, and drop anything that is
not a letter, a digit or a hyphen.
