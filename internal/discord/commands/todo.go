package commands

import (
	"fmt"
	"strings"

	"github.com/Trojan295/organizer-bot/internal/todo"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

type TodoRepository interface {
	Get(ID string) (*todo.List, error)
	Save(ID string, l *todo.List) error
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
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "taskID",
							Description: "Task to mark as done",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (m *TodoModule) GetCommandCreateHandlers() map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"todo": m.todoHandler,
	}
}

func (m *TodoModule) todoHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if len(i.ApplicationCommandData().Options) == 0 {
		m.showTodoHandler(s, i)
		return
	}

	switch i.ApplicationCommandData().Options[0].Name {
	case "add":
		m.addTodoHandler(s, i, i.ApplicationCommandData().Options[0])
	case "show":
		m.showTodoHandler(s, i)
	case "done":
		m.doneTodoHandler(s, i, i.ApplicationCommandData().Options[0])

	default:
		unknownCommandHandler(s, i)
	}
}

func (m *TodoModule) showTodoHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channelID := i.ChannelID
	logrus := logrus.WithField("channelID", channelID)

	list, err := m.todoRepository.Get(channelID)
	if err != nil {
		logrus.WithError(err).Errorf("cannot get Todo list")
		serverErrorCommandHandler(s, i)
		return
	}

	builder := strings.Builder{}
	builder.WriteString("📰 **Tasks:**\n```")

	for i, entry := range list.Entries {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, entry.Text))
	}

	builder.WriteString("```")

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: builder.String(),
		},
	})
	if err != nil {
		logrus.WithError(err).Errorf("cannot respond to todo show")
	}
}

func (m *TodoModule) addTodoHandler(s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) {
	channelID := i.ChannelID
	message := opt.Options[0].StringValue()

	logrus := logrus.WithField("channelID", channelID)

	list, err := m.todoRepository.Get(i.ChannelID)
	if err != nil {
		logrus.WithError(err).Error("cannot get list")
		serverErrorCommandHandler(s, i)
		return
	}

	list.Entries = append(list.Entries, todo.Entry{Text: message})
	if err := m.todoRepository.Save(i.ChannelID, list); err != nil {
		logrus.WithError(err).Error("cannot save list")
		serverErrorCommandHandler(s, i)
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🚀 **Added task!**\n%s", message),
		},
	})

	if err != nil {
		logrus.WithError(err).Errorf("cannot respond to todo add")
	}
}

func (m *TodoModule) doneTodoHandler(s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) {
	channelID := i.ChannelID
	itemPos := int(opt.Options[0].IntValue() - 1)

	logrus := logrus.WithField("channelID", channelID)

	list, err := m.todoRepository.Get(i.ChannelID)
	if err != nil {
		logrus.WithError(err).Error("cannot get list")
		serverErrorCommandHandler(s, i)
		return
	}

	if itemPos < 0 || itemPos > len(list.Entries)-1 {
		clientErrorCommandHandler(s, i, "Wrong task number!")
		return
	}

	task := list.Entries[itemPos]

	list.Entries = append(list.Entries[0:itemPos], list.Entries[itemPos+1:]...)
	err = m.todoRepository.Save(i.ChannelID, list)
	if err != nil {
		logrus.WithError(err).Error("cannot save list")
		serverErrorCommandHandler(s, i)
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✓ **Task done!**\n%s", task.Text),
		},
	})
	if err != nil {
		logrus.WithError(err).Errorf("cannot respond to todo done")
	}
}
