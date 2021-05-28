// +build !windows

package serial

import (
	"io"
	"syscall"
	"time"
	"unsafe"
)

type Port struct {
	BasePort
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

func (p *Port) Read(buf []byte) (n int, err error) {
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

func (p *Port) Write(buf []byte) (n int, err error) {
	n, err = p.f.Write(buf)
	if err != nil {
		p.logMsg("Write", err.Error())
	} else if n > 0 {
		p.logData('-', buf)
	}
	return
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
