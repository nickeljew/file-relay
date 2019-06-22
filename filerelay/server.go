package filerelay

import (
	"bufio"
	"fmt"
	"math"
	"net"
	"regexp"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"

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
	_
	_
	_
	_
	szGB
)

const (
	SlotMinExpriration = 60
	SlabCheckInterval = 10
)


type MemConfig struct {
	Config `yaml:",inline"`

	LRUSize int `yaml:"lru-size"` //mac count of items in LRU list
	SkipListCheckStep int `yaml:"skiplist-check-step"`
	//SkipListCheckIntv `yaml:"skiplist-check-interval"` int //in seconds

	MinExpiration int64 `yaml:"min-expiration"` //in seconds
	SlabCheckIntv int `yaml:"slab-check-interval"` //in seconds

	SlotCapMin int `yaml:"slot-capacity-min"` //in bytes
	SlotCapMax int  `yaml:"slot-capacity-max"` //in bytes

	SlotsInSlab int `yaml:"slots-in-slab"`
	SlabsInGroup int `yaml:"slabs-in-group"`

	MaxStorage string `yaml:"max-storage"` //example: 200MB, 2GB`

	// the following will not read from configuration data/file
	maxStorageSize int
	totalCapacity int
	sync.Mutex //lock for real-time capacity computing
}

func NewMemConfig() *MemConfig {
	return &MemConfig{
		LRUSize: 100000,
		SkipListCheckStep: 20,
		//SkipListCheckIntv: 60,

		MinExpiration: SlotMinExpriration,
		SlabCheckIntv: SlabCheckInterval,
	
		SlotCapMin: ValFrom(sz16B, sz64B).(int),
		SlotCapMax: ValFrom(szKB, sz4KB).(int),
	
		SlotsInSlab: ValFrom(8, 100).(int),
		SlabsInGroup: ValFrom(1, 100).(int),

		MaxStorage: "200MB",
	}
}

func (c *MemConfig) MaxStorageSize() int {
	defSize := 100 * szMB
	patn := `^([1-9]\d{0,3})([MG]B)$`
	reg := regexp.MustCompile(patn)
	matches := reg.FindAllSubmatch([]byte(c.MaxStorage), -1)
	if len(matches) == 0 {
		return defSize
	}

	parts := matches[0]
	if len(parts) < 3 {
		return defSize
	}

	if num, e := strconv.Atoi(string(parts[1])); e == nil {
		base := string(parts[2])
		if base == "MB" && num > 1 {
			return num * szMB
		} else if base == "GB" && num > 1 && num <= 8 {
			return num * szGB
		}
	}
	return defSize
}


//
type Server struct {
	max int
	handlers []*handler
	waitlist []net.Conn
	hdrNotif chan int
	quit chan byte
	connCount uint64

	memCfg MemConfig
	entry *ItemsEntry
	groups map[int]*SlabGroup

	sync.Mutex
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


func NewServer(c *MemConfig) *Server {
	c.maxStorageSize = c.MaxStorageSize()
	c.totalCapacity = 0
	Debugf("## Start server with config: %v", c)
	return &Server{
		max: c.MaxRoutines,
		handlers: make([]*handler, 0, c.MaxRoutines),
		hdrNotif: make(chan int),
		quit: make(chan byte),
		memCfg: *c,
		entry: NewItemsEntry(c.LRUSize, c.SkipListCheckStep),
		groups: make( map[int]*SlabGroup ),
	}
}

func (s *Server) Start() {
	s.initSlabs()
	s.entry.StartCheck()
	var i int
	for {
		select {
		case i = <- s.hdrNotif:
			if i >= 0 && len(s.waitlist) > 0 {
				h := s.handlers[i]
				s.Lock()
				if !h.running {
					c := s.waitlist[0]
					s.waitlist = s.waitlist[1:]
					Debug("* Pop from waitlist for handler: ", i)
					if e := h.process(makeConn(c), s.entry); e != nil {
						log.Error("failed to process connection: ", e.Error())
					}
				}
				s.Unlock()
			}
		case <- s.quit:
			log.Info("Quit server")
			s.Lock()
			for _, h := range s.handlers {
				h.quit <- 0
			}
			s.Unlock()
			return
		}
	}
}

func (s *Server) Stop() {
	s.entry.StopCheck()
	s.quit <- 0
	s.clearSlabs()
}

//
func (s *Server) initSlabs() {
	c := s.memCfg.SlotCapMin
	for {
		g := NewSlabGroup(c,
			s.memCfg.SlabsInGroup, s.memCfg.SlotsInSlab, s.memCfg.SlabCheckIntv, s.memCfg.maxStorageSize)
		s.memCfg.totalCapacity += g.Capacity()
		s.groups[c] = g
		Debugf("Check group: %d; %d, %v", len(s.groups), c, g)

		c = c << szShift
		if c > s.memCfg.SlotCapMax {
			break
		}
	}
	Debug("Total capacity in initialization: ", s.memCfg.totalCapacity)
}

func (s *Server) clearSlabs() {
	c := s.memCfg.SlotCapMin
	for {
		s.groups[c] = nil

		c = c << szShift
		if c > s.memCfg.SlotCapMax {
			break
		}
	}
}


//
func (s *Server) Handle(nc net.Conn) {
	if err := s.handleConn(nc); err != nil {
		log.Error("error in handling connection: ", err.Error())
	}
	if err := nc.Close(); err != nil {
		log.Error("error in closing connection")
	}
}

//
func (s *Server) handleConn(nc net.Conn) error {
	s.Lock()

	cnt := len(s.handlers)

	if s.connCount == math.MaxUint64 {
		s.connCount = 0
	}
	s.connCount++
	Debug("* Running handlers: ", cnt, s.connCount)

	var hdr *handler

	for i, h := range s.handlers {
		if h != nil && !h.running {
			Debug("** Using handler: ", i)
			hdr = h
			break
		}
	}

	if hdr == nil && cnt < s.max {
		Debug("* Get new handler: ", cnt)
		hdr = newHandler(cnt - 1, s.hdrNotif, &s.memCfg, s.groups)
		s.handlers = append(s.handlers, hdr)
	}

	if hdr == nil {
		Debug("* Put into wait-list")
		s.waitlist = append(s.waitlist, nc)

		s.Unlock()
		return nil
	}

	s.Unlock()
	return hdr.process(makeConn(nc), s.entry)
}



//
type handler struct {
	index int

	quit chan byte
	notif chan int
	running bool

	cfg *MemConfig
	groups map[int]*SlabGroup
}

func newHandler(idx int, notif chan int, c *MemConfig, groups map[int]*SlabGroup) *handler {
	return &handler{
		index: idx,
		quit: make(chan byte),
		notif: notif,
		running: false,
		cfg: c,
		groups: groups,
	}
}

func (h *handler) process(cn *conn, entry *ItemsEntry) error {
	//Debug("Nothing here in handler...")

	h.running = true

	//done and notif
	fulfill := func() {
		h.running = false
		h.notif <- h.index
	}
	defer fulfill()

	line, e := cn.rw.ReadSlice('\n')
	if e != nil {
		return e
	}

	msgline := &MsgLine{}
	msgline.parseLine(line)
	Debugf(" - Recv: %T %v\n - - -", msgline, msgline)

	var storeCmds = map[string]bool{
		"set": true,
		"add": true,
		"replace": true,
	}
	if storeCmds[msgline.Cmd] {
		return h.handleStorage(msgline, cn.rw, entry)
	} else if msgline.Cmd == "get" || msgline.Cmd == "gets" {
		return h.handleRetrieval(msgline, cn.rw, entry)
	}

	log.WithFields(logrus.Fields{
		"cmd": msgline.Cmd,
		"itemKey": msgline.Key,
	}).Warn("Unsupported command")
	return nil
}



func (h *handler) handleStorage(msgline *MsgLine, rw *bufio.ReadWriter, entry *ItemsEntry) error {
	_log := log.WithFields(logrus.Fields{
		"cmd": msgline.Cmd,
		"itemKey": msgline.Key,
		"valueLen": msgline.ValueLen,
	})

	exp := msgline.Expiration
	if exp < h.cfg.MinExpiration {
		exp = h.cfg.MinExpiration
	}

	makeResp := func(cmd []byte) {
		if _, e := rw.Write(cmd); e != nil {
			_log.Error(e.Error())
		}
		if e := rw.Flush(); e != nil {
			_log.Error(e.Error())
		}
	}
	failResp := func() {
		makeResp(ResultNotStored)
	}

	item := NewMetaItem(msgline.Key, msgline.Flags, exp, msgline.ValueLen)
	var err error
	switch msgline.Cmd {
	case "set":
		err = entry.Set(item)
	case "add":
		err = entry.Add(item)
	case "replace":
		err = entry.Replace(item)
	}
	if err != nil {
		failResp()
		return err
	}

	if e := h.allocSlots(item); e != nil {
		_log.Error(e.Error())
		_ = entry.Remove(item.key)

		failResp()
		return e
	}

	bytesLeft := msgline.ValueLen
	for i, s := range item.slots {
		Debugf(" - Slot[%d]: %d, byte-left: %d", s.capacity, i, bytesLeft)

		s.SetInfoWithItem(item)
		if n, e := s.ReadAndSet(msgline.Key, rw, bytesLeft); e != nil {
			_log.Error(e.Error())
			Debug(" - Read data failure: ", e.Error())

			failResp()
			return e
		} else {
			bytesLeft -= n
		}
	}

	makeResp(ResultStored)
	return nil
}


func (h *handler) allocSlots(t *MetaItem) error {
	byteLen := t.byteLen
	c := h.cfg.SlotCapMax
	for {
		Debugf("Check slab-groups for capacity: %d; Left-bytes: %d", c, byteLen)

		if byteLen >= c || c == h.cfg.SlotCapMin {
			rest := byteLen % c
			cnt := 0
			if byteLen > rest {
				cnt = (byteLen - rest) / c
				byteLen = rest
			}

			// When there are some bytes left and no smaller capacity,
			// then add one surplus slot to contain the left bytes
			if byteLen > 0 && c == h.cfg.SlotCapMin {
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
		c = c >> szShift
	}
	return nil
}


func (h *handler) findSlots(slotCap, cnt int, t *MetaItem) error {
	group := h.groups[slotCap]
	if slots, e := group.FindAvailableSlots(cnt, h.cfg.totalCapacity); e == nil {
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


func (h *handler) handleRetrieval(msgline *MsgLine, rw *bufio.ReadWriter, entry *ItemsEntry) error {
	_log := log.WithFields(logrus.Fields{
		"cmd": msgline.Cmd,
		"itemKey": msgline.Key,
	})

	endResp := func() {
		if _, e := rw.Write(ResultEnd); e != nil {
			_log.Error(e.Error())
		}
		if e := rw.Flush(); e != nil {
			_log.Error(e.Error())
		}
	}

	item := entry.Get(msgline.Key)
	if item == nil {
		item = NewMetaItem(msgline.Key, 0, 0, 0)
	}

	// Write value block and end with \r\n
	if item.byteLen > 0 && len(item.slots) > 0 {
		byteLen := item.byteLen
		for _, s := range item.slots {
			if s.Vacant() {
				byteLen = 0
				break
			}
		}

		if e := h.writeRespFirstLine(item, rw, byteLen); e != nil {
			endResp()
			return e
		}

		if byteLen > 0 {
			for _, s := range item.slots {
				bytes := s.data[:s.used]
				if _, e := rw.Write(bytes); e != nil {
					endResp()
					return e
				}
			}
			if _, e := rw.Write([]byte("\r\n")); e != nil {
				endResp()
				return e
			}
		}
	}

	endResp()
	return nil
}

func (h *handler) writeRespFirstLine(item *MetaItem, rw *bufio.ReadWriter, byteLen int) error {
	line := fmt.Sprintf("VALUE %s %d %d\r\n", item.key, item.flags, byteLen)
	if _, err := rw.Write([]byte(line)); err != nil {
		return err
	}
	return nil
}

