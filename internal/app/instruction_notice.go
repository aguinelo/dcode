package app

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/aguinelo/dcode/internal/config"
)

// ForeignDefault are the instruction files treated as a shared format, and
// therefore as candidates for translation.
//
// DCODE.md is never here: it is the dcode-specific file, the output of the
// translation rather than an input to it.
var ForeignDefault = []string{"AGENTS.md", "CLAUDE.md"}

// InstructionNotice is what the session says at the start about instruction
// files it read that were not written for dcode.
//
// It does NOT block. Requiring setup before answering "what does this function
// do" in a freshly cloned repository is the bureaucratic tool RN-9 names, and a
// gate that stops you is a gate you learn to walk through without reading.
//
// The warning is enough, because the problem today is not that dcode cannot
// decide — it is that nobody knows this is happening.
func InstructionNotice(workspace string, foreign []string, toolNames []string) string {
	fsys := os.DirFS(workspace)

	dcodeMD, err := fs.ReadFile(fsys, "DCODE.md")
	if err == nil {
		// A DCODE.md exists. The only thing left to say is whether the sources
		// it was generated from have moved — and if so, which.
		names, diverged := config.Diverged(string(dcodeMD), fsys)
		if !diverged {
			return ""
		}
		return fmt.Sprintf(
			"DCODE.md was generated from %s, which %s changed since. "+
				"It has not been touched — it is yours now. Re-run /init if you want the change carried over.",
			strings.Join(names, " and "), plural(len(names)))
	}

	var present []string
	var bytesTotal int
	var absent []config.Finding
	for _, name := range foreign {
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			continue
		}
		present = append(present, name)
		bytesTotal += len(data)
		absent = append(absent, config.VerifyTools(string(data), toolNames)...)
		absent = append(absent, config.ProbeCommands(string(data), fsys)...)
	}
	if len(present) == 0 {
		return ""
	}

	msg := fmt.Sprintf("%s %s %d bytes of instructions written for another tool",
		strings.Join(present, " and "), verb(len(present)), bytesTotal)
	if n := len(absent); n > 0 {
		msg += fmt.Sprintf(", naming %d thing%s that do not exist here", n, ess(n))
	}
	return msg + ". /init translates them into a DCODE.md you can review."
}

func plural(n int) string {
	if n == 1 {
		return "has"
	}
	return "have"
}

func verb(n int) string {
	if n == 1 {
		return "carries"
	}
	return "carry"
}

func ess(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// foreignFiles resolves the configured list.
func foreignFiles(configured string) []string {
	if strings.TrimSpace(configured) == "" {
		return ForeignDefault
	}
	var out []string
	for _, p := range strings.Split(configured, ",") {
		if p = strings.TrimSpace(p); p != "" && filepath.Base(p) != "DCODE.md" {
			out = append(out, p)
		}
	}
	return out
}
