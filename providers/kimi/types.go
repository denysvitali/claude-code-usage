package kimi

// UsageResponse represents the response from the Kimi Code usage endpoint
// (GET https://api.kimi.com/coding/v1/usages)
type UsageResponse struct {
	Usage  UsageDetail `json:"usage"`
	Limits []LimitItem `json:"limits"`
}

// UsageDetail contains the usage information for a window
type UsageDetail struct {
	Limit     string `json:"limit"`
	Used      string `json:"used"`
	Remaining string `json:"remaining"`
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

// SubscriptionResponse represents the response from the GetSubscription endpoint
type SubscriptionResponse struct {
	Subscription *Subscription `json:"subscription"`
	Memberships  []Membership  `json:"memberships"`
	Subscribed   bool          `json:"subscribed"`
}

// Subscription contains subscription details
type Subscription struct {
	SubscriptionID   string `json:"subscriptionId"`
	Goods            Goods  `json:"goods"`
	CurrentStartTime string `json:"currentStartTime"`
	CurrentEndTime   string `json:"currentEndTime"`
	Status           string `json:"status"`
}

// Goods contains subscription plan details
type Goods struct {
	Title           string       `json:"title"`
	MembershipLevel string       `json:"membershipLevel"`
	Amounts         []Amount     `json:"amounts"`
	BillingCycle    BillingCycle `json:"billingCycle"`
}

// Amount represents a price amount
type Amount struct {
	Currency     string `json:"currency"`
	PriceInCents string `json:"priceInCents"`
}

// BillingCycle contains billing period information
type BillingCycle struct {
	Duration int    `json:"duration"`
	TimeUnit string `json:"timeUnit"`
}

// Membership represents a feature membership/quota
type Membership struct {
	Feature    string `json:"feature"`
	LeftCount  int    `json:"leftCount"`
	TotalCount int    `json:"totalCount"`
	Level      string `json:"level"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
}
