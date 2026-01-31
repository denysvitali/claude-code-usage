package kimi

// UsageResponse represents the response from the Kimi usage endpoint
type UsageResponse struct {
	Usages []UsageItem `json:"usages"`
}

// UsageItem represents a single usage entry for a specific scope
type UsageItem struct {
	Scope  string      `json:"scope"`
	Detail UsageDetail `json:"detail"`
	Limits []LimitItem `json:"limits"`
}

// UsageDetail contains the usage information for a scope
type UsageDetail struct {
	Limit     string `json:"limit"`
	Used      string `json:"used"`
	ResetTime string `json:"resetTime"`
}

// LimitItem represents a rate limit window
type LimitItem struct {
	Window WindowDetail `json:"window"`
	Detail UsageDetail  `json:"detail"`
}

// WindowDetail contains the window configuration
type WindowDetail struct {
	Duration int    `json:"duration"`
	TimeUnit string `json:"timeUnit"`
}

