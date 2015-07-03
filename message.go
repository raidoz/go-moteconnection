// Author  Raido Pahtma
// License MIT

package sfconnection

import "encoding/binary"
import "bytes"
import "errors"
import "fmt"

type Message struct {
	destination AMAddr
	source      AMAddr
	sourceSet   bool
	group       AMGroup
	groupSet    bool
	ptype       AMID
	Payload     []byte

	defaultSource AMAddr
	defaultGroup  AMGroup
}

var _ Packet = (*Message)(nil)
var _ PacketFactory = (*Message)(nil)

func NewMessageFactory(defaultGroup AMGroup, defaultSource AMAddr) *Message {
	msg := new(Message)
	msg.defaultGroup = defaultGroup
	msg.defaultSource = defaultSource
	return msg
}

// Message also serves as a factory.
func (self *Message) New() Packet {
	msg := new(Message)
	msg.defaultGroup = self.defaultGroup
	msg.defaultSource = self.defaultSource
	return msg
}

func (self *Message) Type() AMID {
	return self.ptype
}

func (self *Message) SetType(ptype AMID) {
	self.ptype = ptype
}

func (self *Message) Group() AMGroup {
	if self.groupSet {
		return self.group
	}
	return self.defaultGroup
}

func (self *Message) SetGroup(group AMGroup) {
	self.groupSet = true
	self.group = group
}

func (self *Message) Destination() AMAddr {
	return self.destination
}

func (self *Message) SetDestination(destination AMAddr) {
	self.destination = destination
}

func (self *Message) Source() AMAddr {
	if self.sourceSet {
		return self.source
	}
	return self.defaultSource
}

func (self *Message) SetSource(source AMAddr) {
	self.sourceSet = true
	self.source = source
}

func (self *Message) String() string {
	return fmt.Sprintf("%04X->%04X[%02X]% 3d: %X", self.source, self.destination, self.ptype, len(self.Payload), self.Payload)
}

func (self *Message) Serialize() ([]byte, error) {
	var err error
	buf := new(bytes.Buffer)

	if len(self.Payload) > 255-8 {
		return nil, errors.New(fmt.Sprintf("Message payload too long(%d)", len(self.Payload)))
	}

	err = binary.Write(buf, binary.BigEndian, uint8(0))
	if err != nil {
		panic(err)
	}

	err = binary.Write(buf, binary.BigEndian, self.Destination())
	if err != nil {
		panic(err)
	}

	err = binary.Write(buf, binary.BigEndian, self.Source())
	if err != nil {
		panic(err)
	}

	err = binary.Write(buf, binary.BigEndian, uint8(len(self.Payload)))
	if err != nil {
		panic(err)
	}

	err = binary.Write(buf, binary.BigEndian, self.Group())
	if err != nil {
		panic(err)
	}

	err = binary.Write(buf, binary.BigEndian, self.Type())
	if err != nil {
		panic(err)
	}

	_, err = buf.Write(self.Payload)
	if err != nil {
		panic(err)
	}

	return buf.Bytes(), nil
}

func (self *Message) Deserialize(data []byte) error {
	var err error

	var dispatch uint8
	var destination AMAddr
	var source AMAddr
	var length uint8
	var group AMGroup
	var ptype AMID

	buf := bytes.NewReader(data)

	err = binary.Read(buf, binary.BigEndian, &dispatch)
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &destination)
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &source)
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &length)
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &group)
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &ptype)
	if err != nil {
		return err
	}

	buflen := buf.Len()
	if uint8(buflen) != length {
		return errors.New(fmt.Sprintf("Payload length mismatch, header=%d, actual=%d", length, buflen))
	}

	payload := make([]byte, buflen)
	_, err = buf.Read(payload)
	if err != nil {
		return err
	}

	self.SetDestination(destination)
	self.SetSource(source)
	self.SetGroup(group)
	self.SetType(ptype)
	self.Payload = payload

	return nil
}
