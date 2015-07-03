// Author  Raido Pahtma
// License MIT

package sfconnection

import "testing"
import "fmt"

func TestConnectionParser(t *testing.T) {
	host, port, err := ParseSfConnectionString("sf@host:9000")
	if err != nil {
		t.Error(err)
	} else if host != "host" || port != 9000 {
		t.Error(fmt.Sprintf("error %s %d", host, port))
	}

	host, port, err = ParseSfConnectionString("sf@host")
	if err != nil {
		t.Error(err)
	} else if host != "host" || port != 9002 {
		t.Error(fmt.Sprintf("error %s %d", host, port))
	}

	host, port, err = ParseSfConnectionString("")
	if err != nil {
		t.Error(err)
	} else if host != "localhost" || port != 9002 {
		t.Error(fmt.Sprintf("error %s %d", host, port))
	}

	host, port, err = ParseSfConnectionString("sf@host:")
	if err == nil {
		t.Error(err)
	}

	host, port, err = ParseSfConnectionString("sf@host:a")
	if err == nil {
		t.Error(err)
	}

	host, port, err = ParseSfConnectionString("sf@/*?")
	if err == nil {
		t.Error(err)
	}

	host, port, err = ParseSfConnectionString("sf@")
	if err == nil {
		t.Error(err)
	}

}
