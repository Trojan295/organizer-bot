package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

type GetApplicationCommandInput struct {
	ChannelID *string
}

func GetApplicationCommands(input *GetApplicationCommandInput) []*discordgo.ApplicationCommand {
	return NewTodoApplicationCommands(&NewTodoApplicationCommandInput{})
}

func GetCommandHandlers() map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handler := make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))

	for k, v := range TodoCommandHandlers() {
		handler[k] = v
	}

	return handler
}

func unknownCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Unknown command",
		},
	})

	if err != nil {
		logrus.WithError(err).Errorf("cannot respond to todo show")
	}
}
