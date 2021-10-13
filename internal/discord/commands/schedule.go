package commands

import (
	"fmt"
	"time"

	"github.com/Trojan295/organizer-bot/internal/schedule"
	"github.com/bwmarrin/discordgo"
)

type ScheduleRepository interface {
	Get(ID string) (*schedule.Schedule, error)
	Save(ID string, l *schedule.Schedule) error
}

type ScheduleModule struct {
	scheduleRepository ScheduleRepository
}

func NewScheduleModule(repo ScheduleRepository) *ScheduleModule {
	return &ScheduleModule{
		scheduleRepository: repo,
	}
}

func (m *ScheduleModule) GetApplicationCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "schedule",
			Description: "Schedule tasks",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Schedule new task",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "date",
							Description: "Date, e.g. 20.12.2021 15:48",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "title",
							Description: "Title",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (m *ScheduleModule) GetCommandCreateHandlers() map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"schedule": m.scheduleHandler,
	}
}

func (m *ScheduleModule) scheduleHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if len(i.ApplicationCommandData().Options) == 0 {
		clientErrorCommandHandler(s, i, "Invalid command")
		return
	}

	switch i.ApplicationCommandData().Options[0].Name {
	case "add":
		m.scheduleAddHandler(s, i, i.ApplicationCommandData().Options[0])
	default:
		unknownCommandHandler(s, i)
	}
}

func (m *ScheduleModule) scheduleAddHandler(s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) {
	// TODO: consider timezones
	datetimeStr := opt.Options[0].StringValue()
	datetime, err := time.Parse("02.01.2006 15:04", datetimeStr)
	if err != nil {
		clientErrorCommandHandler(s, i, `Date is wrong. Should be in "day.month.year hour:minute" format.`)
		return
	}

	title := opt.Options[1].StringValue()

	stringResponseHandler(s, i, fmt.Sprintf("%v - %v", datetime, title))
}
