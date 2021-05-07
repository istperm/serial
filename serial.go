/*
Goserial is a simple go package to allow you to read and write from
the serial port as a stream of bytes.

It aims to have the same API on all platforms, including windows.  As
an added bonus, the windows package does not use cgo, so you can cross
compile for windows from another platform.  Unfortunately goinstall
does not currently let you cross compile so you will have to do it
manually:

 GOOS=windows make clean install

Currently there is very little in the way of configurability.  You can
set the baud rate.  Then you can Read(), Write(), or Close() the
connection.  Read() will block until at least one byte is returned.
Write is the same.  There is currently no exposed way to set the
timeouts, though patches are welcome.

Currently all ports are opened with 8 data bits, 1 stop bit, no
parity, no hardware flow control, and no software flow control.  This
works fine for many real devices and many faux serial devices
including usb-to-serial converters and bluetooth serial ports.

You may Read() and Write() simulantiously on the same connection (from
different goroutines).

Example usage:

  package main

  import (
        "github.com/tarm/goserial"
        "log"
  )

  func main() {
        c := &serial.Config{Name: "COM5", Baud: 115200}
        s, err := serial.OpenPort(c)
        if err != nil {
                log.Fatal(err)
        }

        n, err := s.Write([]byte("test"))
        if err != nil {
                log.Fatal(err)
        }

        buf := make([]byte, 128)
        n, err = s.Read(buf)
        if err != nil {
                log.Fatal(err)
        }
        log.Print("%q", buf[:n])
  }
*/
package serial

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Config contains the information needed to open a serial port.
//
// Currently few options are implemented, but more may be added in the
// future (patches welcome), so it is recommended that you create a
// new config addressing the fields by name rather than by order.
//
// For example:
//
//    c0 := &serial.Config{Name: "COM45", Baud: 115200, ReadTimeout: time.Millisecond * 500}
// or
//    c1 := new(serial.Config)
//    c1.Name = "/dev/tty.usbserial"
//    c1.Baud = 115200
//    c1.ReadTimeout = time.Millisecond * 500
//
type Config struct {
	Name        string
	Baud        int
	ReadTimeout time.Duration // Total timeout
	LogFile     string

	// Size     int // 0 get translated to 8
	// Parity   SomeNewTypeToGetCorrectDefaultOf_None
	// StopBits SomeNewTypeToGetCorrectDefaultOf_1

	// RTSFlowControl bool
	// DTRFlowControl bool
	// XONFlowControl bool

	// CRLFTranslate bool
}

// OpenPort opens a serial port with the specified configuration
func OpenPort(c *Config) (*Port, error) {
	p, err := openPort(c.Name, c.Baud, c.ReadTimeout)
	if p != nil && err == nil && c.LogFile != "" {
		err = p.openLog(c.LogFile)
		p.logMsg("Open", c.Name)
	}
	return p, err
}

func (p *Port) openLog(logFile string) error {
	f, e := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0755)
	if e == nil {
		p.logger = log.New(f, "", log.LstdFlags)
	}
	return e
}

func (p *Port) logMsg(tag string, msg string, arg ...interface{}) {
	if p.logger == nil {
		return
	}
	p.logFlush()
	if tag != "" {
		msg = "[" + tag + "] " + msg
	}
	p.logger.Printf(msg, arg...)
}

func (p *Port) logData(tag rune, data []byte) {
	if p.logger == nil {
		return
	}
	if tag != p.logTag {
		p.logFlush()
		p.logTag = tag
	}
	for i := 0; i < len(data); i++ {
		if p.logPtr >= len(p.logBuf) {
			p.logFlush()
		}
		p.logBuf[p.logPtr] = data[i]
		p.logPtr++
	}
}

func (p *Port) logFlush() {
	if p.logger != nil && p.logPtr > 0 {
		var hex, asc strings.Builder
		tag := p.logTag
		if tag == 0 {
			tag = ' '
		}
		for i := 0; i < p.logPtr; i++ {
			if i%16 == 0 && hex.Cap() > 0 {
				p.logger.Printf("%c %s %s", tag, hex.String(), asc.String())
				hex.Reset()
				asc.Reset()
			}
			hex.WriteString(fmt.Sprintf("%02X ", p.logBuf[i]))
			c := rune(p.logBuf[i])
			if c < 0x20 {
				c = '.'
			}
			asc.WriteRune(c)
		}
		if hex.Cap() > 0 {
			p.logger.Printf("%c %-48s %s", tag, hex.String(), asc.String())
		}
	}
	p.logPtr = 0
}

// Converts the timeout values for Linux / POSIX systems
func posixTimeoutValues(readTimeout time.Duration) (vmin uint8, vtime uint8) {
	const MAXUINT8 = 1<<8 - 1 // 255
	// set blocking / non-blocking read
	var minBytesToRead uint8 = 1
	var readTimeoutInDeci int64
	if readTimeout > 0 {
		// EOF on zero read
		minBytesToRead = 0
		// convert timeout to deciseconds as expected by VTIME
		readTimeoutInDeci = (readTimeout.Nanoseconds() / 1e6 / 100)
		// capping the timeout
		if readTimeoutInDeci < 1 {
			// min possible timeout 1 Deciseconds (0.1s)
			readTimeoutInDeci = 1
		} else if readTimeoutInDeci > MAXUINT8 {
			// max possible timeout is 255 deciseconds (25.5s)
			readTimeoutInDeci = MAXUINT8
		}
	}
	return minBytesToRead, uint8(readTimeoutInDeci)
}

// func SendBreak()

// func RegisterBreakHandler(func())
