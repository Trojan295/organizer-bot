package common

import (
	"context"
	"fmt"
	"time"

	"github.com/Trojan295/organizer-bot/internal/metrics"
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

const (
	LabelConfigTimezoneSet = "config_timezone_set"
	LabelConfigTimezoneGet = "config_timezone_get"
)

type TimezoneRepository interface {
	GetCurrentTimezone(ctx context.Context, ID string) (*time.Location, error)
	SetCurrentTimezone(ctx context.Context, ID string, tz *time.Location) error
}

type ConfigModule struct {
	logger             *log.Entry
	timezoneRepository TimezoneRepository
}

type ConfigModuleInput struct {
	Logger             *log.Entry
	TimezoneRepository TimezoneRepository
}

func NewConfigModule(input *ConfigModuleInput) (*ConfigModule, error) {
	if input == nil {
		return nil, fmt.Errorf("input not provided")
	}

	if input.TimezoneRepository == nil {
		return nil, fmt.Errorf("TimezoneRepository not provided")
	}

	if input.Logger == nil {
		input.Logger = log.NewEntry(log.New())
	}

	return &ConfigModule{
		timezoneRepository: input.TimezoneRepository,
		logger:             input.Logger,
	}, nil
}

func (module *ConfigModule) GetApplicationCommandSubgroups() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
			Name:        "config",
			Description: "Configuration related commands",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "timezone",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Timezone settings",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "timezone",
							Description: "Timezone name",
						},
					},
				},
			},
		},
	}
}

func (module *ConfigModule) GetApplicationCommandInteractionHandlers() map[string]func(context.Context, *discordgo.Session, *discordgo.InteractionCreate, *discordgo.ApplicationCommandInteractionDataOption) {
	return map[string]func(context.Context, *discordgo.Session, *discordgo.InteractionCreate, *discordgo.ApplicationCommandInteractionDataOption){
		"config": module.configHandler,
	}
}

func (module *ConfigModule) GetMessageComponentInteractionHandlers() map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	return map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate){}
}

func (module *ConfigModule) configHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, cmd *discordgo.ApplicationCommandInteractionDataOption) {
	subCmd := cmd.Options[0]

	switch subCmd.Name {
	case "timezone":
		module.timezoneHandler(ctx, s, i, subCmd)

	default:
		UnknownCommandHandler(module.logger, s, i)
	}
}

func (module *ConfigModule) timezoneHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, cmd *discordgo.ApplicationCommandInteractionDataOption) {
	switch len(cmd.Options) {
	case 0:
		module.getTimezoneHandler(ctx, s, i)
	case 1:
		module.setTimezoneHandler(ctx, s, i, cmd)
	}
}

func (module *ConfigModule) getTimezoneHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	tz, err := module.timezoneRepository.GetCurrentTimezone(ctx, i.ChannelID)
	if err != nil {
		metrics.CountServerErroredCommand(LabelConfigTimezoneGet)
		ServerErrorCommandHandler(module.logger, s, i)
		return
	}

	metrics.CountExecutedCommand(LabelConfigTimezoneGet)

	if tz == nil {
		StringResponseHandler(module.logger, s, i, "🕑 Timezone is not set!")
		return
	}

	msg := fmt.Sprintf("🕑 The current timezone is:\n**%v**", tz.String())
	StringResponseHandler(module.logger, s, i, msg)
}

func (module *ConfigModule) setTimezoneHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, cmd *discordgo.ApplicationCommandInteractionDataOption) {
	tzString := cmd.Options[0].StringValue()

	location, err := time.LoadLocation(tzString)
	if err != nil {
		metrics.CountClientErroredCommand(LabelConfigTimezoneSet)
		ClientErrorCommandHandler(module.logger, s, i, `Incorrect timezone name!
Example names are: **Europe/Berlin**, **UTC** or **EST**.
You can find a list of timezone names here:
https://en.wikipedia.org/wiki/List_of_tz_database_time_zones`)
		return
	}

	if err := module.timezoneRepository.SetCurrentTimezone(ctx, i.ChannelID, location); err != nil {
		metrics.CountServerErroredCommand(LabelConfigTimezoneSet)
		module.logger.WithError(err).Error("failed to set timezone")
		ServerErrorCommandHandler(module.logger, s, i)
		return
	}

	metrics.CountExecutedCommand(LabelConfigTimezoneSet)
	StringResponseHandler(module.logger, s, i, fmt.Sprintf("🚀 Timezone set to **%s**.", location.String()))
}
