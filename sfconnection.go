// Author  Raido Pahtma
// License MIT

// Package moteconnection - A SerialForwarder connection library.
package moteconnection

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync/atomic"
	"time"
)

type connection struct {
	Conn     net.Conn
	Reader   *bufio.Reader
	Incoming chan []byte  // Channel to parent
	Closed   chan bool    // Channel to parent
	IsClosed atomic.Value // Info about the connection itself
}

// SfConnection - A SerialForwarder connection
type SfConnection struct {
	BaseMoteConnection

	Host string
	Port uint16

	connections []*connection
	started     bool
	server      bool

	Protocol string
}

func (sfc *SfConnection) runErrorHandler(err error) error {
	if err != io.EOF && err != io.ErrUnexpectedEOF {
		sfc.Debug.Printf("%s\n", err)
	}
	return err
}

func (conn *connection) runErrorHandler(err error) error {
	if err != io.EOF && err != io.ErrUnexpectedEOF {
		//conn.Sfc.Debug.Printf("%s\n", err)
	}
	return err
}

func (sfc *SfConnection) connectErrorHandler(conn net.Conn, err error) error {
	if conn != nil {
		conn.Close()
	}
	if sfc.autoconnect {
		go sfc.dial(sfc.period)
	}
	return sfc.runErrorHandler(err)
}

// Connect - initiate a synchronous connect
func (sfc *SfConnection) Connect() error {
	sfc.connectlock.Lock()

	sfc.server = false
	sfc.shouldconnect.Store(true)
	sfc.autoconnect = false

	sfc.connectlock.Unlock()

	return sfc.dial(0)
}

// Autoconnect - initiate an asynchronous and self-maintaining connection
func (sfc *SfConnection) Autoconnect(period time.Duration) {
	sfc.connectlock.Lock()
	defer sfc.connectlock.Unlock()

	sfc.server = false
	sfc.shouldconnect.Store(true)
	sfc.autoconnect = true
	sfc.period = period
	go sfc.dial(0)
}

// Listen - Start a server and listen for incoming SF connections
func (sfc *SfConnection) Listen() error {
	sfc.connectlock.Lock()
	defer sfc.connectlock.Unlock()

	sfc.server = true
	sfc.shouldconnect.Store(true)

	if sfc.Protocol == "udp" {
		go sfc.listenUDP()
	} else {
		go sfc.listenTCP()
	}

	return nil
}

func (sfc *SfConnection) connectWatchdog(timeout time.Duration, conn net.Conn) {
	select {
	case v := <-sfc.watchdog:
		if v {
			sfc.Debug.Printf("Watchdog stop.\n")
		} else {
			sfc.Debug.Printf("Watchdog interrupt.\n")
			conn.Close()
		}
	case <-time.After(timeout):
		sfc.Debug.Printf("Timeout.\n")
		conn.Close()
	}
}

func (sfc *SfConnection) listenTCP() {
	l, err := net.Listen("tcp", ":"+strconv.Itoa(int(sfc.Port)))
	if err != nil {
		sfc.Error.Println(err)
		return
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			sfc.Error.Println(err)
			// TODO signal that listener has closed
			return
		}
		sfc.Debug.Printf("Accept %s\n", conn)
		go sfc.connect(conn)
	}
}

func (sfc *SfConnection) listenUDP() {
	uConn, err := net.ListenPacket("udp", ":"+strconv.Itoa(int(sfc.Port)))
	if err != nil {
		return
	}

	if sfc.started == false {
		sfc.started = true
		go sfc.run()
	}

	for {
		buffer := make([]byte, 256)
		n, addr, err := uConn.ReadFrom(buffer)
		fmt.Printf("%s %d\n", addr, n)
		if err != nil {
			sfc.Error.Println(err)
			// TODO signal that listener has closed
			return
		}
		sfc.incoming <- buffer[12:n]
		//go sfc.connect(conn)
	}
}

func (sfc *SfConnection) dial(delay time.Duration) error {
	time.Sleep(delay)

	sfc.connectlock.Lock()
	defer sfc.connectlock.Unlock()

	if sfc.connected.Load().(bool) {
		return errors.New("Already connected")
	}

	if sfc.shouldconnect.Load().(bool) == false {
		return errors.New("Connect interrupted")
	}

	constring := sfc.Host + ":" + strconv.Itoa(int(sfc.Port))

	sfc.Info.Printf("Connecting to %s\n", constring)
	conn, err := net.Dial(sfc.Protocol, constring)
	if err != nil {
		return sfc.connectErrorHandler(conn, err)
	}

	return sfc.connect(conn)
}

func (sfc *SfConnection) connect(conn net.Conn) error {
	if sfc.Protocol == "tcp" {
		return sfc.connectTCP(conn)
	}
	if sfc.Protocol == "udp" {
		return sfc.connectUDP(conn)
	}
	return fmt.Errorf("Unknown protocol '%s'", sfc.Protocol)
}

func (sfc *SfConnection) connectUDP(conn net.Conn) error {
	connbuf := bufio.NewReader(conn)

	sfc.notifyWatchdog(true)
	sfc.connected.Store(true)

	var c connection
	c.Conn = conn
	c.Reader = connbuf
	c.Incoming = sfc.incoming
	c.Closed = sfc.closed

	sfc.connectlock.Lock()
	sfc.connections = append(sfc.connections, &c)
	sfc.connectlock.Unlock()

	sfc.Info.Printf("Connection established.\n")

	go c.read()

	if sfc.started == false {
		sfc.started = true
		go sfc.run()
	}

	return nil
}

func (sfc *SfConnection) connectTCP(conn net.Conn) error {

	go sfc.connectWatchdog(30*time.Second, conn)

	_, err := conn.Write([]byte("U "))
	if err != nil {
		return sfc.connectErrorHandler(conn, err)
	}

	connbuf := bufio.NewReader(conn)

	protocol := make([]byte, 2)
	_, err = io.ReadFull(connbuf, protocol)
	if err == nil {
		if string(protocol) == "U " {
			var c connection
			sfc.notifyWatchdog(true)
			sfc.connected.Store(true)
			c.Conn = conn
			c.Reader = connbuf
			c.Incoming = sfc.incoming
			c.Closed = sfc.closed

			sfc.connectlock.Lock()
			sfc.connections = append(sfc.connections, &c)
			sfc.connectlock.Unlock()

			sfc.Info.Printf("Connection established.\n")

			go c.read()

			if sfc.started == false {
				sfc.started = true
				go sfc.run()
			}

			return nil
		}

		return sfc.connectErrorHandler(conn, fmt.Errorf("Unsupported protocol %s", string(protocol)))
	}

	return sfc.connectErrorHandler(conn, err)
}

func (conn *connection) read() {
	for {
		len, err := conn.Reader.ReadByte()
		if err == nil {
			msg := make([]byte, len)
			_, err := io.ReadFull(conn.Reader, msg)
			if err == nil {
				conn.Incoming <- msg
			} else {
				conn.runErrorHandler(err)
				break
			}
		} else {
			conn.runErrorHandler(err)
			break
		}
	}
	conn.IsClosed.Store(true)
	conn.Closed <- true
}

func (sfc *SfConnection) run() {
	closing := false
	for {
		select {
		case msg := <-sfc.incoming:
			sfc.Debug.Printf("RCV(%d): %X\n", len(msg), msg)
			if len(msg) > 0 {
				if dispatcher, ok := sfc.dispatchers[msg[0]]; ok {
					err := dispatcher.Receive(msg)
					if err != nil {
						sfc.Debug.Printf("Dispatcher error: %s", err)
					}
				} else {
					sfc.Debug.Printf("No dispatcher for %02X!\n", msg[0])
				}
			}
		case msg := <-sfc.outgoing:
			length := make([]byte, 1)
			length[0] = byte(len(msg))
			for _, conn := range sfc.connections {
				sfc.Debug.Printf("SND %s->%s (%d):%X\n", conn.Conn.LocalAddr(), conn.Conn.RemoteAddr(), length[0], msg)
				conn.Conn.Write(length)
				conn.Conn.Write(msg)
			}
		case <-sfc.close:
			for _, conn := range sfc.connections {
				conn.Conn.Close()
			}
			closing = true
		case <-sfc.closed:
			sfc.Debug.Printf("Connection closed.\n")
			sfc.connectlock.Lock()

			// Connection cleanup
			var conns []*connection
			for _, conn := range sfc.connections {
				if conn.IsClosed.Load() == nil {
					conns = append(conns, conn)
				}
			}
			sfc.connections = conns

			if len(sfc.connections) == 0 {
				sfc.connected.Store(false)

				if sfc.server {
					sfc.Debug.Printf("No connections left.\n")
				}
				if closing {
					return
				} else if sfc.autoconnect {
					go sfc.dial(sfc.period)
				}
			}

			sfc.connectlock.Unlock()
		case dispatcher := <-sfc.removeDispatcher:
			if sfc.dispatchers[dispatcher.Dispatch()] != dispatcher {
				panic("Asked to remove a dispatcher that is not registered!")
			}
			delete(sfc.dispatchers, dispatcher.Dispatch())
		case dispatcher := <-sfc.addDispatcher:
			sfc.dispatchers[dispatcher.Dispatch()] = dispatcher
		}
	}
}

// NewSfConnection - create a new connection
func NewSfConnection(host string, port uint16) *SfConnection {
	sfc := new(SfConnection)
	sfc.InitLoggers()
	sfc.connections = make([]*connection, 0)
	sfc.Host = host
	sfc.Port = port
	sfc.dispatchers = make(map[byte]Dispatcher)
	sfc.removeDispatcher = make(chan Dispatcher)
	sfc.addDispatcher = make(chan Dispatcher)
	sfc.outgoing = make(chan []byte)
	sfc.incoming = make(chan []byte)
	sfc.watchdog = make(chan bool)
	sfc.closed = make(chan bool)
	sfc.close = make(chan bool)
	sfc.connected.Store(false)
	sfc.Protocol = "tcp"
	return sfc
}
