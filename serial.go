package serial

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type Config struct {
	Name        string
	Baud        int
	ReadTimeout time.Duration
	LogFile     string

	// Size     int
	// Parity   SomeNewTypeToGetCorrectDefaultOf_None
	StopBits int

	// RTSFlowControl bool
	// DTRFlowControl bool
	// XONFlowControl bool
	// CRLFTranslate bool
}

type BasePort struct {
	f      *os.File
	logger *log.Logger
	logTag rune
	logBuf [64]byte
	logPtr int
}

type SerialError struct {
	Tag string
	Msg string
	Cod int
}

func (se SerialError) Error() string {
	var sb strings.Builder
	if se.Tag != "" {
		sb.WriteString("[" + se.Tag + "] ")
	}
	sb.WriteString(se.Msg)
	if se.Cod != 0 {
		sb.WriteString(" [" + strconv.Itoa(se.Cod) + "]")
	}
	return sb.String()
}

// OpenPort opens a serial port with the specified configuration
func OpenPort(c *Config) (*Port, error) {
	//return openPort(c.Name, c.Baud, c.ReadTimeout)
	// call platform-specific function
	p, err := openPort(c)
	if p != nil && err == nil && c.LogFile != "" {
		err = p.openLog(c.LogFile)
		p.logMsg("Open", c.Name)
	}
	return p, err
}

func (p *BasePort) openLog(logFile string) error {
	f, e := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0755)
	if e == nil {
		p.logger = log.New(f, "", log.LstdFlags)
	}
	return e
}

func (p *BasePort) logMsg(tag string, msg string, arg ...interface{}) {
	if p.logger == nil {
		return
	}
	p.logFlush()
	if tag != "" {
		msg = "[" + tag + "] " + msg
	}
	p.logger.Printf(msg, arg...)
}

func (p *BasePort) logData(tag rune, data []byte) {
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

func (p *BasePort) logFlush() {
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

func (p *BasePort) Close() (err error) {
	p.logFlush()
	p.logMsg("Close", "")
	return p.f.Close()
}

func (p *BasePort) Read(buf []byte) (n int, err error) {
	n, err = p.f.Read(buf)
	if err != nil && err != io.EOF {
		p.logMsg("Read", "Error %d", err)
		return 0, err
	} else if n > 0 {
		p.logData('+', buf)
		return n, nil
	}
	return 0, nil
}

func (p *BasePort) Write(buf []byte) (n int, err error) {
	n, err = p.f.Write(buf)
	if err != nil {
		p.logMsg("Write", err.Error())
	} else if n > 0 {
		p.logData('-', buf)
	}
	return
}

func (p *BasePort) SetDtr(v bool) error {
	return p.setModemLine("DTR", syscall.TIOCM_DTR, v)
}

func (p *BasePort) SetRts(v bool) error {
	return p.setModemLine("RTS", syscall.TIOCM_RTS, v)
}

func (p *BasePort) setModemLine(tag string, line uint, v bool) error {
	req := syscall.TIOCMBIC
	if v {
		req = syscall.TIOCMBIS
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		p.f.Fd(),
		uintptr(req),
		uintptr(unsafe.Pointer(&line)),
	)
	if errno != 0 {
		p.logMsg(tag, "%t -> error %s [%d]", v, errno.Error(), errno)
		return errno
	} else {
		p.logMsg(tag, "%t", v)
		return nil
	}
}

// Converts the timeout values for Linux / POSIX systems
func posixTimeoutValues(readTimeout time.Duration) (vmin uint8, vtime uint8) {
	// set blocking / non-blocking read
	vmin = 1
	vtime = 0
	if readTimeout > 0 {
		// EOF on zero read
		vmin = 0
		// convert timeout to deciseconds as expected by VTIME
		vt := (readTimeout.Nanoseconds() / 1e6 / 100)
		// capping the timeout
		if vt < 1 {
			// min possible timeout 1 Deciseconds (0.1s)
			vtime = 1
		} else if vt > 255 {
			// max possible timeout is 255 deciseconds (25.5s)
			vtime = 255
		} else {
			vtime = uint8(vt)
		}
	}
	return
}
