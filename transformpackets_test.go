// Author  Raido Pahtma
// License MIT

package sfconnection

import "testing"
import "fmt"
import "reflect"
import "bytes"
import "encoding/hex"

type TransformTestPacket struct { // Field names must all be exported!
	TestUint8        uint8
	TestUint16       uint16
	TestUint32       uint32
	TestUint64       uint64
	TestInt8         int8
	TestInt16        int16
	TestInt32        int32
	TestInt64        int64
	TestBool         bool
	TestFixedArray   [4]byte // uint8 also works
	TestBytesLength  uint8   `sfpacket:"len(TestBytes)"`  // Length of the slice TestBytes is stored here. Length types must be unsigned!
	TestStringLength uint8   `sfpacket:"len(TestString)"` // Length of the string TestBytes is stored here. Length types must be unsigned!
	TestString       string
	TestBytes        []byte // uint8 also works
	TestTail         []byte // Everything extra will end up here on deserialization. uint8 also works
}

func createpacket() *TransformTestPacket {
	var p *TransformTestPacket = new(TransformTestPacket)
	p.TestUint8 = 0x00
	p.TestUint16 = 0xA0A1
	p.TestUint32 = 0xA0A1A2A3
	p.TestUint64 = 0xA0A1A2A3A4A5A6A7
	p.TestInt8 = 0x7F
	p.TestInt16 = 0x70F1
	p.TestInt32 = 0x70F1F2F3
	p.TestInt64 = 0x70F1F2F3F4F5F6F7
	p.TestBool = true
	p.TestFixedArray[0] = 0
	p.TestFixedArray[1] = 1
	p.TestFixedArray[2] = 2
	p.TestFixedArray[3] = 3
	p.TestString = "teststring"   // Length is automatically set by serialize
	p.TestBytes = []byte{4, 5, 6} // Length is automatically set by serialize
	p.TestTail = []byte{7, 8, 9}  // Last element of the packet, so length can be derived from packet length
	return p
}

func TestSerializer(t *testing.T) {
	var p *TransformTestPacket = createpacket()
	fmt.Printf("np %v\n", p)
	b := SerializePacket(p)
	fmt.Printf("sp %v\n", p)
	fmt.Printf("rb %X\n", b)
	r, _ := hex.DecodeString("00A0A1A0A1A2A3A0A1A2A3A4A5A6A77F70F170F1F2F370F1F2F3F4F5F6F70100010203030A74657374737472696E67040506070809")
	if !bytes.Equal(r, b) {
		t.Error("mismatch")
	}
}

func TestDeserializer(t *testing.T) {
	var dp TransformTestPacket
	var rp *TransformTestPacket = createpacket()

	raw, _ := hex.DecodeString("00A0A1A0A1A2A3A0A1A2A3A4A5A6A77F70F170F1F2F370F1F2F3F4F5F6F70100010203030A74657374737472696E67040506070809")
	if err := DeserializePacket(&dp, raw); err != nil {
		t.Error("error %s", err)
	}

	// Because createpacket leaves these uninitialized and DeepEqual would thus fail
	rp.TestBytesLength = uint8(len(rp.TestBytes))
	rp.TestStringLength = uint8(len(rp.TestString))

	fmt.Printf("rp %v\n", rp)
	fmt.Printf("dp %v\n", &dp)

	if !reflect.DeepEqual(dp, *rp) {
		t.Error("mismatch")
	}
}

func TestTooShort(t *testing.T) {
	var dp TransformTestPacket
	// The following is simply way too short to be deserialized
	raw, _ := hex.DecodeString("000102030405")
	if err := DeserializePacket(&dp, raw); err == nil {
		t.Error("Deserialize should have failed!")
	} else {
		fmt.Printf("Expected error: %s\n", err)
	}
}

func TestTooShortBuffer(t *testing.T) {
	var dp TransformTestPacket
	// The following packet has an error - the slice length is set to 4, but there are only 3 bytes of data
	raw, _ := hex.DecodeString("00A0A1A0A1A2A3A0A1A2A3A4A5A6A77F70F170F1F2F370F1F2F3F4F5F6F70100010203040A74657374737472696E67040506")
	if err := DeserializePacket(&dp, raw); err == nil {
		t.Error("Deserialize should have failed!")
	} else {
		fmt.Printf("Expected error: %s\n", err)
	}
}

type TransformShortPacket struct { // Field names must all be exported!
	TestUint8  uint8
	TestUint16 uint16
	TestUint32 uint32
}

func TestTooLong(t *testing.T) {
	var sp TransformShortPacket
	// The following packet has an error - it is too long and does not have a slice element in the end to store the extra bytes
	raw, _ := hex.DecodeString("0010123031323240")
	if err := DeserializePacket(&sp, raw); err == nil {
		t.Error("Deserialize should have failed!")
	} else {
		fmt.Printf("Expected error: %s\n", err)
	}
}
