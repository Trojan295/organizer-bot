package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/Trojan295/organizer-bot/internal"
	"github.com/Trojan295/organizer-bot/internal/discord/common"
	"github.com/Trojan295/organizer-bot/internal/discord/deprecated"
	"github.com/Trojan295/organizer-bot/internal/discord/message"
	discordreminder "github.com/Trojan295/organizer-bot/internal/discord/reminder"
	"github.com/Trojan295/organizer-bot/internal/discord/root"
	discordtodo "github.com/Trojan295/organizer-bot/internal/discord/todo"
	"github.com/Trojan295/organizer-bot/internal/metrics"
	"github.com/Trojan295/organizer-bot/internal/organizer"
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

	reminderCheckInterval = 30 * time.Second
	requestTimeout        = 10 * time.Second

	rdb           *redis.Client
	reminderStore *reminder.RedisReminderStore
	configStore   *organizer.RedisConfigStore
)

func setupRedisClient() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}

func getRootModule() (*root.Module, error) {
	rootModule, err := root.NewModule(&root.ModuleConfig{
		Name: "organizer",
	})
	if err != nil {
		return nil, errors.Wrap(err, "while creating root module")
	}

	configStore = organizer.NewRedisConfigStore(rdb)
	reminderStore = reminder.NewRedisReminderStore(rdb)
	todoStore := todo.NewRedisTodoStore(rdb)

	todoModule, err := discordtodo.NewTodoModule(&discordtodo.ModuleConfig{
		TodoRepo: todoStore,
	})
	if err != nil {
		return nil, errors.Wrap(err, "while creating TodoModule")
	}

	reminderModule, err := discordreminder.NewReminderModule(&discordreminder.ModuleConfig{
		ReminderRepo:       reminderStore,
		TimezoneRepository: configStore,
	})
	if err != nil {
		return nil, errors.Wrap(err, "while creating ReminderModule")
	}

	configModule, err := common.NewConfigModule(&common.ConfigModuleInput{
		TimezoneRepository: configStore,
	})
	if err != nil {
		return nil, errors.Wrap(err, "while creating ConfigModule")
	}

	rootModule.AddSubmodules(configModule, reminderModule, todoModule)

	return rootModule, nil
}

func getReminderService() *reminder.Service {
	sender := message.NewSender(ds)
	return reminder.NewService(sender, reminderStore)
}

func setupCommandHandlers(s *discordgo.Session, rootModule *root.Module) {
	s.AddHandler(func(_ *discordgo.Session, _ *discordgo.Connect) {
		log.Info("connected to Discord")
	})

	s.AddHandler(func(_ *discordgo.Session, _ *discordgo.Disconnect) {
		log.Info("disconnected from Discord")
	})

	applicationCommandInteractionHandlers := rootModule.GetApplicationCommandInteractionHandlers()
	messageComponentInteractionHandlers := rootModule.GetMessageComponentInteractionHandlers()

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()

		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			cmd := i.ApplicationCommandData().Options[0].Name

			if f, ok := applicationCommandInteractionHandlers[cmd]; ok {
				f(ctx, s, i, i.ApplicationCommandData().Options[0])
			} else {
				log.WithField("command name", cmd).Warn("failed to find application command interaction")
			}

		case discordgo.InteractionMessageComponent:
			cmd := i.MessageComponentData().CustomID
			if f, ok := messageComponentInteractionHandlers[cmd]; ok {
				f(ctx, s, i)
			} else {
				log.WithField("customID", cmd).Warn("failed to find message component interaction")
			}
		}
	})
}

func startMetricsEndpoint(ctx context.Context) {
	go metrics.RunDiscordMetricsRecorder(ctx, ds)

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Error(err, "failed to serve HTTP server")
		}
	}()
}

func main() {
	var (
		err error
		ctx = context.Background()
	)

	if err := internal.SetupLogging(); err != nil {
		log.Fatal(fmt.Sprintf("while setup logging: %v", err))
	}

	if err := envconfig.Process(envconfigPrefix, &cfg); err != nil {
		log.WithError(err).Fatal("failed to load envconfig")
	}

	setupRedisClient()

	ds, err = discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.WithError(err).Fatal("failed to create Discord client")
	}

	rootModule, err := getRootModule()
	if err != nil {
		log.WithError(err).Fatal("failed to get SlashModule")
	}

	setupCommandHandlers(ds, rootModule)

	if err := ds.Open(); err != nil {
		log.WithError(err).Fatal("failed to open connection to Discord")
	}

	if err := deprecated.UnregisterDeprecatedCommands(ds, ""); err != nil {
		log.WithError(err).Error("failed to unregister deprecated commands")
	}

	cleanupAppCmds, err := common.SetupApplicationCommands(ds, rootModule, cfg.Testing.GuildID)
	if err != nil {
		log.WithError(err).Fatal("failed to setup ApplicationCommands")
	}

	defer cleanupAppCmds()
	defer ds.Close()

	startMetricsEndpoint(ctx)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	reminderSvc := getReminderService()

	for {
		select {
		case <-sc:
			log.Infof("shutting down application")
			return

		case <-time.Tick(reminderCheckInterval):
			if err := reminderSvc.Run(ctx); err != nil {
				log.WithError(err).Error("failed to run ReminderService")
			}
			break
		}
	}
}
