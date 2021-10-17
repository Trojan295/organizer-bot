package commands

import (
	"context"
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
	Add(ctx context.Context, channelID string, r *reminder.Reminder) (string, error)
	List(ctx context.Context, channelID string) ([]string, error)
	Get(ctx context.Context, channelID, reminderID string) (*reminder.Reminder, error)
	Remove(ctx context.Context, channelID, reminderID string) error
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
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove a reminder",
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

func (m *ReminderModule) GetComponentHandlers() map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"reminder_remove": m.reminderRemoveComponentHandler,
	}
}

func (m *ReminderModule) reminderHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if len(i.ApplicationCommandData().Options) == 0 {
		clientErrorCommandHandler(s, i, "Invalid command")
		return
	}

	switch i.ApplicationCommandData().Options[0].Name {
	case "add":
		m.reminderAddHandler(ctx, s, i, i.ApplicationCommandData().Options[0])
	case "show":
		m.reminderShowHandler(ctx, s, i)
	case "remove":
		m.reminderRemoveCommandHandler(ctx, s, i)
	default:
		unknownCommandHandler(s, i)
	}
}

func (m *ReminderModule) reminderAddHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) {
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

	_, err = m.reminderRepository.Add(ctx, channelID, &reminder.Reminder{
		Title: title,
		Date:  &datetime,
	})
	if err != nil {
		logrus.WithError(err).Errorf("failed to get reminders")
		serverErrorCommandHandler(s, i)
		return
	}

	stringResponseHandler(s, i, fmt.Sprintf("🚀 **Reminder added!**\n%s at %s", title, datetime))
}

func (m *ReminderModule) reminderShowHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	channelID := i.ChannelID
	logrus := logrus.WithField("channelID", channelID)

	reminderIDs, err := m.reminderRepository.List(ctx, i.ChannelID)
	if err != nil {
		logrus.WithError(err).Errorf("failed to get reminders")
		serverErrorCommandHandler(s, i)
		return
	}

	builder := strings.Builder{}
	builder.WriteString("⏰ **Reminders:**\n")

	for _, ID := range reminderIDs {
		reminder, err := m.reminderRepository.Get(ctx, channelID, ID)
		if err != nil {
			logrus.WithError(err).Errorf("failed to get reminder")
			serverErrorCommandHandler(s, i)
			return
		}

		builder.WriteString(fmt.Sprintf("**%s:** %s\n", reminder.Date.Format(datetimeFormat), reminder.Title))
	}

	stringResponseHandler(s, i, builder.String())
}

func (m *ReminderModule) reminderRemoveCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	var options []discordgo.SelectMenuOption

	IDs, err := m.reminderRepository.List(ctx, i.ChannelID)
	if err != nil {
		logrus.WithError(err).Error("failed to list reminders")
		serverErrorCommandHandler(s, i)
		return
	}

	for _, ID := range IDs {
		reminder, err := m.reminderRepository.Get(ctx, i.ChannelID, ID)
		if err != nil {
			logrus.WithError(err).Errorf("failed to get reminder")
			serverErrorCommandHandler(s, i)
			return
		}

		options = append(options, discordgo.SelectMenuOption{
			Label:       reminder.Title,
			Description: reminder.Date.Format(datetimeFormat),
			Value:       ID,
		})
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Select the reminder to remove:",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID: "reminder_remove",
							Options:  options,
						},
					},
				},
			},
		},
	})
	if err != nil {
		logrus.WithError(err).Errorf("cannot respond")
	}
}

func (m *ReminderModule) reminderRemoveComponentHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	reminderID := i.MessageComponentData().Values[0]
	if err := m.reminderRepository.Remove(context.TODO(), i.ChannelID, reminderID); err != nil {
		logrus.WithError(err).Error("failed to delete reminder")
		serverErrorCommandHandler(s, i)
		return
	}

	stringResponseHandler(s, i, "Reminder removed!")
}
