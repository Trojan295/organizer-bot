package message

import (
	"context"
	"fmt"
	"strings"

	"github.com/Trojan295/organizer-bot/internal/reminder"
	"github.com/Trojan295/organizer-bot/internal/todo"
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

func (send *Sender) PushTodoListNotification(ctx context.Context, list *todo.List) error {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("📰 **Tasks:** <#%s>\n", list.ChannelID))

	for i, entry := range list.Entries {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, entry.Text))
	}

	_, err := send.session.ChannelMessageSendComplex(list.ChannelID, &discordgo.MessageSend{
		Content: builder.String(),
	})
	if err != nil {
		return err
	}

	return nil
}
