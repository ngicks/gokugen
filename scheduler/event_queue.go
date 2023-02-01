package scheduler

import (
	"github.com/ngicks/eventqueue"
)

type updateEventQueue struct {
	sink *eventqueue.ChannelSink[updateEvent]
	*eventqueue.EventQueue[updateEvent]
}

func newUpdateEventQueue() *updateEventQueue {
	sink := eventqueue.NewChannelSink[updateEvent](0)
	return &updateEventQueue{
		sink:       sink,
		EventQueue: eventqueue.New[updateEvent](sink),
	}
}

func (q *updateEventQueue) Subscribe() <-chan updateEvent {
	return q.sink.Outlet()
}
