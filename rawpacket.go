// Author  Raido Pahtma
// License MIT

package sfconnection

import "fmt"

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
