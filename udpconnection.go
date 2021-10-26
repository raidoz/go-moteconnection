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

type uconnection struct {
	Conn     net.Conn
	Reader   *bufio.Reader
	Incoming chan []byte  // Channel to parent
	Closed   chan bool    // Channel to parent
	IsClosed atomic.Value // Info about the connection itself
}

// UDPConnection - A SerialForwarder connection
type UDPConnection struct {
	BaseMoteConnection

	Host string
	Port uint16

	connections []connection
	started     bool
	server      bool
}

func (sfc *UDPConnection) runErrorHandler(err error) error {
	if err != io.EOF && err != io.ErrUnexpectedEOF {
		sfc.Debug.Printf("%s\n", err)
	}
	return err
}

func (conn *uconnection) runErrorHandler(err error) error {
	if err != io.EOF && err != io.ErrUnexpectedEOF {
		//conn.Sfc.Debug.Printf("%s\n", err)
	}
	return err
}

func (sfc *UDPConnection) connectErrorHandler(conn net.Conn, err error) error {
	if conn != nil {
		conn.Close()
	}
	if sfc.autoconnect {
		go sfc.dial(sfc.period)
	}
	return sfc.runErrorHandler(err)
}

// Connect - initiate a synchronous connect
func (sfc *UDPConnection) Connect() error {
	sfc.connectlock.Lock()

	sfc.server = false
	sfc.shouldconnect.Store(true)
	sfc.autoconnect = false

	sfc.connectlock.Unlock()

	return sfc.dial(0)
}

// Autoconnect - initiate an asynchronous and self-maintaining connection
func (sfc *UDPConnection) Autoconnect(period time.Duration) {
	sfc.connectlock.Lock()
	defer sfc.connectlock.Unlock()

	sfc.server = false
	sfc.shouldconnect.Store(true)
	sfc.autoconnect = true
	sfc.period = period
	go sfc.dial(0)
}

// Listen - Start a server and listen for incoming SF connections
func (sfc *UDPConnection) Listen() error {
	sfc.connectlock.Lock()
	defer sfc.connectlock.Unlock()

	sfc.server = true
	sfc.shouldconnect.Store(true)

	go sfc.listenUDP()

	return nil
}

func (sfc *UDPConnection) connectWatchdog(timeout time.Duration, conn net.Conn) {
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

func (sfc *UDPConnection) listenUDP() {
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
		fmt.Printf("uConn: %s %d\n", addr, n)
		if err != nil {
			sfc.Error.Println(err)
			// TODO signal that listener has closed?
			return
		}
		if n > 12 {
			sfc.incoming <- buffer[12:n]
		} else {
			sfc.Warning.Printf("RCV short %x", buffer[0:n])
		}
	}
}

func (sfc *UDPConnection) dial(delay time.Duration) error {
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
	conn, err := net.Dial("udp", constring)
	if err != nil {
		return sfc.connectErrorHandler(conn, err)
	}

	return sfc.connect(conn)
}

func (sfc *UDPConnection) connect(conn net.Conn) error {
	connbuf := bufio.NewReader(conn)

	sfc.notifyWatchdog(true)
	sfc.connected.Store(true)

	var c connection
	c.Conn = conn
	c.Reader = connbuf
	c.Incoming = sfc.incoming
	c.Closed = sfc.closed
	sfc.connections = append(sfc.connections, c)

	sfc.Info.Printf("Connection established.\n")

	go c.read()

	if sfc.started == false {
		sfc.started = true
		go sfc.run()
	}

	return nil
}

func (conn *uconnection) read() {
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

func (sfc *UDPConnection) run() {
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
			sfc.Debug.Printf("SND(%d): %X\n", length[0], msg)
			for _, conn := range sfc.connections {
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
			var conns []connection
			for _, conn := range sfc.connections {
				if conn.IsClosed.Load() == false {
					conns = append(conns, conn)
				}
			}
			sfc.connections = conns

			sfc.connected.Store(false)
			sfc.connectlock.Unlock()

			if len(sfc.connections) == 0 {
				if sfc.server {
					sfc.Debug.Printf("No connections left.\n")
				}
				if closing {
					return
				} else if sfc.autoconnect {
					go sfc.dial(sfc.period)
				}
			}
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

// NewUDPConnection - create a new connection
func NewUDPConnection(host string, port uint16) *UDPConnection {
	sfc := new(UDPConnection)
	sfc.InitLoggers()
	sfc.connections = make([]connection, 0)
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
	return sfc
}
