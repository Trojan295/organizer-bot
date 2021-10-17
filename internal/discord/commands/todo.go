package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/Trojan295/organizer-bot/internal/todo"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

type TodoRepository interface {
	GetEntry(ctx context.Context, channelID, entryID string) (*todo.Entry, error)
	GetEntries(ctx context.Context, channelID string) ([]*todo.Entry, error)
	AddEntry(ctx context.Context, channelID string, entry *todo.Entry) (string, error)
	RemoveEntry(ctx context.Context, channelID, entryID string) error
}

type TodoModule struct {
	todoRepository TodoRepository
}

func NewTodoModule(repo TodoRepository) *TodoModule {
	return &TodoModule{
		todoRepository: repo,
	}
}

func (m *TodoModule) GetApplicationCommands() []*discordgo.ApplicationCommand {
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
				},
			},
		},
	}
}

func (m *TodoModule) GetCommandCreateHandlers() map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	return map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate){
		"todo": m.todoHandler,
	}
}

func (m *TodoModule) GetComponentHandlers() map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	return map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate){
		"todo_done": m.todoDoneComponentHandler,
	}
}

func (m *TodoModule) todoHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	if len(i.ApplicationCommandData().Options) == 0 {
		m.showTodoHandler(ctx, s, i)
		return
	}

	switch i.ApplicationCommandData().Options[0].Name {
	case "add":
		m.addTodoHandler(ctx, s, i, i.ApplicationCommandData().Options[0])
	case "show":
		m.showTodoHandler(ctx, s, i)
	case "done":
		m.todoDoneCommandHandler(ctx, s, i)

	default:
		unknownCommandHandler(s, i)
	}
}

func (m *TodoModule) showTodoHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	channelID := i.ChannelID
	logrus := logrus.WithField("channelID", channelID)

	entries, err := m.todoRepository.GetEntries(ctx, channelID)
	if err != nil {
		logrus.WithError(err).Errorf("cannot get Todo list")
		serverErrorCommandHandler(s, i)
		return
	}

	builder := strings.Builder{}
	builder.WriteString("📰 **Tasks:**\n")

	for i, entry := range entries {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, entry.Text))
	}

	stringResponseHandler(s, i, builder.String())
}

func (m *TodoModule) addTodoHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) {
	channelID := i.ChannelID
	todoText := opt.Options[0].StringValue()

	logrus := logrus.WithField("channelID", channelID)

	_, err := m.todoRepository.AddEntry(ctx, i.ChannelID, &todo.Entry{
		Text: todoText,
	})
	if err != nil {
		logrus.WithError(err).Error("cannot get list")
		serverErrorCommandHandler(s, i)
		return
	}

	stringResponseHandler(s, i, fmt.Sprintf("🚀 **Task added!**\n%s", todoText))
}

func (m *TodoModule) todoDoneCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	entries, err := m.todoRepository.GetEntries(ctx, i.ChannelID)
	if err != nil {
		logrus.WithError(err).Error("failed to get entries")
		serverErrorCommandHandler(s, i)
		return
	}

	var options []discordgo.SelectMenuOption
	for _, entry := range entries {
		options = append(options, discordgo.SelectMenuOption{
			Label: entry.Text,
			Value: entry.ID,
		})
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Select the task to mark as done:",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID: "todo_done",
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

func (m *TodoModule) todoDoneComponentHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	entryID := i.MessageComponentData().Values[0]

	entry, err := m.todoRepository.GetEntry(ctx, i.ChannelID, entryID)
	if err != nil {
		logrus.WithError(err).Error("failed to get entry")
		serverErrorCommandHandler(s, i)
		return
	}

	if err := m.todoRepository.RemoveEntry(ctx, i.ChannelID, entryID); err != nil {
		logrus.WithError(err).Error("failed to remove entry")
		serverErrorCommandHandler(s, i)
		return
	}

	message := fmt.Sprintf("**Task done!**\n%s", entry.Text)
	stringResponseHandler(s, i, message)
}
