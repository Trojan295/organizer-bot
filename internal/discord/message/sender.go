package message

import (
	"context"
	"fmt"

	"github.com/Trojan295/organizer-bot/internal/reminder"
	"github.com/bwmarrin/discordgo"
)

type Sender struct {
	session *discordgo.Session
}

func NewSender(ds *discordgo.Session) *Sender {
	return &Sender{
		session: ds,
	}
}

func (send *Sender) PushReminder(ctx context.Context, rem *reminder.Reminder) error {
	msg := fmt.Sprintf(`🚨 **Reminder!** <#%s>
%s
`, rem.ChannelID, rem.Title)

	_, err := send.session.ChannelMessageSendComplex(rem.ChannelID, &discordgo.MessageSend{
		Content: msg,
	})
	if err != nil {
		return err
	}

	return nil
}
