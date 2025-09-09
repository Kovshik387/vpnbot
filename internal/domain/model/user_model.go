package model

type User struct {
	Username      string   `json:"username"`
	SubscribedUrl string   `json:"subscription_url"`
	Status        string   `json:"status"`
	OnlineAt      string   `json:"online_at"`
	Links         []string `json:"links"`
	Admin         Admin    `json:"admin"`
	UsedTraffic   int64    `json:"used_traffic"`
}

type Admin struct {
	UserUsage int64 `json:"user_usage"`
}

type UsersResponse struct {
	Users []User `json:"users"`
}
