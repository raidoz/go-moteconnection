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

func (self *HexByte) UnmarshalFlag(value string) error {
	data, err := strconv.ParseUint(value, 16, 8)
	*self = HexByte(data)
	return err
}

func (self HexByte) MarshalFlag() (string, error) {
	return fmt.Sprintf("%02X", self), nil
}

func (self *HexString) UnmarshalFlag(value string) error {
	data, err := hex.DecodeString(value)
	*self = data
	return err
}

func (self HexString) MarshalFlag() (string, error) {
	return hex.EncodeToString(self), nil
}

func (self *AMAddr) UnmarshalFlag(value string) error {
	data, err := strconv.ParseUint(value, 16, 16)
	*self = AMAddr(data)
	return err
}

func (self AMAddr) MarshalFlag() (string, error) {
	return fmt.Sprintf("%04X", self), nil
}

func (self *AMID) UnmarshalFlag(value string) error {
	data, err := strconv.ParseUint(value, 16, 8)
	*self = AMID(data)
	return err
}

func (self AMID) MarshalFlag() (string, error) {
	return fmt.Sprintf("%02X", self), nil
}

func (self *AMGroup) UnmarshalFlag(value string) error {
	data, err := strconv.ParseUint(value, 16, 8)
	*self = AMGroup(data)
	return err
}

func (self AMGroup) MarshalFlag() (string, error) {
	return fmt.Sprintf("%02X", self), nil
}
