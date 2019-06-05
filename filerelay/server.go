package filerelay

import (
	"fmt"
	"net"
	"bufio"
	"sync"
	"time"
	//"errors"

	"github.com/huandu/skiplist"
	. "github.com/nickeljew/file-relay/debug"
)

const (
	szShift = 2
)
const (
	//for test
	sz4B int = 4 << (szShift * iota)
	sz16B
	sz64B
	sz256B
	szKB
	sz4KB
	sz16KB
	sz64KB
	sz256KB
	szMB
)

const (
	SlotMinExpriration = 60
	StubCheckInterval = 5
)


type MemConfig struct {
	MinExpiration int64 //in seconds
	StubCheckIntv int //in seconds

	SlotCapMin int //in bytes
	SlotCapMax int //in bytes

	SlotsInStub int
	SlubsInGroup int
}

func NewMemConfig() MemConfig {
	return MemConfig{
		MinExpiration: SlotMinExpriration,
		StubCheckIntv: StubCheckInterval,
	
		SlotCapMin: sz64B,//szKB,
		SlotCapMax: szKB,//szMB,
	
		SlotsInStub: 10,
		SlubsInGroup: 10,
	}
}


//
type Server struct {
	max int
	handlers []*handler
	waitlist []net.Conn
	hdrNotif chan int
	quit chan int
	memCfg MemConfig

	entry *itemsEntry
	groups map[int]*StubGroup
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


func NewServer(max int, c MemConfig) *Server {
	return &Server{
		max: max,
		handlers: make([]*handler, 0, max),
		hdrNotif: make(chan int),
		quit: make(chan int),
		memCfg: c,
		entry: newItemsEntry(),
		groups: make( map[int]*StubGroup ),
	}
}

func (s *Server) Start() {
	s.initStubs()
	var i int
	for {
		select {
		case i = <- s.hdrNotif:
			if i >= 0 && len(s.waitlist) > 0 {
				h := s.handlers[i]
				if !h.running {
					c := s.waitlist[0]
					s.waitlist = s.waitlist[1:]
					Debug("* Pop from waitlist for handler: ", i)
					h.process(makeConn(c))
				}
			}
		case <- s.quit:
			log.Info("Quit server")
			return
		}
	}
}

func (s *Server) Stop() {
	s.quit <- 0
	s.clearStubs()
}

//
func (s *Server) initStubs() {
	c := s.memCfg.SlotCapMin
	for {
		s.groups[c] = NewStubGroup(c,
			s.memCfg.SlubsInGroup, s.memCfg.SlotsInStub, s.memCfg.StubCheckIntv)

		Debugf("Check group: %d; %d, %v", len(s.groups), c, s.groups[c])

		c = c << szShift
		if (c > s.memCfg.SlotCapMax) {
			break
		}
	}
}

func (s *Server) clearStubs() {
	c := s.memCfg.SlotCapMin
	for {
		s.groups[c] = nil

		c = c << szShift
		if (c > s.memCfg.SlotCapMax) {
			break
		}
	}
}


//
func (s *Server) Handle(nc net.Conn) {
	if err := s.handleConn(nc); err != nil {
		log.Error("* Error in handling connection: ", err.Error())
	}
	nc.Close()
}

//
func (s *Server) handleConn(nc net.Conn) error {
	cnt := len(s.handlers)
	Debug("* Running handlers: ", cnt)

	var hdr *handler

	for i, h := range s.handlers {
		if (h != nil && !h.running) {
			Debug("** Using handler: ", i)
			hdr = h
			break
		}
	}

	if hdr == nil {
		Debug("* Get new handler or wait in line")
		if cnt < s.max {
			hdr = newHandler(cnt - 1, s.hdrNotif, &s.memCfg, s.groups)
			s.handlers = append(s.handlers, hdr)
		}
	}

	if hdr != nil {
		return hdr.process(makeConn(nc))
	}

	s.waitlist = append(s.waitlist, nc)
	return nil
}


type itemsEntry struct {
	list *skiplist.SkipList
	sync.Mutex
}

func newItemsEntry() *itemsEntry {
	return &itemsEntry{
		list: skiplist.New(skiplist.String),
	}
}


type MetaItem struct {
	key string
	flags uint32
	setAt time.Time
	duration time.Duration
	byteLen int
	slots []*Slot
}

func newMetaItem(key string, flags uint32, expiration int64, byteLen int) (t *MetaItem) {
	t = &MetaItem{
		key: key,
		flags: flags,
		setAt: time.Now(),
		byteLen: byteLen,
		//slots: make([]*Slot),
	}
	secs := fmt.Sprintf("%ds", expiration)
	t.duration, _ = time.ParseDuration(secs)
	return
}




//
type handler struct {
	index int
	//cn *conn
	quit chan int
	notif chan int
	running bool

	cfg *MemConfig
	groups map[int]*StubGroup
}

func newHandler(idx int, notif chan int, c *MemConfig, groups map[int]*StubGroup) (*handler) {
	return &handler{
		index: idx,
		quit: make(chan int),
		notif: notif,
		running: false,
		cfg: c,
		groups: groups,
	}
}

func (h *handler) process(cn *conn) error {
	//Debug("Nothing here in handler...")

	h.running = true
	//h.cn = c

	line, err := cn.rw.ReadSlice('\n')
	if err != nil {
		return err
	}

	reqline := &ReqLine{}
	reqline.parseLine(line)
	Debugf(" - Recv: %T %v\n -\n", reqline, reqline)

	exp := reqline.Expiration
	if exp < h.cfg.MinExpiration {
		exp = h.cfg.MinExpiration
	}
	item := newMetaItem(reqline.Key, reqline.Flags, exp, reqline.ValueLen)
	h.allocSlots(item)

	var i, c int
	for _, s := range item.slots {
		if c != s.capacity {
			c = s.capacity
			i = 1
		}
		Debugf("- Slot[%d]: %d", s.capacity, i)
		i++
	}

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

func (h *handler) allocSlots(t *MetaItem) error {
	byteLen := t.byteLen
	c := h.cfg.SlotCapMax
	for {
		//Debugf("Check stub-groups for capacity: %d; Left-bytes: %d", c, byteLen)

		if byteLen > c {
			rest := byteLen % c
			cnt := (byteLen - rest) / c
			byteLen = rest
			if c == h.cfg.SlotCapMin {
				byteLen = 0
				cnt++
			}
			if e := h.findSlots(c, cnt, t); e != nil {
				return e
			}
		}

		if byteLen <= 0 {
			break
		}
		c = c >> 2
	}
	return nil
}


func (h *handler) findSlots(slotCap, cnt int, t *MetaItem) error {
	group := h.groups[slotCap]
	if slots, e := group.FindAvailableSlots(cnt); e == nil {
		if len(t.slots) > 0 {
			t.slots = append(t.slots, slots...)
		} else {
			t.slots = slots
		}
	} else {
		return e
	}
	return nil
}