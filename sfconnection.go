// Author  Raido Pahtma
// License MIT

// SerialForwarder connection library.
package sfconnection

import "net"

import "fmt"
import "strconv"
import "bufio"
import "io"
import "time"
import "errors"
import "sync"
import "sync/atomic"

import "github.com/proactivity-lab/go-loggers"

type Packet interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error
}

type PacketFactory interface {
	New() Packet
}

type SfConnection struct {
	loggers.DIWEloggers

	Host string
	Port uint16

	connectlock   sync.Mutex
	shouldconnect bool
	autoconnect   bool
	period        time.Duration

	connected atomic.Value
	outgoing  chan []byte
	incoming  chan []byte

	dispatchers map[byte]Dispatcher

	conn    net.Conn
	connbuf *bufio.Reader

	watchdog chan bool
	closed   chan bool
}

func (self *SfConnection) runErrorHandler(err error) error {
	if err != io.EOF && err != io.ErrUnexpectedEOF {
		self.Debug.Printf("%s\n", err)
	}
	return err
}

func (self *SfConnection) connectErrorHandler(conn net.Conn, err error) error {
	if conn != nil {
		conn.Close()
	}
	if self.autoconnect {
		go self.connect(self.period)
	}
	return self.runErrorHandler(err)
}

func (self *SfConnection) Connected() bool {
	return self.connected.Load().(bool)
}

func (self *SfConnection) Disconnect() {
	self.notifyWatchdog(false)

	self.connectlock.Lock()
	defer self.connectlock.Unlock()

	self.shouldconnect = false
	self.autoconnect = false
	if self.connected.Load().(bool) {
		self.conn.Close()
	}
}

func (self *SfConnection) Connect(host string, port uint16) error {
	self.connectlock.Lock()

	self.Host = host
	self.Port = port
	self.shouldconnect = true
	self.autoconnect = false

	self.connectlock.Unlock()

	return self.connect(0)
}

func (self *SfConnection) Autoconnect(host string, port uint16, period time.Duration) {
	self.connectlock.Lock()
	defer self.connectlock.Unlock()

	self.Host = host
	self.Port = port
	self.shouldconnect = true
	self.autoconnect = true
	self.period = period
	go self.connect(0)
}

func (self *SfConnection) notifyWatchdog(outcome bool) {
	select {
	case self.watchdog <- outcome:
	case <-time.After(time.Second):
		self.Debug.Printf("No watchdog?\n")
	}
}

func (self *SfConnection) connectWatchdog(timeout time.Duration, conn net.Conn) {
	select {
	case v := <-self.watchdog:
		if v {
			self.Debug.Printf("Watchdog stop.\n")
			return
		} else {
			self.Debug.Printf("Watchdog interrupt.\n")
			conn.Close()
		}
	case <-time.After(timeout):
		self.Debug.Printf("Timeout.\n")
		conn.Close()
	}
}

func (self *SfConnection) connect(delay time.Duration) error {
	time.Sleep(delay)

	self.connectlock.Lock()
	defer self.connectlock.Unlock()

	if self.connected.Load().(bool) {
		return errors.New("Already connected.")
	}

	if self.shouldconnect == false {
		return errors.New("Connect interrupted.")
	}

	constring := self.Host + ":" + strconv.Itoa(int(self.Port))

	self.Debug.Printf("Dialing %s\n", constring)
	conn, err := net.Dial("tcp", constring)
	if err != nil {
		return self.connectErrorHandler(conn, err)
	}

	go self.connectWatchdog(30*time.Second, conn)

	_, err = conn.Write([]byte("U "))
	if err != nil {
		return self.connectErrorHandler(conn, err)
	}

	connbuf := bufio.NewReader(conn)

	protocol := make([]byte, 2)
	_, err = io.ReadFull(connbuf, protocol)
	if err == nil {
		if string(protocol) == "U " {
			self.notifyWatchdog(true)
			self.connected.Store(true)
			self.conn = conn
			self.connbuf = connbuf

			self.Debug.Printf("Connection established.\n")

			go self.read()
			go self.run()

			return nil
		} else {
			return self.connectErrorHandler(conn, errors.New(fmt.Sprintf("Unsupported protocol %s!", string(protocol))))
		}
	} else {
		return self.connectErrorHandler(conn, err)
	}
}

func (self *SfConnection) read() {
	for {
		len, err := self.connbuf.ReadByte()
		if err == nil {
			msg := make([]byte, len)
			_, err := io.ReadFull(self.connbuf, msg)
			if err == nil {
				self.incoming <- msg
			} else {
				self.runErrorHandler(err)
				self.closed <- true
				break
			}
		} else {
			self.runErrorHandler(err)
			self.closed <- true
			break
		}
	}
}

func (self *SfConnection) run() {
	for {
		select {
		case msg := <-self.incoming:
			self.Debug.Printf("RCV(%d): %X\n", len(msg), msg)
			if len(msg) > 0 {
				if dispatcher, ok := self.dispatchers[msg[0]]; ok {
					dispatcher.Receive(msg)
				} else {
					self.Error.Printf("No dispatcher for %d!\n", msg[0])
				}
			}
		case msg := <-self.outgoing:
			length := make([]byte, 1)
			length[0] = byte(len(msg))
			self.Debug.Printf("SND(%d): %X\n", length[0], msg)
			self.conn.Write(length)
			self.conn.Write(msg)
		case <-self.closed:
			self.Debug.Printf("Connection closed.\n")
			self.connectlock.Lock()
			defer self.connectlock.Unlock()

			self.connected.Store(false)
			self.conn.Close()
			if self.autoconnect {
				go self.connect(self.period)
			}
			return
		}
	}
}

func (self *SfConnection) Send(msg Packet) error {
	serialized, err := msg.Serialize()
	if err != nil {
		return err
	}
	select {
	case self.outgoing <- serialized:
		return nil
	case <-time.After(50 * time.Millisecond): // Because the run goroutine might be doing something other than reading self.outgoing at this very moment
		return errors.New("Not connected!")
	}
}

func (self *SfConnection) AddDispatcher(dispatcher Dispatcher) error {
	self.dispatchers[dispatcher.Header()] = dispatcher
	return nil
}

func NewSfConnection() *SfConnection {
	sfc := new(SfConnection)
	sfc.InitLoggers()
	sfc.dispatchers = make(map[byte]Dispatcher)
	sfc.outgoing = make(chan []byte)
	sfc.incoming = make(chan []byte)
	sfc.watchdog = make(chan bool)
	sfc.closed = make(chan bool)
	sfc.connected.Store(false)
	return sfc
}
