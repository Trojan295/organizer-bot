package reminder

import "time"

type Reminders struct {
	Items []Item
}

type Item struct {
	Title string
	Date  *time.Time
}
