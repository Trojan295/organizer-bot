package todo

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type Pusher interface {
	PushTodoListNotification(ctx context.Context, list *List) error
}

type Store interface {
	GetAllChannelsWithTodo(ctx context.Context) ([]string, error)
	GetEntries(ctx context.Context, channelID string) (*List, error)
	GetLastTodoNotificationTimestamp(ctx context.Context, channelID string) (int64, error)
	SetLastTodoNotificationTimestamp(ctx context.Context, channelID string, timestamp int64) error
}

type TimezoneStore interface {
	GetCurrentTimezone(ctx context.Context, channelID string) (*time.Location, error)
}

type Clock interface {
	Now() time.Time
}

type Notifier struct {
	pusher        Pusher
	store         Store
	timezoneStore TimezoneStore
	clock         Clock

	logger *log.Entry
}

type NotifierConfig struct {
	Pusher        Pusher
	Store         Store
	TimezoneStore TimezoneStore
	Clock         Clock
}

type RealClock struct{}

func (*RealClock) Now() time.Time {
	return time.Now()
}

func NewNotifier(cfg *NotifierConfig) (*Notifier, error) {
	if cfg == nil {
		cfg = &NotifierConfig{}
	}

	if cfg.Pusher == nil {
		return nil, fmt.Errorf("Pusher is not set")
	}

	if cfg.Store == nil {
		return nil, fmt.Errorf("Store is not set")
	}

	if cfg.TimezoneStore == nil {
		return nil, fmt.Errorf("TimezoneStore is not set")
	}

	if cfg.Clock == nil {
		cfg.Clock = &RealClock{}
	}

	return &Notifier{
		pusher:        cfg.Pusher,
		store:         cfg.Store,
		timezoneStore: cfg.TimezoneStore,
		clock:         cfg.Clock,
		logger:        log.NewEntry(log.New()).WithField("struct", "todo.Service"),
	}, nil
}

// TODO: write some tests
func (service *Notifier) Run(ctx context.Context) error {
	channelIDs, err := service.store.GetAllChannelsWithTodo(ctx)
	if err != nil {
		return fmt.Errorf("while gettings all channels: %w", err)
	}

	now := service.clock.Now()

	for _, ID := range channelIDs {
		timezone, err := service.timezoneStore.GetCurrentTimezone(ctx, ID)
		if err != nil {
			service.logger.WithError(err).WithField("channelID", ID).Error("failed to get timezone")
			continue
		}

		if timezone == nil {
			timezone = time.UTC
		}

		lastNotification, err := service.store.GetLastTodoNotificationTimestamp(ctx, ID)
		if err != nil {
			service.logger.WithError(err).WithField("channelID", ID).Error("failed to get last notification timestamp")
			continue
		}

		currentTime := now.In(timezone)
		lastNotificationTime := time.Unix(lastNotification, 0).In(timezone)

		// send notification if after 9.00 and not send this day
		if (lastNotification == 0 || currentTime.Day() != lastNotificationTime.Day()) && currentTime.Hour() >= 9 {
			list, err := service.store.GetEntries(ctx, ID)
			if err != nil {
				service.logger.WithError(err).WithField("channelID", ID).Error("failed to get todo list")
				continue
			}

			if err := service.pusher.PushTodoListNotification(ctx, list); err != nil {
				service.logger.WithError(err).WithField("channelID", ID).Error("failed to push todo list")
				continue
			}

			if err := service.store.SetLastTodoNotificationTimestamp(ctx, ID, now.Unix()); err != nil {
				service.logger.WithError(err).WithField("channelID", ID).Error("failed to set last notification timestamp")
				continue
			}
		}
	}

	return nil
}
