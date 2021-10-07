package commands

import (
	"fmt"
	"strings"

	"github.com/Trojan295/organizer-bot/internal/todo"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

var (
	repo todo.InMemoryListRepository
)

type NewTodoApplicationCommandInput struct {
}

func NewTodoApplicationCommands(input *NewTodoApplicationCommandInput) []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "todo",
			Description: "Show todo list",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add an todo item",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "msg",
							Description: "Message",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "show",
					Description: "Show todo list",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "done",
					Description: "Mark task as done",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "task",
							Description: "Task to mark as done",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func TodoCommandHandlers() map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"todo": todoHandler,
	}
}

func todoHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if len(i.ApplicationCommandData().Options) == 0 {
		showTodoHandler(s, i)
		return
	}

	switch i.ApplicationCommandData().Options[0].Name {
	case "add":
		addTodoHandler(s, i, i.ApplicationCommandData().Options[0])
	case "show":
		showTodoHandler(s, i)

	default:
		unknownCommandHandler(s, i)
	}
}

func showTodoHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	list, _ := repo.Get(i.ChannelID)

	builder := strings.Builder{}
	builder.WriteString("This is TODO:\n")

	for _, entry := range list.Entries {
		if entry.Done {
			builder.WriteString(fmt.Sprintf("✓ %s\n", entry.Text))
		} else {
			builder.WriteString(fmt.Sprintf("⃞ %s\n", entry.Text))
		}
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: builder.String(),
		},
	})

	if err != nil {
		logrus.WithError(err).Errorf("cannot respond to todo show")
	}
}

func addTodoHandler(s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) {
	message := opt.Options[0].StringValue()

	list, _ := repo.Get(i.ChannelID)
	list.Entries = append(list.Entries, todo.Entry{Text: message, Done: false})
	repo.Save(i.ChannelID, list)

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🚀 Task was added!",
		},
	})

	for _, cmd := range GetApplicationCommands(&GetApplicationCommandInput{
		ChannelID: &i.ChannelID,
	}) {
		logrus.Println(cmd.Options[2].Options[0].Choices)
		if _, err := s.ApplicationCommandCreate(s.State.User.ID, i.GuildID, cmd); err != nil {
			logrus.WithError(err).WithField("name", cmd.Name).Errorf("while adding application commands")
		}
	}

	if err != nil {
		logrus.WithError(err).Errorf("cannot respond to todo add")
	}
}

func markTodoHandler(s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{})

	if err != nil {
		logrus.WithError(err).Errorf("cannot respond to todo add")
	}
}
