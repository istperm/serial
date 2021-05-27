// +build !windows,!linux,cgo
// -build !windows,cgo

package serial

import "C"

// TODO: Maybe change to using syscall package + ioctl instead of cgo

import (
	"io"
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"
)

type Port struct {
	f      *os.File
	logger *log.Logger
	logTag rune
	logBuf [64]byte
	logPtr int
}

func openPort(name string, baud int, readTimeout time.Duration) (p *Port, err error) {
	f, err := os.OpenFile(name, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
	if err != nil {
		return
	}

	fd := C.int(f.Fd())
	if C.isatty(fd) != 1 {
		f.Close()
		return nil, SerialError{Msg: "File is not a tty"}
	}

	var st C.struct_termios
	_, err = C.tcgetattr(fd, &st)
	if err != nil {
		f.Close()
		return
	}
	var speed C.speed_t
	switch baud {
	case 115200:
		speed = C.B115200
	case 57600:
		speed = C.B57600
	case 38400:
		speed = C.B38400
	case 19200:
		speed = C.B19200
	case 9600:
		speed = C.B9600
	case 4800:
		speed = C.B4800
	case 2400:
		speed = C.B2400
	default:
		f.Close()
		return nil, SerialError{Msg: "Invalid baud rate", Cod: baud}
	}

	_, err = C.cfsetispeed(&st, speed)
	if err != nil {
		f.Close()
		return
	}
	_, err = C.cfsetospeed(&st, speed)
	if err != nil {
		f.Close()
		return
	}

	// Turn off break interrupts, CR->NL, Parity checks, strip, and IXON
	st.c_iflag &= ^C.tcflag_t(C.BRKINT | C.ICRNL | C.INPCK | C.ISTRIP | C.IXOFF | C.IXON | C.PARMRK)

	// Select local mode, turn off parity, set to 8 bits
	CRTSCTS := 020000000000
	st.c_cflag &= ^C.tcflag_t(C.PARENB | C.CSIZE | CRTSCTS)
	st.c_cflag |= (C.CREAD | C.CLOCAL | syscall.CSTOPB | C.CS8)

	// Select raw mode
	st.c_lflag &= ^C.tcflag_t(C.ICANON | C.ECHO | C.ECHOE | syscall.ECHONL | C.ISIG)
	st.c_oflag &= ^C.tcflag_t(C.OPOST | syscall.ONLCR)

	// set blocking / non-blocking read
	// http://man7.org/linux/man-pages/man3/termios.3.html
	// Supports blocking read and read with timeout operations
	vmin, vtime := posixTimeoutValues(readTimeout)
	st.c_cc[C.VMIN] = C.cc_t(vmin)
	st.c_cc[C.VTIME] = C.cc_t(vtime)

	_, err = C.tcsetattr(fd, C.TCSANOW, &st)
	if err != nil {
		f.Close()
		return nil, err
	}

	r1, _, e := syscall.Syscall(syscall.SYS_FCNTL,
		f.Fd(),
		uintptr(syscall.F_SETFL),
		uintptr(0),
	)
	if e != 0 || r1 != 0 {
		f.Close()
		return nil, SerialError{Tag: "Clear NONBLOCK", Msg: e.Error(), Cod: int(r1)}
	}

	return &Port{f: f}, nil
}

func (p *Port) Close() (err error) {
	p.logMsg("Close", "")
	return p.f.Close()
}

func (p *Port) Read(buf []byte) (n int, err error) {
	n, err = p.f.Read(buf)
	if err != nil && err != io.EOF {
		p.logMsg("Read", "Error %d", err)
		return n, err
	} else if n > 0 {
		p.logData('+', buf)
		return n, nil
	}
	return 0, nil
}

func (p *Port) Write(buf []byte) (n int, err error) {
	n, err = p.f.Write(buf)
	if err != nil {
		p.logMsg("Write", err.Error())
	} else if n > 0 {
		p.logData('-', buf)
	}
	return n, err
}

// Discards data written to the port but not transmitted,
// or data received but not read
func (p *Port) Flush() error {
	_, err := C.tcflush(C.int(p.f.Fd()), C.TCIOFLUSH)
	if err != nil {
		p.logMsg("Flush", "Error %d", err)
		return err
	}
	return nil
}

func (p *Port) SetDtr(v bool) error {
	return p.setModemLine("DTR", syscall.TIOCM_DTR, v)
}

func (p *Port) SetRts(v bool) error {
	return p.setModemLine("RTS", syscall.TIOCM_RTS, v)
}

func (p *Port) setModemLine(tag string, line uint, v bool) error {
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
