package reg

import (
	"sync"
)

type (
	registry map[string]any
)

var instance registry
var once sync.Once

func Get[T any](key string, defaults T) T {
	once.Do(func() { instance = registry{} })
	if _, ok := instance[key]; !ok {
		instance[key] = defaults
	}
	return instance[key].(T)
}

func Set(key string, value any) {
	once.Do(func() { instance = registry{} })
	instance[key] = value
}

func Delete(key string) {
	once.Do(func() { instance = registry{} })
	delete(instance, key)
}
