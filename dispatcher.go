// Author  Raido Pahtma
// License MIT

package sfconnection

import "fmt"

type Dispatcher interface {
	NewPacket() Packet
	Header() byte
	RegisterReceiver(chan Packet) error
	Receive([]byte)
}

type PacketDispatcher struct {
	header   byte
	factory  PacketFactory
	receiver chan Packet
}

type MessageDispatcher struct {
	header    byte
	receivers map[byte]chan Packet
	PacketDispatcher
}

func (self *MessageDispatcher) Receive(msg []byte) {
	p := self.factory.New()
	err := p.Deserialize(msg)
	if err == nil {
		if receiver, ok := self.receivers[self.header]; ok {
			receiver <- p
		}
		if self.receiver != nil {
			self.receiver <- p
		}
	} else {
		fmt.Printf("Unable to deserialize %s\n", err)
	}
}

func (self *PacketDispatcher) Receive(msg []byte) {
	p := self.factory.New()
	err := p.Deserialize(msg)
	if err == nil {
		if self.receiver != nil {
			self.receiver <- p
		}
	} else {
		fmt.Printf("Unable to deserialize %s\n", err)
	}
}

func (self *PacketDispatcher) Header() byte {
	return self.header
}

func (self *PacketDispatcher) NewPacket() Packet {
	return self.factory.New()
}

func (self *PacketDispatcher) RegisterReceiver(receiver chan Packet) error {
	self.receiver = receiver
	return nil
}

func (self *MessageDispatcher) RegisterMessageSnooper(receiver chan Packet) error {
	return self.RegisterReceiver(receiver)
}

func (self *MessageDispatcher) RegisterMessageReceiver(ptype byte, receive chan Packet) error {
	self.receivers[ptype] = receive
	return nil
}

func NewPacketDispatcher(header byte, packetfactory PacketFactory) *PacketDispatcher {
	d := new(PacketDispatcher)
	d.header = header
	d.factory = packetfactory
	return d
}

func NewMessageDispatcher(packetfactory PacketFactory) *MessageDispatcher {
	d := new(MessageDispatcher)
	d.header = 0
	d.factory = packetfactory
	d.receivers = make(map[byte]chan Packet)
	return d
}
