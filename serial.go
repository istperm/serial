package serial

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
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
			b := p.logBuf[i]
			hex.WriteString(fmt.Sprintf("%02X ", b))
			c := '.'
			if b >= 0x20 {
				c = AsciiToUnicode(b)
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

func AsciiToUnicode(b byte) rune {
	switch b {
	case 0xC0, 0xE0:
		return 'А'
	case 0xC1, 0xE1:
		return 'Б'
	case 0xC2, 0xE2:
		return 'В'
	case 0xC3, 0xE3:
		return 'Г'
	case 0xC4, 0xE4:
		return 'Д'
	case 0xC5, 0xE5:
		return 'Е'
	case 0xC6, 0xE6:
		return 'Ж'
	case 0xC7, 0xE7:
		return 'З'
	case 0xC8, 0xE8:
		return 'И'
	case 0xC9, 0xE9:
		return 'Й'
	case 0xCA, 0xEA:
		return 'К'
	case 0xCB, 0xEB:
		return 'Л'
	case 0xCC, 0xEC:
		return 'М'
	case 0xCD, 0xED:
		return 'Н'
	case 0xCE, 0xEE:
		return 'О'
	case 0xCF, 0xEF:
		return 'П'

	case 0xD0, 0xF0:
		return 'Р'
	case 0xD1, 0xF1:
		return 'С'
	case 0xD2, 0xF2:
		return 'Т'
	case 0xD3, 0xF3:
		return 'У'
	case 0xD4, 0xF4:
		return 'Ф'
	case 0xD5, 0xF5:
		return 'Х'
	case 0xD6, 0xF6:
		return 'Ц'
	case 0xD7, 0xF7:
		return 'Ч'
	case 0xD8, 0xF8:
		return 'Ш'
	case 0xD9, 0xF9:
		return 'Щ'
	case 0xDA, 0xFA:
		return 'Ъ'
	case 0xDB, 0xFB:
		return 'Ы'
	case 0xDC, 0xFC:
		return 'Ь'
	case 0xDD, 0xFD:
		return 'Э'
	case 0xDE, 0xFE:
		return 'Ю'
	case 0xDF, 0xFF:
		return 'Я'

	default:
		return rune(b)
	}
}
