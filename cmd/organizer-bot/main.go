package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/Trojan295/organizer-bot/internal"
	"github.com/Trojan295/organizer-bot/internal/discord/commands"
	"github.com/bwmarrin/discordgo"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DiscordToken string `required:"true"`
	GuildID      *string
}

const (
	envconfigPrefix = "app"
)

var (
	ds  *discordgo.Session
	cfg Config
)

func setupDiscordHandlers() {
	ds.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is running")
	})

	handlers := commands.GetCommandHandlers()

	ds.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := handlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func setupApplicationCommands() error {
	log.WithField("GuildID", cfg.GuildID).Printf("setup application commands")

	for _, cmd := range commands.GetApplicationCommands(&commands.GetApplicationCommandInput{
		ChannelID: cfg.GuildID,
	}) {
		if _, err := ds.ApplicationCommandCreate(ds.State.User.ID, *cfg.GuildID, cmd); err != nil {
			return fmt.Errorf("while adding %s application command: %v", cmd.Name, err)
		}
	}

	return nil
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

	setupDiscordHandlers()

	if err := ds.Open(); err != nil {
		log.Fatal(fmt.Sprintf("while opening connection to Discord: %v", err))
	}

	if err := setupApplicationCommands(); err != nil {
		log.Fatal(fmt.Sprintf("while setup application commands: %v", err))
	}

	defer ds.Close()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
