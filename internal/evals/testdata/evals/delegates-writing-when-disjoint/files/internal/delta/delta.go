// Package delta is one of five that do not know about each other.
package delta

// Config is what delta is given when it starts.
type Config struct {
	Name    string
	Retries int
}

// Run does the one thing delta does, and returns what it produced.
func Run(cfg Config) (string, error) {
	if cfg.Name == "" {
		return "", errEmpty
	}
	return cfg.Name + ":delta", nil
}
