package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
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

func serverErrorCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: serverNotAvailableMessage,
		},
	})

	if err != nil {
		logrus.WithError(err).Errorf("cannot respond")
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
		logrus.WithError(err).Errorf("cannot respond")
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
		logrus.WithError(err).Errorf("cannot respond")
	}
}
