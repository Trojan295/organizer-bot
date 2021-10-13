package todo

type Entry struct {
	Text string
	// TODO add updatedAt
}

type List struct {
	Entries []Entry
}
