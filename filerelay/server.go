package filerelay

import (
	"fmt"
	"net"
)

//
type Server struct {
	max int
	handlers []*handler
	waitlist []*net.Conn
	hdrNotif chan int
	quit chan int
}

func NewServer(max int) *Server {
	return &Server{
		max: max,
		hdrNotif: make(chan int),
	}
}

func (s *Server) start() {
	var i int
	for {
		select {
		case i = <- s.hdrNotif:
			if i >= 0 && len(s.waitlist) > 0 {
				h := s.handlers[i]
				if h.running == true {
					c := s.waitlist[0]
					s.waitlist = s.waitlist[1:]
					h.process(c)
				}
			}
		case <-s.quit:
			fmt.Println("Quit server")
			return
		}
	}
}

func (s *Server) stop() {
	s.quit <- 0
}

//
func (s *Server) handleConn(conn *net.Conn) error {
	cnt := len(s.handlers)
	fmt.Println("Running handlers: ", cnt)

	var hdr *handler

	for i, h := range s.handlers {
		if (h != nil) {
			fmt.Println("Handle running: ", i, h.running)
			hdr = h
		}
	}

	if hdr == nil {
		if cnt < s.max {
			hdr = newHandler(cnt - 1, s.hdrNotif)
			s.handlers = append(s.handlers, hdr)
		}
		fmt.Println("No running hanlder: ")
	}

	if hdr != nil {
		return hdr.process(nil)
	}

	s.waitlist = append(s.waitlist, conn)
	return nil
}

//
type handler struct {
	index int
	conn *net.Conn
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

func (h *handler) process(c *net.Conn) error {
	fmt.Println("Nothing here in handler...")

	h.running = true
	h.conn = c

	//done and notif
	h.running = false
	h.notif <- h.index

	return nil
}
