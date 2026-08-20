// Package echo is one of five that do not know about each other.
package echo

// Config is what echo is given when it starts.
type Config struct {
	Name    string
	Retries int
}

// Run does the one thing echo does, and returns what it produced.
func Run(cfg Config) (string, error) {
	if cfg.Name == "" {
		return "", errEmpty
	}
	return cfg.Name + ":echo", nil
}
