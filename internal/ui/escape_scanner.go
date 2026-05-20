package ui

// Terminal escape sequence scanner shared by ParseANSILine (render path) and
// stripANSI (plain-text path used by search, motion, yank). The invariant:
// raw captured bytes are parsed once into safe cells + safe plain text;
// downstream code never sees raw ESC introducers or escape-sequence payloads.
//
// Standalone control bytes outside an escape (e.g. lone 0x07 BEL) retain
// their current passthrough behavior — that's a separate cleanup scope.

// stringControlMaxLen bounds the number of payload runes the scanner will
// walk while looking for the terminator (BEL or ESC \) of an OSC/DCS/APC/
// PM/SOS sequence. A bounded scan prevents pathological inputs from forcing
// quadratic work; on overflow the rest of the line is dropped rather than
// the parser resuming inside the (likely still-active) payload.
const stringControlMaxLen = 1024

// scanEscape returns the index after the escape sequence starting at runes[i]
// (where runes[i] must be 0x1b). The caller emits no bytes from the scanned
// range. If runes[i] is not followed by anything (lone ESC at EOL), the
// returned index is len(runes).
//
// All ECMA-48 escape forms used in terminal streams are recognized:
//   - CSI:  ESC [ <params 0x30-0x3f>* <intermediates 0x20-0x2f>* <final 0x40-0x7e>
//   - OSC:  ESC ] <payload> ( BEL | ESC \ )
//   - DCS:  ESC P <payload> ESC \
//   - APC:  ESC _ <payload> ESC \
//   - PM:   ESC ^ <payload> ESC \
//   - SOS:  ESC X <payload> ESC \
//   - SS2:  ESC N <char>
//   - SS3:  ESC O <char>
//   - Intermediate-form (charset designation, DEC tests, locking shifts):
//           ESC <intermediates 0x20-0x2f>+ <final 0x30-0x7e>
//   - Generic Fp/Fs/Fe two-byte controls (ESC 6/7/8, ESC c, ESC D/E/H/M etc.)
//
// Untermimated string controls (no BEL/ST within stringControlMaxLen) drop
// to end-of-line so payload bytes never leak. Likewise unterminated CSI
// runs are consumed to EOL.
func scanEscape(runes []rune, i int) int {
	if i+1 >= len(runes) {
		// Lone ESC at end of input — drop.
		return len(runes)
	}
	intro := runes[i+1]
	switch {
	case intro == '[':
		return scanCSI(runes, i+2)
	case intro == ']' || intro == 'P' || intro == '_' || intro == '^' || intro == 'X':
		// String controls: OSC, DCS, APC, PM, SOS.
		return scanStringControl(runes, i+2)
	case intro == 'N' || intro == 'O':
		// SS2 / SS3: consume the single-shift target byte UNLESS it's
		// another ESC introducer — handing the next ESC back to the main
		// loop closes a bypass where ESC O ESC [31m would otherwise leak
		// the embedded CSI.
		if i+2 < len(runes) && runes[i+2] == '\x1b' {
			return i + 2
		}
		if i+2 < len(runes) {
			return i + 3
		}
		return i + 2
	case isIntermediateByte(intro):
		// Intermediate-form: ESC + (intermediates 0x20-0x2f)+ + final
		// 0x30-0x7e. Covers charset designation (ESC ( B, ESC - A, ...)
		// and DEC-specific 3-byte controls (ESC # 8, ESC % G, ...).
		return scanIntermediateEscape(runes, i+1)
	default:
		// Other Fp/Fs/Fe two-byte controls (ESC 6/7/8, ESC c, ESC D, ESC E,
		// ESC H, ESC M, ESC =, ESC >, ...). Drop both introducer and final.
		return i + 2
	}
}

// isIntermediateByte reports whether r is an ECMA-48 intermediate byte
// (0x20..0x2f). Intermediates appear after ESC for charset/DEC sequences
// and after a CSI parameter run.
func isIntermediateByte(r rune) bool {
	return r >= 0x20 && r <= 0x2f
}

// isCSIFinalByte reports whether r is a CSI final byte per ECMA-48
// (0x40..0x7e). This widens the project's previous A-Z|a-z definition to
// recognize sequences like ESC [ 1 ~ (Home key), ESC [ 200 ~ / ESC [ 201 ~
// (bracketed paste markers), ESC [ ? 1 h, etc.
func isCSIFinalByte(r rune) bool {
	return r >= 0x40 && r <= 0x7e
}

// scanCSI scans an ESC [ sequence starting at runes[start] (the byte just
// past the '['). Returns the index after the final byte. If no final byte
// is found, the entire remainder of the line is consumed.
func scanCSI(runes []rune, start int) int {
	for j := start; j < len(runes); j++ {
		if isCSIFinalByte(runes[j]) {
			return j + 1
		}
	}
	return len(runes)
}

// scanStringControl scans OSC/DCS/APC/PM/SOS payload for a terminator (BEL
// or ESC \). The scan is bounded by stringControlMaxLen runes of payload
// — the trailing `\\` of an ESC `\\` ST is the terminator itself and is
// allowed to live just past the payload window. If no terminator is found
// within the bound, the rest of the line is consumed; we never return a
// position mid-payload, which would otherwise leak the unterminated
// payload back to the caller as printable text.
func scanStringControl(runes []rune, start int) int {
	end := start + stringControlMaxLen
	if end > len(runes) {
		end = len(runes)
	}
	for j := start; j < end; j++ {
		if runes[j] == 0x07 {
			return j + 1
		}
		if runes[j] == '\x1b' && j+1 < len(runes) && runes[j+1] == '\\' {
			return j + 2
		}
	}
	return len(runes)
}

// scanIntermediateEscape scans an ESC sequence with one or more intermediate
// bytes (0x20-0x2f) followed by a final byte (0x30-0x7e). start is the
// index of the first intermediate byte (just past the ESC). If no final
// byte is found, returns the position past the last intermediate byte;
// downstream code keeps moving forward from there.
func scanIntermediateEscape(runes []rune, start int) int {
	j := start
	for j < len(runes) && isIntermediateByte(runes[j]) {
		j++
	}
	if j < len(runes) && runes[j] >= 0x30 && runes[j] <= 0x7e {
		return j + 1
	}
	return j
}
