# go-moteconnection
SmartDust mote serial and SerialForwarder connection library for go.

Supports the TinyOS-inspired [serial](https://github.com/proactivity-lab/docs/wiki/Serial-protocol)
and [serial-forwarder](https://github.com/proactivity-lab/docs/wiki/SerialForwarder-protocol)
transports.

# sforwarder

sforwarder is a go implementation of the serial-forwarder application with
some additional features:

 * sforwarder correctly increments sequence numbers
 * sforwarder will recover from a UART/USB disconnect
 * sforwarder can act as a server or client on both ends
 * sforwarder can also forward a TCP source connection

sforwarder can be cross-compiled (see the [sforwarder/Makefile](sforwarder/Makefile))
and packaged as a Debian package with `dpkg-buildpackage -b --no-sign`.

# sfserver

sfserver creates a serial-forwarder server that prints out received packets.
May be useful for debugging other applications wanting to connect to a network.

# amlistener

amlistener connects to a serial-forwarder server and prints out received
ActiveMessage packets (dispatcher 0x00).

https://github.com/proactivity-lab/go-amlistener
