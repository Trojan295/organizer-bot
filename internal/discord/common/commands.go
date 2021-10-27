package common

import (
	"fmt"

	"github.com/Trojan295/organizer-bot/internal/discord/root"
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

const (
	unknownCommandMessage     = "Unknown command."
	serverNotAvailableMessage = "is not available right now... Try again in a moment."
)

func SetupApplicationCommands(s *discordgo.Session, module *root.Module, guildID string) (func(), error) {
	cmd := module.GetApplicationCommand()

	registeredCommand, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
	if err != nil {
		return nil, fmt.Errorf("while adding %s application command: %v", cmd.Name, err)
	}

	if guildID != "" {
		return func() {
			err := s.ApplicationCommandDelete(s.State.User.ID, guildID, registeredCommand.ID)
			if err != nil {
				log.WithError(err).WithField("name", registeredCommand.Name).Error("failed to delete command")
			}
		}, nil
	}

	return func() {}, nil
}

func ServerErrorCommandHandler(log *log.Entry, s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: serverNotAvailableMessage,
		},
	})

	if err != nil {
		log.WithError(err).
			Error("cannot respond with server error")
	}
}

func ClientErrorCommandHandler(log *log.Entry, s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ %s", msg),
		},
	})

	if err != nil {
		log.WithError(err).
			Error("cannot repond with client error")
	}
}

func UnknownCommandHandler(log *log.Entry, s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: unknownCommandMessage,
		},
	})

	if err != nil {
		log.WithError(err).
			Error("cannot respond with unknown command")
	}
}

func StringResponseHandler(log *log.Entry, s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})
	if err != nil {
		log.WithError(err).
			Error("cannot respond with string response")
	}
}
