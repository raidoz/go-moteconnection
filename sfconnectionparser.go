// Author  Raido Pahtma
// License MIT

package sfconnection

import "fmt"
import "regexp"
import "strconv"
import "errors"

func ParseSfConnectionString(connectionstring string) (string, uint16, error) {
	if connectionstring != "" {
		re := regexp.MustCompile("sf@([a-zA-Z0-9.-]+)(:([0-9]+))?\\z")
		match := re.FindStringSubmatch(connectionstring) // [sf@localhost:9002 localhost :9002 9002]
		//fmt.Printf("%s\n", match)
		if len(match) == 4 {
			host := match[1]
			if len(match[3]) > 0 {
				p, err := strconv.ParseUint(match[3], 10, 16)
				if err == nil {
					return host, uint16(p), nil
				} else {
					err := errors.New(fmt.Sprintf("%s cannot be used as a TCP port number!", match[2]))
					return "", 0, err
				}
			} else {
				return host, 9002, nil
			}
		} else {
			err := errors.New(fmt.Sprintf("ERROR: %s cannot be used as a connectionstring!", connectionstring))
			return "", 0, err
		}
	}
	return "localhost", 9002, nil
}
