// +build !windows,!linux,cgo

package serial

// #include <termios.h>
// #include <unistd.h>
import "C"

// TODO: Maybe change to using syscall package + ioctl instead of cgo

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
	"time"
	//"unsafe"
)

type Port struct {
	// We intentionly do not use an "embedded" struct so that we
	// don't export File
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
		return nil, errors.New("File is not a tty")
	}

	var st C.struct_termios
	_, err = C.tcgetattr(fd, &st)
	if err != nil {
		f.Close()
		return nil, err
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
		return nil, fmt.Errorf("Unknown baud rate %v", baud)
	}

	_, err = C.cfsetispeed(&st, speed)
	if err != nil {
		f.Close()
		return nil, err
	}
	_, err = C.cfsetospeed(&st, speed)
	if err != nil {
		f.Close()
		return nil, err
	}

	// Turn off break interrupts, CR->NL, Parity checks, strip, and IXON
	st.c_iflag &= ^C.tcflag_t(C.BRKINT | C.ICRNL | C.INPCK | C.ISTRIP | C.IXOFF | C.IXON | C.PARMRK)

	// Select local mode, turn off parity, set to 8 bits
	st.c_cflag &= ^C.tcflag_t(C.CSIZE | C.PARENB)
	st.c_cflag |= (C.CLOCAL | C.CREAD | C.CS8)

	// Select raw mode
	st.c_lflag &= ^C.tcflag_t(C.ICANON | C.ECHO | C.ECHOE | C.ISIG)
	st.c_oflag &= ^C.tcflag_t(C.OPOST)

	// set blocking / non-blocking read
	/*
	*	http://man7.org/linux/man-pages/man3/termios.3.html
	* - Supports blocking read and read with timeout operations
	 */
	vmin, vtime := posixTimeoutValues(readTimeout)
	st.c_cc[C.VMIN] = C.cc_t(vmin)
	st.c_cc[C.VTIME] = C.cc_t(vtime)

	_, err = C.tcsetattr(fd, C.TCSANOW, &st)
	if err != nil {
		f.Close()
		return nil, err
	}

	//fmt.Println("Tweaking", name)
	r1, _, e := syscall.Syscall(syscall.SYS_FCNTL,
		uintptr(f.Fd()),
		uintptr(syscall.F_SETFL),
		uintptr(0))
	if e != 0 || r1 != 0 {
		s := fmt.Sprint("Clearing NONBLOCK syscall error:", e, r1)
		f.Close()
		return nil, errors.New(s)
	}

	/*
				r1, _, e = syscall.Syscall(syscall.SYS_IOCTL,
			                uintptr(f.Fd()),
			                uintptr(0x80045402), // IOSSIOSPEED
			                uintptr(unsafe.Pointer(&baud)));
			        if e != 0 || r1 != 0 {
			                s := fmt.Sprint("Baudrate syscall error:", e, r1)
					f.Close()
		                        return nil, os.NewError(s)
				}
	*/

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
	a2 := syscall.TIOCMBIC
	if v {
		a2 = syscall.TIOCMBIS
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(p.f.Fd()),
		uintptr(a2),
		uintptr(syscall.TIOCM_DTR),
	)
	if errno != 0 {
		p.logMsg("SetDtr", "(%t) -> error %d", v, errno)
		return errno
	} else {
		p.logMsg("DTR", "%t", v)
		return nil
	}
}

func (p *Port) SetRts(v bool) error {
	a2 := syscall.TIOCMBIC
	if v {
		a2 = syscall.TIOCMBIS
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(p.f.Fd()),
		uintptr(a2),
		uintptr(syscall.TIOCM_RTS),
	)
	if errno != 0 {
		p.logMsg("SetDtr", "(%t) -> error %d", v, errno)
		return errno
	} else {
		p.logMsg("DTR", "%t", v)
		return nil
	}
}
