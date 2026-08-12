package config

// Resolve applies the layers in order: flags, environment, file, defaults.
// The first layer that answers wins, and nothing merges.
func Resolve(key string) string {
	for _, layer := range []func(string) string{FromFlags, FromEnv, FromFile} {
		if v := layer(key); v != "" {
			return v
		}
	}
	return Defaults[key]
}
