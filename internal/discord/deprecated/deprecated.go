package deprecated

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func UnregisterDeprecatedCommands(s *discordgo.Session, guildID string) error {
	appID := s.State.User.ID

	cmds, err := s.ApplicationCommands(appID, guildID)
	if err != nil {
		return fmt.Errorf("while getting application commands: %w", err)
	}

	for _, cmd := range cmds {
		if cmd.Name != "todo" && cmd.Name != "reminder" {
			continue
		}

		if err := s.ApplicationCommandDelete(appID, guildID, cmd.ID); err != nil {
			return fmt.Errorf("while deleting command %s: %w", cmd.Name, err)
		}
	}

	return nil
}
