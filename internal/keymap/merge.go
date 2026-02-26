package keymap

// Merge applies overrides on top of a base keymap and returns a new keymap.
// Override entries with ActionNone remove the corresponding binding (unbind).
// Non-empty override entries replace existing ones.
func (base Keymap) Merge(overrides Keymap) Keymap {
	result := Keymap{
		Direct:      make(map[KeySpec]Action, len(base.Direct)),
		Prefix:      make(map[byte]map[byte]Action, len(base.Prefix)),
		CharCapture: make(map[byte]Action, len(base.CharCapture)),
		TextObjects: make(map[[2]byte]Action, len(base.TextObjects)),
	}

	// Copy base Direct bindings
	for k, v := range base.Direct {
		result.Direct[k] = v
	}
	// Apply overrides
	for k, v := range overrides.Direct {
		if v == ActionNone {
			delete(result.Direct, k)
		} else {
			result.Direct[k] = v
		}
	}

	// Copy base Prefix bindings (deep copy inner maps)
	for prefix, secondMap := range base.Prefix {
		inner := make(map[byte]Action, len(secondMap))
		for k, v := range secondMap {
			inner[k] = v
		}
		result.Prefix[prefix] = inner
	}
	// Apply prefix overrides
	for prefix, secondMap := range overrides.Prefix {
		if result.Prefix[prefix] == nil {
			result.Prefix[prefix] = make(map[byte]Action)
		}
		for k, v := range secondMap {
			if v == ActionNone {
				delete(result.Prefix[prefix], k)
			} else {
				result.Prefix[prefix][k] = v
			}
		}
		// Clean up empty prefix groups
		if len(result.Prefix[prefix]) == 0 {
			delete(result.Prefix, prefix)
		}
	}

	// Copy base CharCapture bindings
	for k, v := range base.CharCapture {
		result.CharCapture[k] = v
	}
	// Apply overrides
	for k, v := range overrides.CharCapture {
		if v == ActionNone {
			delete(result.CharCapture, k)
		} else {
			result.CharCapture[k] = v
		}
	}

	// Copy base TextObjects bindings
	for k, v := range base.TextObjects {
		result.TextObjects[k] = v
	}
	// Apply overrides
	for k, v := range overrides.TextObjects {
		if v == ActionNone {
			delete(result.TextObjects, k)
		} else {
			result.TextObjects[k] = v
		}
	}

	return result
}
