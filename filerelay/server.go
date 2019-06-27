package filerelay

import (
	"bufio"
	linkedlist "container/list"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	. "github.com/nickeljew/file-relay/debug"
)

const (
	szShift = 2
)
const (
	sz4B uint64 = 4 << (szShift * iota)
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

type HdrState byte
const (
	HdrReady HdrState = iota + 0
	HdrRunning
	HdrIdle
	HdrQuit
)

const (
	SlotMinExpriration = 60
	SlabCheckInterval = 10
)

var _StoreCmds = map[string]bool{
	"set": true,
	"add": true,
	"replace": true,
}


type MemConfig struct {
	Config `yaml:",inline"`

	LRUSize int `yaml:"lru-size"` //mac count of items in LRU-list
	SkipListCheckStep int `yaml:"skiplist-check-step"`
	//SkipListCheckIntv `yaml:"skiplist-check-interval"` int //in seconds

	MinExpiration int64 `yaml:"min-expiration"` //in seconds
	SlabCheckIntv int `yaml:"slab-check-interval"` //in seconds

	SlotCapMin uint64 `yaml:"slot-capacity-min"` //in bytes
	SlotCapMax uint64  `yaml:"slot-capacity-max"` //in bytes

	SlotsInSlab int `yaml:"slots-in-slab"`
	SlabsInGroup int `yaml:"slabs-in-group"`

	MaxStorage string `yaml:"max-storage"` //example: 200MB, 2GB`

	// the following will not read from configuration data/file
	maxStorageSize uint64
	totalCapacity uint64
	sync.Mutex //lock for real-time capacity computing
}

func NewMemConfig() *MemConfig {
	return &MemConfig{
		LRUSize: 100000,
		SkipListCheckStep: 20,
		//SkipListCheckIntv: 60,

		MinExpiration: SlotMinExpriration,
		SlabCheckIntv: SlabCheckInterval,
	
		SlotCapMin: ValFrom(sz16B, sz64B).(uint64),
		SlotCapMax: ValFrom(szKB, szMB).(uint64),
	
		SlotsInSlab: ValFrom(10, 100).(int),
		SlabsInGroup: ValFrom(20, 100).(int),

		MaxStorage: "200MB",
	}
}

func (c *MemConfig) MaxStorageSize() uint64 {
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

	if num, e := strconv.ParseUint(string(parts[1]), 10, 64); e == nil {
		base := string(parts[2])
		if base == "MB" && num > 1 {
			return num * szMB
		} else if base == "GB" && num > 1 && num <= 8 {
			return num * szGB
		}
	}
	return defSize
}

func (c *MemConfig) AddCapToTotal(cap uint64) {
	c.Lock()
	c.totalCapacity += cap
	c.Unlock()
}

func (c *MemConfig) TotalCapacity() uint64 {
	c.Lock()
	total := c.totalCapacity
	c.Unlock()
	return total
}




//
type ServConn struct {
	nc net.Conn
	rw *bufio.ReadWriter
	index uint64

	timer *time.Timer
	closed bool
}

func MakeServConn(nc net.Conn, index uint64) *ServConn {
	return &ServConn{
		nc: nc,
		rw: bufio.NewReadWriter(bufio.NewReader(nc), bufio.NewWriter(nc)),
		index: index,
		closed: false,
	}
}

func (sc *ServConn) Close() {
	if !sc.closed {
		sc.closed = true

		timeout := false
		if sc.timer != nil {
			sc.timer.Stop()
			sc.timer = nil
			timeout = true
		}

		if e := sc.nc.Close(); e != nil {
			logger.Errorf("error in closing connection at index [%d]", sc.index)
		}

		if timeout {
			logger.Warnf("connection timed out at index [%d]", sc.index)
		} else {
			logger.Infof("connection closed at index [%d]", sc.index)
		}
	}
}

func (sc *ServConn) AutoTimeOut() {
	if sc.timer != nil {
		return
	}
	sc.timer = time.NewTimer(time.Second * 10)
	go func() {
		<- sc.timer.C
		sc.timer = nil
		sc.Close()
	}()
}



//
type WaitQueue struct {
	size int
	queue *linkedlist.List
	sync.Mutex
}

func NewWaitQueue(size int) *WaitQueue {
	return &WaitQueue{
		size: size,
		queue: linkedlist.New(),
	}
}

func (w *WaitQueue) Purge() {
	w.queue.Init()
}

func (w *WaitQueue) Push(sc *ServConn) {
	w.Lock()
	if l := w.queue.Len(); l >= w.size {
		// drop connection when reaching the limit
		sc.Close()
	} else {
		w.queue.PushFront(sc)
		sc.AutoTimeOut()
	}
	w.Unlock()
}

func (w *WaitQueue) Pop() (sc *ServConn) {
	w.Lock()
	elem := w.queue.Back()
	sc = nil
	if elem != nil {
		sc = w.queue.Remove(elem).(*ServConn)
	}
	w.Unlock()
	return
}

func (w *WaitQueue) Len() int {
	return w.queue.Len()
}


//
type ReadyHandlers struct {
	queue *linkedlist.List
	sync.Mutex
}

func NewReadyHandlers() *ReadyHandlers {
	return &ReadyHandlers{
		queue: linkedlist.New(),
	}
}

func (r *ReadyHandlers) Purge() {
	r.queue.Init()
}

func (r *ReadyHandlers) Push(h *handler) {
	r.Lock()
	r.queue.PushBack(h)
	r.Unlock()
}

func (r *ReadyHandlers) Pop() (h *handler) {
	r.Lock()
	elem := r.queue.Front()
	h = nil
	if elem != nil {
		h = r.queue.Remove(elem).(*handler)
	}
	r.Unlock()
	return
}

func (r *ReadyHandlers) Len() int {
	return r.queue.Len()
}



type slabGroupMap map[uint64]*SlabGroup


//
type Server struct {
	maxRoutines int
	handlers []*handler
	readyHdrs *ReadyHandlers
	waitQueue *WaitQueue
	hdrNotif chan interface{}
	quit chan bool

	memCfg MemConfig
	entry *ItemsEntry
	groups slabGroupMap

	sync.Mutex
}


func NewServer(c *MemConfig) *Server {
	c.maxStorageSize = c.MaxStorageSize()
	c.totalCapacity = 0
	dtrace.Logf("## Start server with config: %+v", c)

	return &Server{
		maxRoutines: c.MaxRoutines,
		handlers: make([]*handler, 0, c.MaxRoutines),
		readyHdrs: NewReadyHandlers(),
		waitQueue: NewWaitQueue(c.MaxRoutines * 10),
		hdrNotif: make(chan interface{}, c.MaxRoutines),
		quit: make(chan bool, 1),

		memCfg: *c,
		entry: NewItemsEntry(c.LRUSize, c.SkipListCheckStep),
		groups: make( slabGroupMap ),
	}
}

func (s *Server) Start() {
	s.initSlabs()
	s.entry.StartCheck()

	go func() {
		checks := 0
		timeout := 50 * time.Millisecond

		for {
			select {
			case i := <- s.hdrNotif:
				if h := i.(*handler); h != nil {
					h.state = HdrReady
					s.readyHdrs.Push(h)
				}

			case <- s.quit:
				dtrace.Log("quit server")
				return

			default:
				var err error
				for {
					if s.waitQueue.Len() == 0 {
						break
					}
					e := s.handleNext()
					if e != nil {
						if e != err {
							logger.Warnf("handling next: %v", e.Error())
							err = e
						}
						checks++
						break
					}

					err = e
					checks = 0
				}

				to := timeout
				if checks > 100 {
					to *= 10
				}
				time.Sleep(timeout)
			}
		}//for
	}()
}

func (s *Server) Stop() {
	s.entry.StopCheck()
	s.quit <- true
	s.clearSlabs()

	s.waitQueue.Purge()
	s.readyHdrs.Purge()

	s.Lock()
	for _, h := range s.handlers {
		h.stop()
	}
	s.Unlock()
	logger.Info("Server stop")
}

//
func (s *Server) initSlabs() {
	c := s.memCfg.SlotCapMin
	for {
		g := NewSlabGroup(c,
			s.memCfg.SlabsInGroup, s.memCfg.SlotsInSlab, s.memCfg.SlabCheckIntv, s.memCfg.maxStorageSize)
		s.memCfg.AddCapToTotal( g.Capacity() )
		s.groups[c] = g
		dtrace.Logf("Check group: %d; %d, %v", len(s.groups), c, g)

		c = c << szShift
		if c > s.memCfg.SlotCapMax {
			break
		}
	}
	dtrace.Logf("Total capacity in initialization - storage: %d, runtime: %d", s.memCfg.TotalCapacity(), getMemUsed())
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
func (s *Server) Handle(sc *ServConn) {
	s.waitQueue.Push(sc)
}

//
func (s *Server) handleNext() error {
	hdr := s.readyHdrs.Pop()

	if hdr == nil {
		if cnt := len(s.handlers); cnt < s.maxRoutines {
			dtrace.Logf("* Running handlers: %d", cnt)
			s.Lock()
			hdr = newHandler(cnt, s.hdrNotif, &s.memCfg, s.groups)
			s.handlers = append(s.handlers, hdr)
			s.Unlock()
		}
	}

	if hdr == nil {
		return errors.New("all handlers are busy")
	}

	sc := s.waitQueue.Pop()
	if sc == nil {
		s.readyHdrs.Push(hdr) //no conn to handle then put it back into ready-list
		return errors.New("no serv-conn in waiting")
	}

	dtrace.Logf("* Process conn[%d] with handler: %d", sc.index, hdr.index)

	go func(s *Server, h *handler, sc *ServConn) {
		if e := h.process(sc, s.entry); e != nil {
			logger.Errorf("error in handling connection[%d] by handler[%d]: %v", sc.index, h.index, e.Error())
		}
		sc.Close()
	}(s, hdr, sc)
	return nil
}






//
type handler struct {
	index int

	notif chan interface{}
	state HdrState

	cfg *MemConfig //only reference
	groups slabGroupMap //only reference
}

func newHandler(idx int, notif chan interface{}, c *MemConfig, groups slabGroupMap) *handler {
	return &handler{
		index: idx,
		notif: notif,
		state: HdrReady,
		cfg: c,
		groups: groups,
	}
}

func (h *handler) stop() {
	dtrace.Log("Handler stop at", h.index)
	h.state = HdrQuit
}


func (h *handler) process(sc *ServConn, entry *ItemsEntry) error {
	//dtrace.Log("Nothing here in handler...")

	h.state = HdrRunning

	defer func() {
		if err := recover(); err != nil {
			logger.Errorf("handler-process failed: %v", err)
		}

		h.state = HdrIdle
		dtrace.Log("handler process completed at index: ", h.index)
		h.notif <- h
	}()

	line, e := sc.rw.ReadSlice('\n')
	if e != nil {
		return e
	}

	msgline := &MsgLine{}
	msgline.parseLine(line)
	dtrace.Logf(" - Recv: %T %v\n - - -", msgline, msgline)

	log := logger.WithFields(logrus.Fields{
		"cmd": msgline.Cmd,
		"itemKey": msgline.Key,
	})
	log.Info("Incoming command")

	err := errors.New("unsupported command")

	if _StoreCmds[msgline.Cmd] {
		err = h.handleStorage(msgline, sc.rw, entry)
	} else if msgline.Cmd == "get" || msgline.Cmd == "gets" {
		err = h.handleRetrieval(msgline, sc.rw, entry)
	}
	return err
}



func (h *handler) handleStorage(msgline *MsgLine, rw *bufio.ReadWriter, entry *ItemsEntry) error {
	log := logger.WithFields(logrus.Fields{
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
			log.Errorf("Write buffer error for key[%s] at handler[%d]: %v", msgline.Key, h.index, e.Error())
		}
		if e := rw.Flush(); e != nil {
			log.Errorf("Flush buffer error for key[%s] at handler[%d]: %v", msgline.Key, h.index, e.Error())
		}
	}
	failResp := func(e error) error {
		makeResp(ResultNotStored)
		dtrace.Logf("Storage request failure for key[%s] at handler[%d]: %v", msgline.Key, h.index, e.Error())
		return e
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
		return failResp(err)
	}

	if e := h.allocSlots(item); e != nil {
		log.Errorf("Allocate slots error for key[%s] at handler[%d]: %v", msgline.Key, h.index, e.Error())
		_ = entry.Remove(item.key)

		return failResp(e)
	}

	bytesLeft := msgline.ValueLen
	for i, s := range item.slots {
		dtrace.Logf(" - For key[%s] at handler[%d] # slot|%d|: %d, byte-left: %d", msgline.Key, h.index, s.capacity, i, bytesLeft)

		s.SetInfoWithItem(item)
		if n, e := s.ReadAndSet(msgline.Key, rw, bytesLeft); e != nil {
			log.Errorf("Error when read buffer and set into slot for key[%s] by handler[%d]: %v", msgline.Key, h.index, e.Error())
			dtrace.Logf(" - For key[%s] by handler[%d] # Read data failure: %v", msgline.Key, h.index, e.Error())

			return failResp(e)
		} else {
			bytesLeft -= n
		}
	}

	makeResp(ResultStored)
	dtrace.Logf("Storage request success for key[%s] at handler[%d]", msgline.Key, h.index)
	log.Infof("Successful command for storage for key[%s]", msgline.Key)
	return nil
}


func (h *handler) allocSlots(t *MetaItem) error {
	byteLen := t.byteLen
	c := h.cfg.SlotCapMax
	for {
		dtrace.Logf("Check slab-groups for capacity: %d; Left-bytes: %d", c, byteLen)

		if byteLen >= c || c == h.cfg.SlotCapMin {
			rest := byteLen % c
			cnt := 0
			if byteLen > rest {
				cnt = int( (byteLen - rest) / c )
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


func (h *handler) findSlots(slotCap uint64, cnt int, t *MetaItem) error {
	dtrace.Logf("Finding %d slots for key[%s] with cap[%d]", cnt, t.key, slotCap)

	getTotalCap := func() uint64 {
		return h.cfg.TotalCapacity()
	}

	group := h.groups[slotCap]
	if slots, extraCap, e := group.FindAvailableSlots(t.key, cnt, getTotalCap); e == nil {
		if Dev {
			arr := make([]string, 0, len(slots))
			for _, s := range slots {
				p := fmt.Sprintf("%p", s)
				arr = append(arr, p)
			}
			dtrace.Logf(" - Got slots for key[%s] with cap[%d]: %v", t.key, slotCap, arr)
		}

		if extraCap > 0 {
			h.cfg.AddCapToTotal(extraCap)
		}

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
	log := logger.WithFields(logrus.Fields{
		"cmd": msgline.Cmd,
		"itemKey": msgline.Key,
	})

	endResp := func(er error) {
		if _, e := rw.Write(ResultEnd); e != nil {
			log.Errorf("write buffer error: %v", e.Error())
		}
		if e := rw.Flush(); e != nil {
			log.Errorf("flush buffer error: %v", e.Error())
		}

		if er == nil {
			log.Info("Successful command for retrieval")
		}
	}

	item := entry.Get(msgline.Key)
	if item == nil {
		item = NewMetaItem(msgline.Key, 0, 0, 0)
	}

	// Write value block and end with \r\n
	byteLen := item.byteLen
	for _, s := range item.slots {
		if s.Vacant() {
			byteLen = 0
			break
		}
	}

	if e := h.writeRespFirstLine(item, rw, byteLen); e != nil {
		endResp(e)
		return e
	}

	if byteLen > 0 {
		for _, s := range item.slots {
			bytes := s.Data()
			if _, e := rw.Write(bytes); e != nil {
				endResp(e)
				return e
			}
		}
		if _, e := rw.Write([]byte("\r\n")); e != nil {
			endResp(e)
			return e
		}
	}

	endResp(nil)
	return nil
}

func (h *handler) writeRespFirstLine(item *MetaItem, rw *bufio.ReadWriter, byteLen uint64) error {
	line := fmt.Sprintf("VALUE %s %d %d\r\n", item.key, item.flags, byteLen)
	if _, err := rw.Write([]byte(line)); err != nil {
		return err
	}
	return nil
}

