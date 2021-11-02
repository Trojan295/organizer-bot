package todo

type Entry struct {
	ID   string
	Text string
	// TODO add updatedAt
}

type List struct {
	ChannelID string
	Entries   []*Entry
}
