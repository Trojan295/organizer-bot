package metrics

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/log"
)

type CommandResult string

const (
	commandLabel = "command"
	resultLabel  = "result"

	ResultSuccess     CommandResult = "success"
	ResultClientError CommandResult = "error_client"
	ResultServerError CommandResult = "error_server"
)

var (
	recordPeriod      = 60 * time.Second
	activeGuildsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "organizer_bot_active_guilds",
		Help: "The total number of guilds, in which the bot is active.",
	})

	executedCommandsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "organizer_bot_executed_commands_total",
	}, []string{commandLabel, resultLabel})
)

func RunDiscordMetricsRecorder(ctx context.Context, ds *discordgo.Session) {
	log.Info("Starting Discord metrics recorder")

	recordDiscordMetrics(ds)

	for {
		select {
		case <-time.After(recordPeriod):
			recordDiscordMetrics(ds)

		case <-ctx.Done():
			log.Info("Discord metrics recording stopped")
			return
		}
	}
}

func recordDiscordMetrics(ds *discordgo.Session) {
	activeGuilds := len(ds.State.Guilds)
	activeGuildsGauge.Set(float64(activeGuilds))
}

func CountExecutedCommand(command string) {
	executedCommandsCounter.With(prometheus.Labels{
		commandLabel: command,
		resultLabel:  string(ResultSuccess),
	}).Inc()
}

func CountServerErroredCommand(command string) {
	executedCommandsCounter.With(prometheus.Labels{
		commandLabel: command,
		resultLabel:  string(ResultServerError),
	}).Inc()
}

func CountClientErroredCommand(command string) {
	executedCommandsCounter.With(prometheus.Labels{
		commandLabel: command,
		resultLabel:  string(ResultClientError),
	}).Inc()
}
