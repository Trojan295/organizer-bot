package reminder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Trojan295/organizer-bot/internal/discord/common"
	"github.com/Trojan295/organizer-bot/internal/reminder"
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

const (
	datetimeFormat = "02.01.2006 15:04"
)

type Repository interface {
	AddReminder(ctx context.Context, channelID string, r *reminder.Reminder) (string, error)
	GetReminders(ctx context.Context, channelID string) ([]*reminder.Reminder, error)
	RemoveReminder(ctx context.Context, channelID, reminderID string) error
}

type TimezoneRepository interface {
	GetCurrentTimezone(ctx context.Context, ID string) (*time.Location, error)
}

type Module struct {
	reminderRepository Repository
	timezoneRepository TimezoneRepository
	logger             *log.Entry
}

type ModuleConfig struct {
	ReminderRepo       Repository
	TimezoneRepository TimezoneRepository
	Logger             *log.Entry
}

func NewReminderModule(cfg *ModuleConfig) (*Module, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cfg is missing")
	}

	if cfg.ReminderRepo == nil {
		return nil, fmt.Errorf("missing ReminderRepo")
	}

	if cfg.TimezoneRepository == nil {
		return nil, fmt.Errorf("missing TimezoneRepository")
	}

	if cfg.Logger == nil {
		cfg.Logger = log.NewEntry(log.New()).
			WithField("struct", "ReminderModule")
	}

	return &Module{
		reminderRepository: cfg.ReminderRepo,
		timezoneRepository: cfg.TimezoneRepository,
		logger:             cfg.Logger,
	}, nil
}

func (m *Module) GetApplicationCommandSubgroups() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Name:        "reminder",
			Description: "Manage reminders",
			Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add new reminder",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "date",
							Description: "Date in UTC, e.g. 20.12.2021 15:48",
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

func (m *Module) GetApplicationCommandInteractionHandlers() map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) {
	return map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption){
		"reminder": m.reminderHandler,
	}
}

func (m *Module) GetMessageComponentInteractionHandlers() map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	return map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate){
		"reminder_remove": m.reminderRemoveComponentHandler,
	}
}

func (m *Module) reminderHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) {
	cmdOpt := opt.Options[0]

	switch cmdOpt.Name {
	case "add":
		m.reminderAddHandler(ctx, s, i, cmdOpt)
	case "show":
		m.reminderShowHandler(ctx, s, i)
	case "remove":
		m.reminderRemoveCommandHandler(ctx, s, i)
	default:
		common.UnknownCommandHandler(m.logger, s, i)
	}
}

func (m *Module) reminderAddHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) {
	datetimeStr := opt.Options[0].StringValue()

	location, err := m.timezoneRepository.GetCurrentTimezone(ctx, i.ChannelID)
	if err != nil {
		m.logger.WithError(err).Error("failed to get current timezone")
		common.ServerErrorCommandHandler(m.logger, s, i)
		return
	}
	if location == nil {
		common.ClientErrorCommandHandler(m.logger, s, i, "You have to first set your timezone!\nUse `/organizer config timezone` to set the timezone.")
		return
	}

	datetime, err := time.ParseInLocation(datetimeFormat, datetimeStr, location)
	if err != nil {
		common.ClientErrorCommandHandler(m.logger, s, i, `Date is wrong. Should be in "day.month.year hour:minute" format.`)
		return
	}

	title := opt.Options[1].StringValue()

	_, err = m.reminderRepository.AddReminder(ctx, i.ChannelID, &reminder.Reminder{
		Title: title,
		Date:  &datetime,
	})
	if err != nil {
		m.logger.WithError(err).Error("failed to get reminders")
		common.ServerErrorCommandHandler(m.logger, s, i)
		return
	}

	msg := fmt.Sprintf("🚀 **Reminder added!**\n%s at %s", title, datetime.Format(datetimeFormat))

	common.StringResponseHandler(m.logger, s, i, msg)
}

func (m *Module) reminderShowHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	reminders, err := m.reminderRepository.GetReminders(ctx, i.ChannelID)
	if err != nil {
		m.logger.WithError(err).Error("failed to get reminders")
		common.ServerErrorCommandHandler(m.logger, s, i)
		return
	}

	builder := strings.Builder{}
	builder.WriteString("⏰ **Reminders:**\n")

	for _, reminder := range reminders {
		builder.WriteString(fmt.Sprintf("**%s:** %s\n", reminder.Date.Format(datetimeFormat), reminder.Title))
	}

	common.StringResponseHandler(m.logger, s, i, builder.String())
}

func (m *Module) reminderRemoveCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	var options []discordgo.SelectMenuOption

	reminders, err := m.reminderRepository.GetReminders(ctx, i.ChannelID)
	if err != nil {
		m.logger.WithError(err).Error("failed to get reminders")
		common.ServerErrorCommandHandler(m.logger, s, i)
		return
	}

	if len(reminders) == 0 {
		common.StringResponseHandler(m.logger, s, i, "**There are no reminders!**")
		return
	}

	for _, reminder := range reminders {
		options = append(options, discordgo.SelectMenuOption{
			Label:       reminder.Title,
			Description: reminder.Date.Format(datetimeFormat),
			Value:       reminder.ID,
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
		m.logger.WithError(err).
			Error("cannot respond to remove command")
	}
}

func (m *Module) reminderRemoveComponentHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	reminderID := i.MessageComponentData().Values[0]

	if err := s.ChannelMessageDelete(i.Message.ChannelID, i.Message.ID); err != nil {
		m.logger.WithError(err).Error("failed to delete message")
	}

	if err := m.reminderRepository.RemoveReminder(ctx, i.ChannelID, reminderID); err != nil {
		m.logger.WithError(err).Error("failed to delete reminder")
		common.ServerErrorCommandHandler(m.logger, s, i)
		return
	}

	common.StringResponseHandler(m.logger, s, i, "Reminder removed!")
}
