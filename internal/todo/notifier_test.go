package todo_test

import (
	"context"
	"testing"
	"time"

	"github.com/Trojan295/organizer-bot/internal/todo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type Mock struct {
	mock.Mock
}

func (p *Mock) PushTodoListNotification(ctx context.Context, list *todo.List) error {
	args := p.Called(ctx, list)
	return args.Error(0)
}

func (p *Mock) GetAllChannelsWithTodo(ctx context.Context) ([]string, error) {
	args := p.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (p *Mock) GetEntries(ctx context.Context, channelID string) (*todo.List, error) {
	args := p.Called(ctx, channelID)
	return args.Get(0).(*todo.List), args.Error(1)
}

func (p *Mock) GetLastTodoNotificationTimestamp(ctx context.Context, channelID string) (int64, error) {
	args := p.Called(ctx, channelID)
	return args.Get(0).(int64), args.Error(1)
}

func (p *Mock) SetLastTodoNotificationTimestamp(ctx context.Context, channelID string, timestamp int64) error {
	args := p.Called(ctx, channelID, timestamp)
	return args.Error(0)
}

func (p *Mock) GetCurrentTimezone(ctx context.Context, channelID string) (*time.Location, error) {
	args := p.Called(ctx, channelID)
	return args.Get(0).(*time.Location), args.Error(1)
}

type MockClock struct {
	FixedTime time.Time
}

func (c *MockClock) Now() time.Time {
	return c.FixedTime
}

func DatePtr(t time.Time) *time.Time {
	return &t
}

func RequireLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}

	return loc
}

func TestNotifier_Run(t *testing.T) {
	tt := map[string]struct {
		currentTimestamp          time.Time
		lastNotificationTimestamp int64
		timeZone                  *time.Location
		notified                  bool
	}{
		"UTC_NoNotificationBeforeTime": {
			currentTimestamp:          time.Date(2021, 11, 2, 8, 37, 0, 0, time.UTC),
			lastNotificationTimestamp: DatePtr(time.Date(2021, 11, 1, 9, 37, 0, 0, time.UTC)).Unix(),
			timeZone:                  time.UTC,
			notified:                  false,
		},
		"UTC_SendNotificationAfterTime": {
			currentTimestamp:          time.Date(2021, 11, 2, 9, 37, 0, 0, time.UTC),
			lastNotificationTimestamp: DatePtr(time.Date(2021, 11, 1, 9, 37, 0, 0, time.UTC)).Unix(),
			timeZone:                  time.UTC,
			notified:                  true,
		},
		"UTC_FirstNotificationBefore": {
			currentTimestamp:          time.Date(2021, 11, 2, 8, 37, 0, 0, time.UTC),
			lastNotificationTimestamp: 0,
			timeZone:                  time.UTC,
			notified:                  false,
		},
		"UTC_FirstNotificationAfter": {
			currentTimestamp:          time.Date(2021, 11, 2, 9, 37, 0, 0, time.UTC),
			lastNotificationTimestamp: 0,
			timeZone:                  time.UTC,
			notified:                  true,
		},
		"UTC_NotificationAlreadySend": {
			currentTimestamp:          time.Date(2021, 11, 2, 10, 37, 0, 0, time.UTC),
			lastNotificationTimestamp: DatePtr(time.Date(2021, 11, 2, 9, 37, 0, 0, time.UTC)).Unix(),
			timeZone:                  time.UTC,
			notified:                  false,
		},
		"TZ_NoNotificationBeforeTime": {
			currentTimestamp:          time.Date(2021, 11, 2, 8, 37, 0, 0, RequireLocation("America/Chicago")),
			lastNotificationTimestamp: DatePtr(time.Date(2021, 11, 1, 9, 37, 0, 0, RequireLocation("America/Chicago"))).Unix(),
			timeZone:                  RequireLocation("America/Chicago"),
			notified:                  false,
		},
		"TZ_SendNotificationAfterTime": {
			currentTimestamp:          time.Date(2021, 11, 2, 9, 37, 0, 0, RequireLocation("America/Chicago")),
			lastNotificationTimestamp: DatePtr(time.Date(2021, 11, 1, 9, 37, 0, 0, RequireLocation("America/Chicago"))).Unix(),
			timeZone:                  RequireLocation("America/Chicago"),
			notified:                  true,
		},
		"MissingTimezone_AssumeUTC": {
			currentTimestamp:          time.Date(2021, 11, 2, 8, 37, 0, 0, time.UTC),
			lastNotificationTimestamp: DatePtr(time.Date(2021, 11, 1, 9, 37, 0, 0, time.UTC)).Unix(),
			timeZone:                  nil,
			notified:                  false,
		},
	}

	for name, test := range tt {
		list := &todo.List{
			ChannelID: "channelID",
			Entries: []*todo.Entry{
				{
					ID:   "1",
					Text: "Hello!",
				},
			},
		}
		ctx := context.Background()

		mock := &Mock{}

		mock.On("GetCurrentTimezone", ctx, list.ChannelID).Return(test.timeZone, nil)
		mock.On("GetAllChannelsWithTodo", ctx).Return([]string{list.ChannelID}, nil)
		mock.On("GetLastTodoNotificationTimestamp", ctx, list.ChannelID).Return(test.lastNotificationTimestamp, nil)

		if test.notified {
			mock.On("GetEntries", ctx, list.ChannelID).Return(list, nil)
			mock.On("SetLastTodoNotificationTimestamp", ctx, list.ChannelID, test.currentTimestamp.Unix()).Return(nil)
			mock.On("PushTodoListNotification", ctx, list).Return(nil)
		}

		t.Run(name, func(t *testing.T) {
			notifier, err := todo.NewNotifier(&todo.NotifierConfig{
				Pusher:        mock,
				Store:         mock,
				TimezoneStore: mock,
				Clock:         &MockClock{FixedTime: test.currentTimestamp},
			})
			require.NoError(t, err)

			err = notifier.Run(context.Background())
			require.NoError(t, err)

			mock.AssertExpectations(t)
		})
	}
}
