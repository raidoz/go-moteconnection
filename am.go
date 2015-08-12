// Author  Raido Pahtma
// License MIT

package sfconnection

import "encoding/hex"
import "strconv"
import "fmt"

type HexByte byte
type HexString []byte
type AMAddr uint16
type AMID uint8
type AMGroup uint8

func (self HexByte) String() string {
	return fmt.Sprintf("%02X", byte(self))
}

func (self *HexByte) UnmarshalFlag(value string) error {
	data, err := strconv.ParseUint(value, 16, 8)
	*self = HexByte(data)
	return err
}

func (self HexByte) MarshalFlag() (string, error) {
	return self.String(), nil
}

func (self HexString) String() string {
	return hex.EncodeToString(self)
}

func (self *HexString) UnmarshalFlag(value string) error {
	data, err := hex.DecodeString(value)
	*self = data
	return err
}

func (self HexString) MarshalFlag() (string, error) {
	return self.String(), nil
}

func (self AMAddr) String() string {
	return fmt.Sprintf("%04X", uint16(self))
}

func (self *AMAddr) UnmarshalFlag(value string) error {
	data, err := strconv.ParseUint(value, 16, 16)
	*self = AMAddr(data)
	return err
}

func (self AMAddr) MarshalFlag() (string, error) {
	return self.String(), nil
}

func (self AMID) String() string {
	return fmt.Sprintf("%02X", uint8(self))
}

func (self *AMID) UnmarshalFlag(value string) error {
	data, err := strconv.ParseUint(value, 16, 8)
	*self = AMID(data)
	return err
}

func (self AMID) MarshalFlag() (string, error) {
	return self.String(), nil
}

func (self AMGroup) String() string {
	return fmt.Sprintf("%02X", uint8(self))
}

func (self *AMGroup) UnmarshalFlag(value string) error {
	data, err := strconv.ParseUint(value, 16, 8)
	*self = AMGroup(data)
	return err
}

func (self AMGroup) MarshalFlag() (string, error) {
	return self.String(), nil
}
