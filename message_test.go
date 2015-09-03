// Author  Raido Pahtma
// License MIT

package sfconnection

import "testing"

import "fmt"
import "encoding/hex"
import "bytes"

func TestMessageSerialization(t *testing.T) {
	payload := "00112233445566778899AABBCCDDEEFF"

	msg := new(Message)
	msg.SetDestination(0xFFFF)
	msg.SetSource(1)
	msg.SetType(3)
	msg.SetGroup(0xFF)
	msg.Payload, _ = hex.DecodeString(payload)
	ser, _ := msg.Serialize()

	fmt.Println(ser)

	expected, _ := hex.DecodeString("00FFFF0001" + fmt.Sprintf("%02X", len(payload)/2) + "FF03" + payload)

	if bytes.Compare(ser, expected) != 0 {
		t.Error(fmt.Sprintf("%X != %X", ser, expected))
	}
}

func TestMessageDeserialization(t *testing.T) {
	raw, _ := hex.DecodeString("00FFFF000110FF0300112233445566778899AABBCCDDEEFF")

	msg := new(Message)
	err := msg.Deserialize(raw)
	if err != nil {
		t.Error(err)
	}

	if msg.Destination() != 0xFFFF || msg.Source() != 1 || msg.Type() != 3 || msg.Group() != 0xFF {
		t.Error("Header mismatch")
	}

	if bytes.Compare(msg.Payload, raw[8:]) != 0 {
		t.Error(fmt.Sprintf("Payload mismatch %X != %X", msg.Payload, raw[8:]))
	}
}

func TestMessagePrinting(t *testing.T) {
	raw, _ := hex.DecodeString("00FFFF000110FF0300112233445566778899AABBCCDDEEFF")

	msg := new(Message)
	err := msg.Deserialize(raw)
	if err != nil {
		t.Error(err)
	}

	expected := "{FF}0001->FFFF[03] 16: 00112233445566778899AABBCCDDEEFF"
	stringed := fmt.Sprintf("%s", msg)

	if stringed != expected {
		t.Error(fmt.Sprintf("Packet did not print correctly:\nExpected: %s\nReceived: %s", expected, stringed))
	}
}
