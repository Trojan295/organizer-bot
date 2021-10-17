package reminder

import "time"

type Reminder struct {
	Title string
	Date  *time.Time
}
