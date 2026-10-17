package app

import (
	"testing"
)

// learned-memory.config declares memory.enabled and memory.max_entries as
// configuration, and FromEnv read both — but neither was in KnownKeys, so a
// config.toml that wrote them was refused as an unknown key and the two
// DCODE_MEMORY_* variables were never looked up. The setting existed only as
// its default.

func TestMemoryKeysAreHonouredFromConfigFile(t *testing.T) {
	env := envFrom(map[string]string{"DCODE_HOME": t.TempDir()})
	ws := t.TempDir()
	projectConfig(t, ws, "[memory]\nenabled = false\nmax_entries = 7\n")

	opts, _, err := FromEnv(env, ws)
	if err != nil {
		t.Fatalf("a config.toml that sets the memory keys was refused: %v", err)
	}
	if opts.Memory {
		t.Error("memory.enabled = false in config.toml left memory on")
	}
	if opts.MemoryMax != 7 {
		t.Errorf("memory.max_entries = 7 in config.toml gave %d", opts.MemoryMax)
	}
}

func TestMemoryKeysAreHonouredFromTheEnvironment(t *testing.T) {
	env := envFrom(map[string]string{
		"DCODE_HOME":               t.TempDir(),
		"DCODE_MEMORY_ENABLED":     "false",
		"DCODE_MEMORY_MAX_ENTRIES": "9",
	})
	opts, _, err := FromEnv(env, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if opts.Memory {
		t.Error("DCODE_MEMORY_ENABLED=false left memory on")
	}
	if opts.MemoryMax != 9 {
		t.Errorf("DCODE_MEMORY_MAX_ENTRIES=9 gave %d", opts.MemoryMax)
	}
}
