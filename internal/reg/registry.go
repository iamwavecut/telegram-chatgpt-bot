package reg

import (
	"sync"
)

type (
	registry map[string]any
)

var instance registry //nolint:gochecknoglobals // desired behavior
var once sync.Once    //nolint:gochecknoglobals // desired behavior
var mx sync.Mutex     //nolint:gochecknoglobals // desired behavior

func Get[T any](key string, defaults T) T {
	once.Do(func() { instance = registry{} })
	if _, ok := instance[key]; !ok {
		instance[key] = defaults
	}
	return instance[key].(T)
}

func Set(key string, value any) {
	mx.Lock()
	defer mx.Unlock()
	once.Do(func() { instance = registry{} })
	instance[key] = value
}

func Delete(key string) {
	mx.Lock()
	defer mx.Unlock()
	once.Do(func() { instance = registry{} })
	delete(instance, key)
}
