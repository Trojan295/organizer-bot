package reminder

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type Pusher interface {
	PushReminder(ctx context.Context, reminder *Reminder) error
}

type Service struct {
	pusher Pusher
	store  *RedisReminderStore
}

func NewService(pusher Pusher, store *RedisReminderStore) *Service {
	return &Service{
		pusher: pusher,
		store:  store,
	}
}

func (svc *Service) Run(ctx context.Context) error {
	reminders, err := svc.store.GetTriggeredReminders(ctx)
	if err != nil {
		return errors.Wrap(err, "while getting triggered reminders")
	}

	var pushErr error

	for _, rem := range reminders {
		if err := svc.pusher.PushReminder(ctx, rem); err != nil {
			pushErr = multierror.Append(pushErr, err)
			continue
		}

		if err := svc.store.RemoveReminder(ctx, rem.ChannelID, rem.ID); err != nil {
			pushErr = multierror.Append(pushErr, err)
			continue
		}
	}

	if pushErr != nil {
		return errors.Wrap(err, "while sending reminders")
	}

	return nil
}
