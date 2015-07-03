// Author  Raido Pahtma
// License MIT

package sfconnection

import "strconv"
import "fmt"

type AMAddr uint16
type AMID uint8
type AMGroup uint8

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
