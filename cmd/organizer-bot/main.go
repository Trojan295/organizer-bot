package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/Trojan295/organizer-bot/internal"
	"github.com/Trojan295/organizer-bot/internal/discord/commands"
	"github.com/Trojan295/organizer-bot/internal/todo"
	"github.com/bwmarrin/discordgo"
	"github.com/kelseyhightower/envconfig"
)

type TodoConfig struct {
	DynamoDBTableName string `required:"true"`
}

type Config struct {
	DiscordToken string `required:"true"`
	GuildID      string

	TodoConfig TodoConfig `required:"true"`
}

const (
	envconfigPrefix = "app"
)

var (
	ds  *discordgo.Session
	cfg Config
)

func setupDiscordHandlers(module commands.SlashModule) {
	ds.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is running")
	})

	handlers := module.GetCommandCreateHandlers()

	ds.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := handlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func setupApplicationCommands(module commands.SlashModule) ([]*discordgo.ApplicationCommand, error) {
	log.WithField("GuildID", cfg.GuildID).Printf("setup application commands")

	registeredCommands := make([]*discordgo.ApplicationCommand, 0)

	for _, cmd := range module.GetApplicationCommands() {
		cmd, err := ds.ApplicationCommandCreate(ds.State.User.ID, cfg.GuildID, cmd)
		if err != nil {
			return nil, fmt.Errorf("while adding %s application command: %v", cmd.Name, err)
		}

		registeredCommands = append(registeredCommands, cmd)
	}

	return registeredCommands, nil
}

func getSlashModule() (commands.SlashModule, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, errors.Wrap(err, "while creating AWS session")
	}

	todoRepo := todo.NewDynamoDBRepostory(sess, cfg.TodoConfig.DynamoDBTableName)

	todoModule := commands.NewTodoModule(todoRepo)
	return todoModule, err
}

func main() {
	var err error

	if err := internal.SetupLogging(); err != nil {
		log.Fatal(fmt.Sprintf("while setup logging: %v", err))
	}

	if err := envconfig.Process(envconfigPrefix, &cfg); err != nil {
		log.Fatal(fmt.Sprintf("while loading envconfig: %v", err))
	}

	ds, err = discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatal(fmt.Sprintf("while creating Discord client: %v", err))
	}

	slashMod, err := getSlashModule()
	if err != nil {
		log.Fatal(fmt.Sprintf("while getting SlashModule: %v", err))
	}

	setupDiscordHandlers(slashMod)

	if err := ds.Open(); err != nil {
		log.Fatal(fmt.Sprintf("while opening connection to Discord: %v", err))
	}

	registeredAppCmds, err := setupApplicationCommands(slashMod)
	if err != nil {
		log.Fatal(fmt.Sprintf("while setup application commands: %v", err))
	}

	defer func() {
		for _, cmd := range registeredAppCmds {
			if err := ds.ApplicationCommandDelete(cmd.ApplicationID, cfg.GuildID, cmd.ID); err != nil {
				logrus.WithError(err).Error("failed to delete ApplicationCommand")
			}
		}
	}()

	defer ds.Close()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
