package todo

import "errors"

type Entry struct {
	Text string
	Done bool
}

type List struct {
	Entries []Entry
}

type InMemoryListRepository struct {
	Lists map[string]List
}

func (r *InMemoryListRepository) Get(ID string) (*List, error) {
	list, ok := r.Lists[ID]
	if !ok {
		return &List{}, nil
	}

	return &list, nil
}

func (r *InMemoryListRepository) Save(ID string, list *List) error {
	if list == nil {
		return errors.New("list must not be nil")
	}

	if r.Lists == nil {
		r.Lists = make(map[string]List)
	}

	r.Lists[ID] = *list
	return nil
}
