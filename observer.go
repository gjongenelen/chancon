package chancon

import (
	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
	"sync"
)

type ObserverCallback func(m *Message) error

type Observer struct {
	Id       uuid.UUID
	Channel  string
	Callback ObserverCallback
}

type observerManager struct {
	channels map[string][]*Observer
	lock     *sync.RWMutex
}

func newObserverManager() *observerManager {
	return &observerManager{
		channels: map[string][]*Observer{},
		lock:     &sync.RWMutex{},
	}
}

func (o *observerManager) On(channel string, callback ObserverCallback) func() {
	o.lock.Lock()
	defer o.lock.Unlock()

	if o.channels[channel] == nil {
		o.channels[channel] = []*Observer{}
	}

	observer := &Observer{
		Id:       uuid.New(),
		Callback: callback,
		Channel:  channel,
	}

	o.channels[channel] = append(o.channels[channel], observer)

	return func() {
		o.lock.Lock()
		defer o.lock.Unlock()

		newObservers := []*Observer{}
		for _, slicedObserver := range o.channels[channel] {
			if observer.Id != slicedObserver.Id {
				newObservers = append(newObservers, slicedObserver)
			}
		}
		o.channels[channel] = newObservers
	}
}

func (o *observerManager) Handle(message *Message) {
	o.lock.RLock()
	defer o.lock.RUnlock()

	observers, ok := o.channels[message.Channel.Name]
	if !ok {
		return
	}

	for _, observer := range observers {
		go func(observer *Observer) {
			err := observer.Callback(message)
			if err != nil {
				log.Errorf("Error in observer (channel %s): %s", message.Channel.Name, err.Error())
			}
		}(observer)
	}
}
