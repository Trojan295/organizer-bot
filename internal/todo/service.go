package todo

import (
	"context"
	"fmt"
	"time"

	"github.com/Trojan295/organizer-bot/internal/organizer"
	log "github.com/sirupsen/logrus"
)

type Pusher interface {
	PushTodoListNotification(ctx context.Context, ID string, list List) error
}

type Service struct {
	pusher        Pusher
	store         *RedisTodoStore
	timezoneStore *organizer.RedisConfigStore

	logger *log.Entry
}

type ServiceConfig struct {
	Pusher        Pusher
	Store         *RedisTodoStore
	TimezoneStore *organizer.RedisConfigStore
}

func NewService(cfg *ServiceConfig) (*Service, error) {
	if cfg == nil {
		cfg = &ServiceConfig{}
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

	return &Service{
		pusher:        cfg.Pusher,
		store:         cfg.Store,
		timezoneStore: cfg.TimezoneStore,
		logger:        log.NewEntry(log.New()).WithField("struct", "todo.Service"),
	}, nil
}

func (service *Service) Run(ctx context.Context) error {
	channelIDs, err := service.store.GetAllChannelsWithTodo(ctx)
	if err != nil {
		return fmt.Errorf("while gettings all channels: %w", err)
	}

	now := time.Now()

	for _, ID := range channelIDs {
		timezone, err := service.timezoneStore.GetCurrentTimezone(ctx, ID)
		if err != nil {
			service.logger.WithError(err).WithField("channelID", ID).Error("failed to get timezone")
			continue
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

			if err := service.pusher.PushTodoListNotification(ctx, ID, list); err != nil {
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
