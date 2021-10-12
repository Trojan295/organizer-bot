package todo

import "errors"

type InMemoryListRepository struct {
	Lists map[string]List
}

func (r *InMemoryListRepository) Get(id string) (*List, error) {
	list, ok := r.Lists[id]
	if !ok {
		return &List{}, nil
	}

	return &list, nil
}

func (r *InMemoryListRepository) Save(id string, list *List) error {
	if list == nil {
		return errors.New("list must not be nil")
	}

	if r.Lists == nil {
		r.Lists = make(map[string]List)
	}

	r.Lists[id] = *list
	return nil
}
