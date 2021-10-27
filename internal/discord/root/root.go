package root

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Submodule interface {
	GetApplicationCommandSubgroups() []*discordgo.ApplicationCommandOption
	GetApplicationCommandInteractionHandlers() map[string]func(context.Context, *discordgo.Session, *discordgo.InteractionCreate, *discordgo.ApplicationCommandInteractionDataOption)
	GetMessageComponentInteractionHandlers() map[string]func(context.Context, *discordgo.Session, *discordgo.InteractionCreate)
}

type Module struct {
	name        string
	description string
	submodules  []Submodule
}

type ModuleConfig struct {
	Name        string
	Description string
}

func NewModule(cfg *ModuleConfig) (*Module, error) {
	if cfg == nil {
		return nil, fmt.Errorf("ModuleConfig is nil")
	}

	if cfg.Name == "" {
		return nil, fmt.Errorf("Module name is empty")
	}

	if cfg.Description == "" {
		cfg.Description = fmt.Sprintf("%s bot commands", cfg.Name)
	}

	return &Module{
		name:        cfg.Name,
		description: cfg.Description,
	}, nil
}

func (module *Module) AddSubmodules(submodules ...Submodule) {
	module.submodules = append(module.submodules, submodules...)
}

func (module *Module) GetApplicationCommand() *discordgo.ApplicationCommand {
	subcommands := make([]*discordgo.ApplicationCommandOption, 0)

	for _, submodule := range module.submodules {
		cmds := submodule.GetApplicationCommandSubgroups()
		subcommands = append(subcommands, cmds...)
	}

	return &discordgo.ApplicationCommand{
		Name:        module.name,
		Description: module.description,
		Options:     subcommands,
	}
}

func (module *Module) GetApplicationCommandInteractionHandlers() map[string]func(context.Context, *discordgo.Session, *discordgo.InteractionCreate, *discordgo.ApplicationCommandInteractionDataOption) {
	handlers := make(map[string]func(context.Context, *discordgo.Session, *discordgo.InteractionCreate, *discordgo.ApplicationCommandInteractionDataOption))

	for _, submodule := range module.submodules {
		modHandlers := submodule.GetApplicationCommandInteractionHandlers()

		for ID, handler := range modHandlers {
			handlers[ID] = handler
		}
	}

	return handlers
}

func (module *Module) GetMessageComponentInteractionHandlers() map[string]func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) {
	handlers := make(map[string]func(context.Context, *discordgo.Session, *discordgo.InteractionCreate))

	for _, submodule := range module.submodules {
		modHandlers := submodule.GetMessageComponentInteractionHandlers()

		for ID, handler := range modHandlers {
			handlers[ID] = handler
		}
	}

	return handlers
}
