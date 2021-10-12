package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

const (
	unknownCommandMessage     = "Unknown command."
	serverNotAvailableMessage = "is not available right now... Try again in a moment."
)

type SlashModule interface {
	GetApplicationCommands() []*discordgo.ApplicationCommand
	GetCommandCreateHandlers() map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

type ModuleAggregator struct {
	modules []SlashModule
}

func NewModuleAggregator() *ModuleAggregator {
	return &ModuleAggregator{}
}

func (a *ModuleAggregator) AddModule(module SlashModule) {
	a.modules = append(a.modules, module)
}

func (a *ModuleAggregator) GetApplicationCommands() []*discordgo.ApplicationCommand {
	return a.modules[0].GetApplicationCommands()
}

func (a *ModuleAggregator) GetCommandCreateHandlers() map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return a.modules[0].GetCommandCreateHandlers()
}

func SetupApplicationCommands(s *discordgo.Session, module SlashModule, guildID string) (func(), error) {
	registeredCommands := make([]*discordgo.ApplicationCommand, 0)

	for _, cmd := range module.GetApplicationCommands() {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
		if err != nil {
			return nil, fmt.Errorf("while adding %s application command: %v", cmd.Name, err)
		}

		registeredCommands = append(registeredCommands, cmd)
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

	handlers := module.GetCommandCreateHandlers()

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := handlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
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
