package datacompletion

import (
	"sync"
)

type DataSource interface {
	GetItems() map[string]bool
	setFilter()
}

type source struct {
	items sync.Map
}

func (s *source) GetItems() map[string]bool {
	m := map[string]bool{}
	s.items.Range(func(key, value interface{}) bool {
		m[key.(string)] = true
		return true
	})
	return m
}
