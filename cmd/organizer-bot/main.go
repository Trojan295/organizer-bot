package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"

	"github.com/Trojan295/organizer-bot/internal"
	"github.com/Trojan295/organizer-bot/internal/discord/commands"
	"github.com/Trojan295/organizer-bot/internal/reminder"
	"github.com/Trojan295/organizer-bot/internal/todo"
	"github.com/bwmarrin/discordgo"
	"github.com/kelseyhightower/envconfig"
)

type TodoConfig struct {
	RedisAddress  string `required:"true"`
	RedisDB       int    `default:"0"`
	RedisPassword string
}

type ReminderConfig struct {
	DynamoDBTableName string `required:"true"`
}

type TestingConfig struct {
	GuildID string
}

type Config struct {
	DiscordToken string `required:"true"`

	Testing  TestingConfig
	Todo     TodoConfig     `required:"true"`
	Reminder ReminderConfig `required:"true"`
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

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Todo.RedisAddress,
		Password: cfg.Todo.RedisPassword,
		DB:       cfg.Todo.RedisDB,
	})

	todoRepo := todo.NewRedisTodoStore(rdb)
	todoModule := commands.NewTodoModule(todoRepo)

	reminderRepo := reminder.NewRedisReminderStore(rdb)
	reminderModule := commands.NewReminderModule(reminderRepo)

	module.AddModules(todoModule, reminderModule)

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
