package admin

import (
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/model"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"time"
)

func HandlePollUpdate(update tgbotapi.Update, userUC *usecases.UserUsecase) {
	var p *tgbotapi.Poll

	switch {
	case update.Poll != nil:
		p = update.Poll
	case update.Message != nil && update.Message.Poll != nil:
		p = update.Message.Poll
	default:
		return
	}

	res := &model.PollResult{
		PollID:         p.ID,
		Question:       p.Question,
		IsAnonymous:    p.IsAnonymous,
		AllowsMultiple: p.AllowsMultipleAnswers,
		CreatedAt:      time.Now(),
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

	if err := userUC.SavePollResults(res); err != nil {
		log.Printf("failed to save poll results: %v", err)
	} else {
		log.Printf("poll %s saved/updated", res.PollID)
	}
}
