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

type RedisConfig struct {
	Address  string `required:"true"`
	Password string
	DB       int
}

type TestingConfig struct {
	GuildID string
}

type Config struct {
	DiscordToken string `required:"true"`

	Redis   RedisConfig
	Testing TestingConfig
}

const (
	envconfigPrefix = "app"
)

var (
	ds  *discordgo.Session
	cfg Config
)

func getSlashModule() commands.SlashModule {
	module := commands.NewModuleAggregator()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	todoRepo := todo.NewRedisTodoStore(rdb)
	todoModule := commands.NewTodoModule(todoRepo)

	reminderRepo := reminder.NewRedisReminderStore(rdb)
	reminderModule := commands.NewReminderModule(reminderRepo)

	module.AddModules(todoModule, reminderModule)

	return module
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

	slashMod := getSlashModule()

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
