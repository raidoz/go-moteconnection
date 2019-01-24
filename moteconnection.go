// Author  Raido Pahtma
// License MIT

// SmartDustMote connection library.
package moteconnection

import "time"
import "sync"
import "sync/atomic"
import "errors"

import "github.com/proactivity-lab/go-loggers"

type Packet interface {
	Dispatch() byte
	Serialize() ([]byte, error)
	Deserialize([]byte) error
	SetPayload([]byte) error
	GetPayload() []byte
}

type PacketFactory interface {
	Dispatch() byte
	NewPacket() Packet
}

type Dispatcher interface {
	Dispatch() byte
	Receive([]byte) error
	NewPacket() Packet
}

type MoteConnection interface {
	loggers.DIWElog
	Connect() error
	Autoconnect(period time.Duration)
	Send(msg Packet) error
	AddDispatcher(dispatcher Dispatcher) error
	RemoveDispatcher(dispatch Dispatcher) error
	Connected() bool
	Disconnect()
}

type BaseMoteConnection struct {
	loggers.DIWEloggers

	connectlock   sync.Mutex
	shouldconnect atomic.Value
	autoconnect   bool
	period        time.Duration

	connected atomic.Value
	outgoing  chan []byte
	incoming  chan []byte

	dispatchers      map[byte]Dispatcher
	removeDispatcher chan Dispatcher
	addDispatcher    chan Dispatcher

	watchdog chan bool
	closed   chan bool
	close    chan bool
}

func (self *BaseMoteConnection) notifyWatchdog(outcome bool) {
	select {
	case self.watchdog <- outcome:
	case <-time.After(50 * time.Millisecond): // Because notification could happen before the watchdog has started
		self.Debug.Printf("No watchdog?\n")
	}
}

func (self *BaseMoteConnection) Send(msg Packet) error {
	serialized, err := msg.Serialize()
	if err != nil {
		return err
	}
	select {
	case self.outgoing <- serialized:
		return nil
	case <-time.After(50 * time.Millisecond): // Because the run goroutine might be doing something other than reading self.outgoing at this very moment
		return errors.New("Not connected!")
	}
}

func (self *BaseMoteConnection) AddDispatcher(dispatcher Dispatcher) error {
	self.connectlock.Lock()
	defer self.connectlock.Unlock()
	if self.connected.Load().(bool) {
		self.addDispatcher <- dispatcher
	} else {
		self.dispatchers[dispatcher.Dispatch()] = dispatcher
	}
	return nil
}

func (self *BaseMoteConnection) RemoveDispatcher(dispatcher Dispatcher) error {
	self.connectlock.Lock()
	defer self.connectlock.Unlock()
	if self.connected.Load().(bool) {
		self.removeDispatcher <- dispatcher
	} else {
		delete(self.dispatchers, dispatcher.Dispatch())
	}
	return nil
}

func (self *BaseMoteConnection) Connected() bool {
	return self.connected.Load().(bool)
}

func (self *BaseMoteConnection) Disconnect() {
	self.notifyWatchdog(false) // Because of autoconnect, watchdog may be active

	self.connectlock.Lock()
	defer self.connectlock.Unlock()

	self.shouldconnect.Store(false)
	self.autoconnect = false
	if self.connected.Load().(bool) {
		self.close <- true
	}
}
