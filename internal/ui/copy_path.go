package ui

import (
	"fmt"
	"os"
	"path/filepath"
)

// statExists wraps os.Stat into the signature resolveCopyScriptPath expects:
// nil error means the path exists, otherwise it does not.
func statExists(p string) error {
	_, err := os.Stat(p)
	return err
}

// copyScriptCandidates returns the candidate locations for copy_stdin.sh,
// in priority order, given the running binary's path.
//
// Security property: no returned candidate ever resolves to a path under the
// caller's working directory. When execPath is absolute (the normal case via
// os.Executable()), the binDir-derived candidate is included. When execPath
// is relative or empty (anomalous), the binDir derivation is skipped entirely
// so it can't fall through to a CWD-relative path.
func copyScriptCandidates(execPath string) []string {
	const systemPath = "/usr/local/bin/copy_stdin.sh"
	if !filepath.IsAbs(execPath) {
		return []string{systemPath}
	}
	binDir := filepath.Dir(execPath)
	return []string{
		filepath.Join(binDir, "..", "scripts", "copy_stdin.sh"), // adjacent to installed binary
		systemPath, // system-wide install
	}
}

// resolveCopyScriptPath picks the first existing candidate from
// copyScriptCandidates. statFn returns nil if the path exists (typically
// wrap os.Stat). Returns an error listing the searched candidates when
// none exist.
func resolveCopyScriptPath(execPath string, statFn func(string) error) (string, error) {
	candidates := copyScriptCandidates(execPath)
	for _, p := range candidates {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if statFn(abs) == nil {
			return abs, nil
		}
	}
	return "", fmt.Errorf("copy_stdin.sh not found in any of: %v", candidates)
}
