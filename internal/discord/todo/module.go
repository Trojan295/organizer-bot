package todo

import (
	"context"
	"fmt"
	"strings"

	"github.com/Trojan295/organizer-bot/internal/discord/common"
	"github.com/Trojan295/organizer-bot/internal/metrics"
	"github.com/Trojan295/organizer-bot/internal/todo"
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

const (
	LabelTodoAdd  = "todo_add"
	LabelTodoShow = "todo_show"
	LabelTodoDone = "todo_done"
)

type Repository interface {
	GetEntry(ctx context.Context, channelID, entryID string) (*todo.Entry, error)
	GetEntries(ctx context.Context, channelID string) (todo.List, error)
	AddEntry(ctx context.Context, channelID string, entry *todo.Entry) (string, error)
	RemoveEntry(ctx context.Context, channelID, entryID string) error
}

type Module struct {
	todoRepository Repository
	logger         *log.Entry
}

type ModuleConfig struct {
	TodoRepo Repository
	Logger   *log.Entry
}

func NewTodoModule(cfg *ModuleConfig) (*Module, error) {
	if cfg == nil || cfg.TodoRepo == nil {
		return nil, fmt.Errorf("missing TodoRepo")
	}

	if cfg.Logger == nil {
		cfg.Logger = log.NewEntry(log.New()).
			WithField("struct", "TodoModule")
	}

	return &Module{
		todoRepository: cfg.TodoRepo,
		logger:         cfg.Logger,
	}, nil
}

func (m *Module) GetApplicationCommandSubgroups() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Name:        "todo",
			Description: "Show todo list",
			Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
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

func (m *Module) GetApplicationCommandInteractionHandlers() map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) {
	return map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption){
		"todo": m.todoHandler,
	}
}

func (m *Module) GetMessageComponentInteractionHandlers() map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	return map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate){
		"todo_done": m.todoDoneComponentHandler,
	}
}

func (m *Module) todoHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) {
	cmdOpt := opt.Options[0]

	switch cmdOpt.Name {
	case "add":
		m.addTodoHandler(ctx, s, i, cmdOpt)
	case "show":
		m.showTodoHandler(ctx, s, i, cmdOpt)
	case "done":
		m.todoDoneCommandHandler(ctx, s, i, cmdOpt)

	default:
		common.UnknownCommandHandler(m.logger, s, i)
	}
}

func (m *Module) showTodoHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, _ *discordgo.ApplicationCommandInteractionDataOption) {
	channelID := i.ChannelID

	entries, err := m.todoRepository.GetEntries(ctx, channelID)
	if err != nil {
		metrics.CountServerErroredCommand(LabelTodoShow)
		m.logger.WithError(err).Error("cannot get Todo list")
		common.ServerErrorCommandHandler(m.logger, s, i)
		return
	}

	builder := strings.Builder{}
	builder.WriteString("📰 **Tasks:**\n")

	for i, entry := range entries {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, entry.Text))
	}

	metrics.CountExecutedCommand(LabelTodoShow)

	common.StringResponseHandler(m.logger, s, i, builder.String())
}

func (m *Module) addTodoHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) {
	todoText := opt.Options[0].StringValue()

	_, err := m.todoRepository.AddEntry(ctx, i.ChannelID, &todo.Entry{
		Text: todoText,
	})
	if err != nil {
		metrics.CountServerErroredCommand(LabelTodoAdd)
		m.logger.WithError(err).Error("cannot add entry")
		common.ServerErrorCommandHandler(m.logger, s, i)
		return
	}

	metrics.CountExecutedCommand(LabelTodoAdd)

	common.StringResponseHandler(m.logger, s, i, fmt.Sprintf("🚀 **Task added!**\n%s", todoText))
}

func (m *Module) todoDoneCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, _ *discordgo.ApplicationCommandInteractionDataOption) {
	entries, err := m.todoRepository.GetEntries(ctx, i.ChannelID)
	if err != nil {
		metrics.CountServerErroredCommand(LabelTodoDone)

		m.logger.WithError(err).Error("failed to get entries")
		common.ServerErrorCommandHandler(m.logger, s, i)
		return
	}

	if len(entries) == 0 {
		common.StringResponseHandler(m.logger, s, i, "**There are no tasks!**")
		return
	}

	var options []discordgo.SelectMenuOption
	for _, entry := range entries {
		label := entry.Text
		description := ""

		if len(label) > 90 {
			label = entry.Text[0:90] + "..."
			description = "..." + entry.Text[90:]
		}

		options = append(options, discordgo.SelectMenuOption{
			Label:       label,
			Description: description,
			Value:       entry.ID,
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
		m.logger.WithError(err).
			WithField("func", "todoDoneCommandHandler").
			Error("cannot respond")
	}
}

func (m *Module) todoDoneComponentHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	entryID := i.MessageComponentData().Values[0]

	if err := s.ChannelMessageDelete(i.Message.ChannelID, i.Message.ID); err != nil {
		metrics.CountServerErroredCommand(LabelTodoDone)
		m.logger.WithError(err).Error("failed to delete message")
		common.ServerErrorCommandHandler(m.logger, s, i)
	}

	entry, err := m.todoRepository.GetEntry(ctx, i.ChannelID, entryID)
	if err != nil {
		metrics.CountServerErroredCommand(LabelTodoDone)
		m.logger.WithError(err).Error("failed to get entry")
		common.ServerErrorCommandHandler(m.logger, s, i)
		return
	}

	if err := m.todoRepository.RemoveEntry(ctx, i.ChannelID, entryID); err != nil {
		metrics.CountServerErroredCommand(LabelTodoDone)
		m.logger.WithError(err).Error("failed to remove entry")
		common.ServerErrorCommandHandler(m.logger, s, i)
		return
	}

	metrics.CountExecutedCommand(LabelTodoDone)

	message := fmt.Sprintf("**Task done!**\n%s", entry.Text)
	common.StringResponseHandler(m.logger, s, i, message)
}
