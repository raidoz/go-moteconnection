// Author  Raido Pahtma
// License MIT

package sfconnection

import "fmt"

type Dispatcher interface {
	New() Packet
	Header() byte
	RegisterReceiver(byte, chan Packet) error
	RegisterSnooper(chan Packet) error
	Receive([]byte)
}

type PacketDispatcher struct {
	header    byte
	factory   PacketFactory
	receivers map[byte]chan Packet
	snooper   chan Packet
}

func (self *PacketDispatcher) Receive(msg []byte) {
	p := self.factory.New()
	err := p.Deserialize(msg)
	if err == nil {
		if receiver, ok := self.receivers[p.Type()]; ok {
			receiver <- p
		}
		if self.snooper != nil {
			self.snooper <- p
		}
	} else {
		fmt.Printf("Unable to deserialize %s\n", err)
	}
}

func (self *PacketDispatcher) Header() byte {
	return self.header
}

func (self *PacketDispatcher) New() Packet {
	return self.factory.New()
}

func (self *PacketDispatcher) RegisterReceiver(ptype byte, receive chan Packet) error {
	self.receivers[ptype] = receive
	return nil
}

func (self *PacketDispatcher) RegisterSnooper(receive chan Packet) error {
	self.snooper = receive
	return nil
}

func NewPacketDispatcher(header byte, packetfactory PacketFactory) *PacketDispatcher {
	d := new(PacketDispatcher)
	d.header = header
	d.factory = packetfactory
	d.receivers = make(map[byte]chan Packet)
	return d
}

type RawPacket struct {
	Payload []byte
}

func (self *RawPacket) New() Packet {
	return new(RawPacket)
}

func (self *RawPacket) Type() byte {
	return 0
}

func (self *RawPacket) Serialize() ([]byte, error) {
	p := make([]byte, len(self.Payload))
	copy(p, self.Payload)
	return p, nil
}

func (self *RawPacket) Deserialize(data []byte) error {
	self.Payload = make([]byte, len(data))
	copy(self.Payload, data)
	return nil
}

func (self *RawPacket) String() string {
	return fmt.Sprintf("%X", self.Payload)
}
