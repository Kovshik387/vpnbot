package admin

import (
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/model"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func HandlePollUpdate(update tgbotapi.Update, userUC *usecases.UserUsecase) {
	if update.Poll == nil {
		return
	}
	if err := userUC.UpdatePollResults(pollSnapshot(update.Poll)); err != nil {
		log.Printf("failed to update poll %s: %v", update.Poll.ID, err)
	}
}

func pollSnapshot(p *tgbotapi.Poll) *model.PollResult {
	res := &model.PollResult{
		PollID:          p.ID,
		Question:        p.Question,
		IsAnonymous:     p.IsAnonymous,
		AllowsMultiple:  p.AllowsMultipleAnswers,
		TotalVoterCount: p.TotalVoterCount,
		IsClosed:        p.IsClosed,
	}

	options := make([]model.PollOptionResult, len(p.Options))
	for i, o := range p.Options {
		options[i] = model.PollOptionResult{
			OptionIndex: i,
			Text:        o.Text,
			Votes:       o.VoterCount,
		}
	}
	res.Options = options
	return res
}
