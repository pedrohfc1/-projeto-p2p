//go:build !windows

package adapters

// Linux retains native and explicit interface modes. No unverified Hamachi API
// is assumed; identification can use the driver description if supplied later.
func List() ([]Info, error) {
	items, e := basic()
	for n := range items {
		items[n] = Classify(items[n])
	}
	return items, e
}
