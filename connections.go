// Author  Raido Pahtma
// License MIT

package moteconnection

import "fmt"
import "regexp"
import "strconv"
import "errors"

func CreateConnection(connectionstring string) (MoteConnection, string, error) {
	if connectionstring != "" {
		re := regexp.MustCompile("sf@([a-zA-Z0-9.-]+)(:([0-9]+))?\\z")
		match := re.FindStringSubmatch(connectionstring) // [sf@localhost:9002 localhost :9002 9002]
		//fmt.Printf("%s\n", match)
		if len(match) == 4 {
			host := match[1]
			if len(match[3]) > 0 {
				p, err := strconv.ParseUint(match[3], 10, 16)
				if err == nil {
					return NewSfConnection(host, uint16(p)), fmt.Sprintf("sf@%s:%d", host, uint16(p)), nil
				} else {
					return nil, "", errors.New(fmt.Sprintf("%s cannot be used as a TCP port number!", match[2]))
				}
			} else {
				return NewSfConnection(host, 9002), fmt.Sprintf("sf@%s:%d", host, uint16(9002)), nil
			}
		} else {
			re = regexp.MustCompile("serial@([a-zA-Z0-9.-/]+)(:([0-9]+))?\\z")
			match = re.FindStringSubmatch(connectionstring) // [serial@/dev/ttyUSB0:115200 /dev/ttyUSB0 :115200 115200]
			//fmt.Printf("%s\n", match)l
			if len(match) == 4 {
				port := match[1]
				if len(match[3]) > 0 {
					p, err := strconv.ParseUint(match[3], 10, 32)
					if err == nil {
						return NewSerialConnection(port, int(p)), fmt.Sprintf("serial@%s:%d", port, int(p)), nil
					} else {
						return nil, "", errors.New(fmt.Sprintf("%s cannot be used as a baudrate!", match[2]))
					}
				} else {
					return NewSerialConnection(port, 115200), fmt.Sprintf("serial@%s:%d", port, 115200), nil
				}
			} else {
				return nil, "", errors.New(fmt.Sprintf("ERROR: %s cannot be used as a connectionstring!", connectionstring))
			}
		}
	}
	return NewSfConnection("localhost", 9002), "sf@localhost:9002", nil
}
