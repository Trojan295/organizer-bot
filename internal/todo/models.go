package todo

type Entry struct {
	Text string
	// TODO add addedAt
}

type List struct {
	Entries []Entry
}
