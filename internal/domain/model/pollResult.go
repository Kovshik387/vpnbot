package model

import "time"

type PollResult struct {
	PollID         string
	Question       string
	IsAnonymous    bool
	AllowsMultiple bool
	CreatedAt      time.Time
	Options        []PollOptionResult
}
