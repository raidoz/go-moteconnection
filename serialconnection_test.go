// Author  Raido Pahtma
// License MIT

package moteconnection

import "testing"
import "time"
import "fmt"
import "log"
import "os"

func TestSomePortConnection(t *testing.T) {
	ports, _ := ListSerialPorts()
	for _, port := range ports {
		fmt.Printf("Found port: %v\n", port)
	}

	if len(ports) == 0 {
		t.Error(fmt.Sprintf("No serial ports found, need at least one to run tests"))
	}

	conn := NewSerialConnection(ports[0], 115200)
	// Configure logging
	logformat := log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile
	conn.SetDebugLogger(log.New(os.Stdout, "DEBUG: ", logformat))
	conn.SetInfoLogger(log.New(os.Stdout, "INFO:  ", logformat))
	conn.SetWarningLogger(log.New(os.Stdout, "WARN:  ", logformat))
	conn.SetErrorLogger(log.New(os.Stdout, "ERROR: ", logformat))

	err := conn.Connect()
	if err != nil {
		t.Error(err)
	}

	//conn.read()
	time.Sleep(5 * time.Second)
}
