package types

// Remote represents a Git remote configuration.
type Remote struct {
	// Name is the remote name (e.g., "origin").
	Name string `json:"name"`
	// URLs is the list of push URLs for this remote.
	URLs []string `json:"urls"`
	// FetchURLs is the list of fetch URLs for this remote.
	FetchURLs []string `json:"fetch_urls,omitempty"`
}
