package main

import (
	"os"
	"syscall"
	"unsafe"
)

type syscallTermios struct {
	Iflag  uint64
	Oflag  uint64
	Cflag  uint64
	Lflag  uint64
	Cc     [20]uint8
	Ispeed uint64
	Ospeed uint64
}

func enableRaw(orig *syscallTermios) error {
	fd := int(os.Stdin.Fd())
	var term syscall.Termios
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&term)), 0, 0, 0); err != 0 {
		return err
	}
	*orig = syscallTermios{Iflag: uint64(term.Iflag), Oflag: uint64(term.Oflag), Cflag: uint64(term.Cflag), Lflag: uint64(term.Lflag)}
	raw := term
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Cflag |= syscall.CS8
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&raw)), 0, 0, 0); err != 0 {
		return err
	}
	return nil
}

func disableRaw(orig *syscallTermios) error {
	fd := int(os.Stdin.Fd())
	var term syscall.Termios
	term.Iflag = uint32(orig.Iflag)
	term.Oflag = uint32(orig.Oflag)
	term.Cflag = uint32(orig.Cflag)
	term.Lflag = uint32(orig.Lflag)
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&term)), 0, 0, 0); err != 0 {
		return err
	}
	return nil
}

type win struct{ Row, Col, Xpixel, Ypixel uint16 }

func winsize() (int, int, error) {
	ws := &win{}
	fd := int(os.Stdout.Fd())
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(ws))); err != 0 {
		return 0, 0, err
	}
	return int(ws.Row), int(ws.Col), nil
}

func readKey() (int, error) {
	var b [1]byte
	n, err := os.Stdin.Read(b[:])
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, nil
	}
	ch := b[0]
	if ch == 13 {
		return KeyEnter, nil
	}
	if ch == 127 {
		return KeyBackspace, nil
	}
	if ch == 27 {
		var seq [3]byte
		n, _ := os.Stdin.Read(seq[:1])
		if n == 0 {
			return KeyEsc, nil
		}
		if seq[0] == '[' {
			n2, _ := os.Stdin.Read(seq[1:2])
			if n2 == 0 {
				return KeyEsc, nil
			}
			c := seq[1]
			if c >= '0' && c <= '9' {
				n3, _ := os.Stdin.Read(seq[2:3])
				if n3 == 0 {
					return KeyEsc, nil
				}
				if seq[2] == '~' {
					switch c {
					case '1':
						return KeyHome, nil
					case '3':
						return KeyDel, nil
					case '4':
						return KeyEnd, nil
					case '5':
						return KeyPageUp, nil
					case '6':
						return KeyPageDown, nil
					case '7':
						return KeyHome, nil
					case '8':
						return KeyEnd, nil
					}
				}
			} else {
				switch c {
				case 'A':
					return KeyArrowUp, nil
				case 'B':
					return KeyArrowDown, nil
				case 'C':
					return KeyArrowRight, nil
				case 'D':
					return KeyArrowLeft, nil
				case 'H':
					return KeyHome, nil
				case 'F':
					return KeyEnd, nil
				}
			}
		} else if seq[0] == 'O' {
			var s [1]byte
			n2, _ := os.Stdin.Read(s[:])
			if n2 == 0 {
				return KeyEsc, nil
			}
			switch s[0] {
			case 'H':
				return KeyHome, nil
			case 'F':
				return KeyEnd, nil
			}
		}
		return KeyEsc, nil
	}
	if ch <= 26 {
		if ch == 17 {
			return KeyQuit, nil
		}
		if ch == 19 {
			return KeySave, nil
		}
		return int(ch), nil
	}
	if ch >= 32 && ch <= 126 {
		return int(ch), nil
	}
	return int(ch), nil
}
