package reminder

import "time"

type Reminder struct {
	ID        string
	ChannelID string
	Title     string
	Date      *time.Time
}
