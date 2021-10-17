package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/Trojan295/organizer-bot/internal/reminder"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

const (
	datetimeFormat = "02.01.2006 15:04"
)

type ReminderRepository interface {
	Get(ID string) (*reminder.Reminders, error)
	Save(ID string, l *reminder.Reminders) error
}

type ReminderModule struct {
	reminderRepository ReminderRepository
}

func NewReminderModule(repo ReminderRepository) *ReminderModule {
	return &ReminderModule{
		reminderRepository: repo,
	}
}

func (m *ReminderModule) GetApplicationCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "reminder",
			Description: "Manage reminders",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add new reminder",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "date",
							Description: "Date, e.g. 20.12.2021 15:48",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "text",
							Description: "Text",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "show",
					Description: "Show set reminders",
				},
			},
		},
	}
}

func (m *ReminderModule) GetCommandCreateHandlers() map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"reminder": m.reminderHandler,
	}
}

func (m *ReminderModule) reminderHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if len(i.ApplicationCommandData().Options) == 0 {
		clientErrorCommandHandler(s, i, "Invalid command")
		return
	}

	switch i.ApplicationCommandData().Options[0].Name {
	case "add":
		m.reminderAddHandler(s, i, i.ApplicationCommandData().Options[0])
	case "show":
		m.reminderShowHandler(s, i)
	default:
		unknownCommandHandler(s, i)
	}
}

func (m *ReminderModule) reminderAddHandler(s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) {
	channelID := i.ChannelID
	logrus := logrus.WithField("channelID", channelID)

	datetimeStr := opt.Options[0].StringValue()
	// TODO: consider timezones
	datetime, err := time.Parse(datetimeFormat, datetimeStr)
	if err != nil {
		clientErrorCommandHandler(s, i, `Date is wrong. Should be in "day.month.year hour:minute" format.`)
		return
	}

	title := opt.Options[1].StringValue()

	reminders, err := m.reminderRepository.Get(i.ChannelID)
	if err != nil {
		logrus.WithError(err).Errorf("failed to get reminders")
		serverErrorCommandHandler(s, i)
		return
	}

	reminders.Items = append(reminders.Items, reminder.Item{
		Title: title,
		Date:  &datetime,
	})

	if err := m.reminderRepository.Save(i.ChannelID, reminders); err != nil {
		logrus.WithError(err).Errorf("failed to save reminders")
		serverErrorCommandHandler(s, i)
	}

	stringResponseHandler(s, i, fmt.Sprintf("🚀 **Reminder added!**\n%s at %s", title, datetime))
}

func (m *ReminderModule) reminderShowHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channelID := i.ChannelID
	logrus := logrus.WithField("channelID", channelID)

	reminders, err := m.reminderRepository.Get(i.ChannelID)
	if err != nil {
		logrus.WithError(err).Errorf("failed to get reminders")
		serverErrorCommandHandler(s, i)
		return
	}

	builder := strings.Builder{}
	builder.WriteString("⏰ **Reminders:**\n")

	for i, item := range reminders.Items {
		builder.WriteString(fmt.Sprintf("**%d.** %s - %s\n", i+1, item.Date.Format(datetimeFormat), item.Title))
	}

	stringResponseHandler(s, i, builder.String())
}
