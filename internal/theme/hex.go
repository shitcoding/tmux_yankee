package theme

import (
	"strconv"
	"strings"
)

// ParseHex parses a "#rrggbb" (or "rrggbb") string into r, g, b components in
// [0,255]. ok is false if the string is not a valid 6-digit hex color.
func ParseHex(hex string) (r, g, b int, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	rv, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	gv, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	bv, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(rv), int(gv), int(bv), true
}
