// +build linux
// -build !windows,!cgo

package serial

import (
	"os"
	"syscall"
	"unsafe"
)

type Port struct {
	BasePort
}

func openPort(c *Config) (p *Port, err error) {
	var bauds = map[int]uint32{
		50:      syscall.B50,
		75:      syscall.B75,
		110:     syscall.B110,
		134:     syscall.B134,
		150:     syscall.B150,
		200:     syscall.B200,
		300:     syscall.B300,
		600:     syscall.B600,
		1200:    syscall.B1200,
		1800:    syscall.B1800,
		2400:    syscall.B2400,
		4800:    syscall.B4800,
		9600:    syscall.B9600,
		19200:   syscall.B19200,
		38400:   syscall.B38400,
		57600:   syscall.B57600,
		115200:  syscall.B115200,
		230400:  syscall.B230400,
		460800:  syscall.B460800,
		500000:  syscall.B500000,
		576000:  syscall.B576000,
		921600:  syscall.B921600,
		1000000: syscall.B1000000,
		1152000: syscall.B1152000,
		1500000: syscall.B1500000,
		2000000: syscall.B2000000,
		2500000: syscall.B2500000,
		3000000: syscall.B3000000,
		3500000: syscall.B3500000,
		4000000: syscall.B4000000,
	}
	rate := bauds[c.Baud]
	if rate == 0 {
		return nil, SerialError{Msg: "Invalid baud rate", Cod: c.Baud}
	}

	//	f, err := os.OpenFile(c.Name, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
	f, err := os.OpenFile(c.Name, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil && f != nil {
			f.Close()
		}
	}()

	fd := f.Fd()

	// Get current port settings
	var ps syscall.Termios
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TCGETS),
		uintptr(unsafe.Pointer(&ps)),
	)
	if errno != 0 {
		return nil, errno
	}

	// #define CRTSCTS 020000000000 /* Flow control. */
	CRTSCTS := 020000000000
	ps.Cflag &= ^uint32(syscall.PARENB | syscall.CSIZE | syscall.CSTOPB | CRTSCTS)
	ps.Cflag |= (syscall.CREAD | syscall.CLOCAL | syscall.CS8)
	if c.StopBits > 1 {
		ps.Cflag |= syscall.CSTOPB
	}

	ps.Lflag &= ^uint32(syscall.ICANON | syscall.ECHO | syscall.ECHOE | syscall.ECHONL | syscall.ISIG)

	ps.Iflag &= ^uint32(syscall.IXON | syscall.IXOFF | syscall.IXANY)
	ps.Iflag &= ^uint32(syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL)
	ps.Iflag |= syscall.IGNPAR

	ps.Oflag &= ^uint32(syscall.OPOST | syscall.ONLCR)

	vmin, vtime := posixTimeoutValues(c.ReadTimeout)
	ps.Cc[syscall.VMIN] = vmin
	ps.Cc[syscall.VTIME] = vtime

	ps.Ispeed = rate
	ps.Ospeed = rate

	_, _, errno = syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TCSETS),
		uintptr(unsafe.Pointer(&ps)),
	)
	if errno != 0 {
		return nil, errno
	}

	if err = syscall.SetNonblock(int(fd), true); err != nil {
		return
	}

	return &Port{BasePort{f: f}}, nil
}

// Discards data written to the port but not transmitted,
// or data received but not read
func (p *Port) Flush() error {
	const TCFLSH = 0x540B
	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(p.f.Fd()),
		uintptr(TCFLSH),
		uintptr(syscall.TCIOFLUSH),
	)
	return err
}
