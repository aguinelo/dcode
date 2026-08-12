package config

import "os"

// FromEnv reads the environment layer. Wins over the file.
func FromEnv(key string) string { return os.Getenv("DCODE_" + key) }
