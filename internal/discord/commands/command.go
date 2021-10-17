package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

const (
	unknownCommandMessage     = "Unknown command."
	serverNotAvailableMessage = "is not available right now... Try again in a moment."
)

type SlashModule interface {
	GetApplicationCommands() []*discordgo.ApplicationCommand
	GetCommandCreateHandlers() map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate)
	GetComponentHandlers() map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate)
}

type ModuleAggregator struct {
	modules []SlashModule
}

func NewModuleAggregator() *ModuleAggregator {
	return &ModuleAggregator{}
}

func (a *ModuleAggregator) AddModules(modules ...SlashModule) {
	a.modules = append(a.modules, modules...)
}

func (a *ModuleAggregator) GetApplicationCommands() []*discordgo.ApplicationCommand {
	cmds := make([]*discordgo.ApplicationCommand, 0)

	for _, m := range a.modules {
		cmds = append(cmds, m.GetApplicationCommands()...)
	}

	return cmds
}

func (a *ModuleAggregator) GetCommandCreateHandlers() map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	handlers := make(map[string]func(context.Context, *discordgo.Session, *discordgo.InteractionCreate))

	for _, m := range a.modules {
		for k, v := range m.GetCommandCreateHandlers() {
			handlers[k] = v
		}
	}

	return handlers
}

func (a *ModuleAggregator) GetComponentHandlers() map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	handlers := make(map[string]func(context.Context, *discordgo.Session, *discordgo.InteractionCreate))

	for _, m := range a.modules {
		for k, v := range m.GetComponentHandlers() {
			handlers[k] = v
		}
	}

	return handlers
}

func SetupApplicationCommands(s *discordgo.Session, module SlashModule, guildID string) (func(), error) {
	registeredCommands := make([]*discordgo.ApplicationCommand, 0)

	for _, cmd := range module.GetApplicationCommands() {
		ccmd, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
		if err != nil {
			return nil, fmt.Errorf("while adding %s application command: %v", cmd.Name, err)
		}

		registeredCommands = append(registeredCommands, ccmd)
	}

	if guildID != "" {
		return func() {
			for _, cmd := range registeredCommands {
				err := s.ApplicationCommandDelete(s.State.User.ID, guildID, cmd.ID)
				if err != nil {
					log.WithError(err).WithField("name", cmd.Name).Error("failed to delete command")
				}
			}
		}, nil
	}

	return func() {}, nil
}

func SetupDiscordHandlers(s *discordgo.Session, module SlashModule) {
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is running")
	})

	commandHandlers := module.GetCommandCreateHandlers()
	componentHandlers := module.GetComponentHandlers()

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				h(ctx, s, i)
			}
		case discordgo.InteractionMessageComponent:
			if h, ok := componentHandlers[i.MessageComponentData().CustomID]; ok {
				h(ctx, s, i)
			}
		}
	})
}

func serverErrorCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: serverNotAvailableMessage,
		},
	})

	if err != nil {
		log.WithError(err).Errorf("cannot respond")
	}
}

func clientErrorCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ %s", msg),
		},
	})

	if err != nil {
		log.WithError(err).Errorf("cannot respond")
	}
}

func unknownCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: unknownCommandMessage,
		},
	})

	if err != nil {
		log.WithError(err).Errorf("cannot respond")
	}
}

func stringResponseHandler(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})
	if err != nil {
		logrus.WithError(err).Errorf("cannot respond")
	}
}
