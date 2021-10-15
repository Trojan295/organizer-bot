package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"
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

type TestingConfig struct {
	GuildID string
}

type Config struct {
	DiscordToken string `required:"true"`

	Testing TestingConfig
	Todo    TodoConfig `required:"true"`
}

const (
	envconfigPrefix = "app"
)

var (
	ds  *discordgo.Session
	cfg Config
)

func getSlashModule() (commands.SlashModule, error) {
	module := commands.NewModuleAggregator()

	sess, err := session.NewSession()
	if err != nil {
		return nil, errors.Wrap(err, "while creating AWS session")
	}

	todoRepo := todo.NewDynamoDBRepostory(sess, cfg.Todo.DynamoDBTableName)

	todoModule := commands.NewTodoModule(todoRepo)
	module.AddModule(todoModule)

	return module, nil
}

func main() {
	var err error

	if err := internal.SetupLogging(); err != nil {
		log.Fatal(fmt.Sprintf("while setup logging: %v", err))
	}

	if err := envconfig.Process(envconfigPrefix, &cfg); err != nil {
		log.WithError(err).Fatal("failed to load envconfig")
	}

	ds, err = discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.WithError(err).Fatal("failed to create Discord client")
	}

	slashMod, err := getSlashModule()
	if err != nil {
		log.WithError(err).Fatal("failed to get SlashModule")
	}

	commands.SetupDiscordHandlers(ds, slashMod)

	if err := ds.Open(); err != nil {
		log.WithError(err).Fatal("failed to open connection to Discord")
	}

	cleanupAppCmds, err := commands.SetupApplicationCommands(ds, slashMod, cfg.Testing.GuildID)
	if err != nil {
		log.WithError(err).Fatal("failed to setup ApplicationCommands")
	}

	defer cleanupAppCmds()
	defer ds.Close()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
