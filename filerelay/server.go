package filerelay

import (
	"fmt"
	"net"
	"bufio"
)

//
type Server struct {
	max int
	handlers []*handler
	waitlist []net.Conn
	hdrNotif chan int
	quit chan int
}

//
type conn struct {
	nc net.Conn
	rw *bufio.ReadWriter
}

func makeConn(nc net.Conn) *conn {
	return &conn{
		nc: nc,
		rw: bufio.NewReadWriter(bufio.NewReader(nc), bufio.NewWriter(nc)),
	}
}


func NewServer(max int) *Server {
	return &Server{
		max: max,
		hdrNotif: make(chan int),
	}
}

func (s *Server) Start() {
	var i int
	for {
		select {
		case i = <- s.hdrNotif:
			if i >= 0 && len(s.waitlist) > 0 {
				h := s.handlers[i]
				if !h.running {
					c := s.waitlist[0]
					s.waitlist = s.waitlist[1:]
					fmt.Println("* Pop from waitlist for handler: ", i)
					h.process(makeConn(c))
				}
			}
		case <-s.quit:
			fmt.Println("Quit server")
			return
		}
	}
}

func (s *Server) Stop() {
	s.quit <- 0
}


//
func (s *Server) Handle(nc net.Conn) {
	if err := s.handleConn(nc); err != nil {
		fmt.Println("* Error in handling connection: ", err.Error())
	}
	nc.Close()
}

//
func (s *Server) handleConn(nc net.Conn) error {
	cnt := len(s.handlers)
	fmt.Println("* Running handlers: ", cnt)

	var hdr *handler

	for i, h := range s.handlers {
		if (h != nil && !h.running) {
			fmt.Println("** Using handler: ", i)
			hdr = h
			break
		}
	}

	if hdr == nil {
		fmt.Println("* Get new handler or wait in line")
		if cnt < s.max {
			hdr = newHandler(cnt - 1, s.hdrNotif)
			s.handlers = append(s.handlers, hdr)
		}
	}

	if hdr != nil {
		return hdr.process(makeConn(nc))
	}

	s.waitlist = append(s.waitlist, nc)
	return nil
}

//
type handler struct {
	index int
	//cn *conn
	quit chan int
	notif chan int
	running bool
}

func newHandler(idx int, notif chan int) (*handler) {
	return &handler{
		index: idx,
		quit: make(chan int),
		notif: notif,
		running: false,
	}
}

func (h *handler) process(cn *conn) error {
	//fmt.Println("Nothing here in handler...")

	h.running = true
	//h.cn = c

	line, err := cn.rw.ReadSlice('\n')
	if err != nil {
		return err
	}

	reqline := &ReqLine{}
	reqline.parseLine(line)
	fmt.Printf(" - Recv: %T %v\n -\n", reqline, reqline)

	// cnt := 0
	// for {
	// 	data, err := cn.rw.ReadSlice('\n')
	// 	if err != nil {
	// 		return err
	// 	}
	// 	cnt += len(data)
	// 	if (cnt < reqline.ValueLen) {
	// 		//
	// 	}
	// }

	if _, err = cn.rw.Write(ResultStored); err != nil {
		return err
	}
	if err := cn.rw.Flush(); err != nil {
		return err
	}

	//done and notif
	h.running = false
	h.notif <- h.index

	return nil
}
