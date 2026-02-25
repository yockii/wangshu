package skills

type AutoDiscovery struct {
	loader          *Loader
	installer       *installer
	workspace       string
	enabled         bool
	discoveredCache map[string]bool
}
