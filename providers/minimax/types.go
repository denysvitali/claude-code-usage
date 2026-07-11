package minimax

// CodingPlanResponse represents the response from the coding_plan/remains endpoint
type CodingPlanResponse struct {
	ModelRemains []ModelRemain `json:"model_remains"`
	BaseResp     BaseResp      `json:"base_resp"`
}

// ModelRemain represents usage information for a specific model
type ModelRemain struct {
	StartTime                 int64  `json:"start_time"`
	EndTime                   int64  `json:"end_time"`
	RemainsTime               int64  `json:"remains_time"`
	CurrentIntervalTotalCount int64  `json:"current_interval_total_count"`
	CurrentIntervalUsageCount int64  `json:"current_interval_usage_count"`
	ModelName                 string `json:"model_name"`
}

// BaseResp contains the base response status
type BaseResp struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

// SubscriptionResponse represents the response from the subscription endpoint
type SubscriptionResponse struct {
	BaseResp BaseResp `json:"base_resp"`
	// Add other fields as needed based on actual API response
}
