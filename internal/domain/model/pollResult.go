package model

import "time"

type PollResult struct {
	PollID          string
	Question        string
	IsAnonymous     bool
	AllowsMultiple  bool
	TotalVoterCount int
	IsClosed        bool
	ChatID          int64
	MessageID       int
	SourcePollID    string
	CreatedAt       time.Time
	Options         []PollOptionResult
}
