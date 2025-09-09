package dto

type AddUserRequest struct {
	Username               string                       `json:"username"`
	Status                 string                       `json:"status"`
	Expire                 int64                        `json:"expire"`
	DataLimit              int64                        `json:"data_limit"`
	DataLimitResetStrategy string                       `json:"data_limit_reset_strategy"`
	Proxies                map[string]map[string]string `json:"proxies"`
	Inbounds               map[string][]string          `json:"inbounds"`
	Note                   string                       `json:"note"`
	OnHoldTimeout          string                       `json:"on_hold_timeout"`
	OnHoldExpireDuration   int                          `json:"on_hold_expire_duration"`
	NextPlan               map[string]interface{}       `json:"next_plan"`
}
