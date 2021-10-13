package schedule

import "time"

type Schedule struct {
	Items []Item
}

type Item struct {
	Title string
	Date  *time.Time
}
