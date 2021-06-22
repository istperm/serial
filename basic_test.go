package serial

import (
	"testing"
)

func TestConnection(t *testing.T) {
	// cfg := Config{Name: "/dev/ttyACM0", Baud: 115200}
	// ser, err := OpenPort(&cfg)
	//  if err != nil {
	//  	t.Fatal(err)
	// }
	// ser.SetDtr(false)
	// ser.Flush()
	// time.Sleep(100*time.Millisecond)
	// ser.SetDtr(true)

	// buf := make([]byte, 64)
	// n, err := ser.Read(buf)

	// c1 := Config{Name: "/dev/ttyUSB1", Baud: 115200}
	// s2, err := OpenPort(&c1)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// ch := make(chan int, 1)
	// go func() {
	// 	buf := make([]byte, 128)
	// 	var readCount int
	// 	for {
	// 		n, err := s2.Read(buf)
	// 		if err != nil {
	// 			return
	// 			//t.Fatal(err)
	// 		}
	// 		readCount++
	// 		t.Logf("Read %v %v bytes: % 02x %s", readCount, n, buf[:n], buf[:n])
	// 		select {
	// 		case <-ch:
	// 			ch <- readCount
	// 			close(ch)
	// 		default:
	// 		}
	// 	}
	// }()

	// if _, err = s1.Write([]byte("hello")); err != nil {
	// 	t.Fatal(err)
	// }
	// if _, err = s1.Write([]byte(" ")); err != nil {
	// 	t.Fatal(err)
	// }
	// time.Sleep(time.Second)
	// if _, err = s1.Write([]byte("world")); err != nil {
	// 	t.Fatal(err)
	// }
	// time.Sleep(time.Second / 10)

	// ch <- 0
	// s1.Write([]byte(" ")) // We could be blocked in the read without this
	// c := <-ch
	// exp := 5
	// if c >= exp {
	// 	t.Fatalf("Expected less than %v read, got %v", exp, c)
	// }
}
