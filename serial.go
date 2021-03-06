package serial

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/istperm/utils"
	//"main/utils"
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
	logBuf [128]byte
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
	f, e := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
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
			b := p.logBuf[i]
			hex.WriteString(fmt.Sprintf("%02X ", b))
			r := '.'
			if b >= 0x20 {
				r = utils.CharToRune(b)
			}
			asc.WriteRune(r)
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
