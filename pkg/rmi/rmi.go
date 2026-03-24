package rmi

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/winezer0/revsuit/internal/database"
	"github.com/winezer0/revsuit/internal/ipinfo"
	"github.com/winezer0/revsuit/internal/recycler"
	log "unknwon.dev/clog/v2"
)

type Server struct {
	Config
	rules     []*Rule
	rulesLock sync.RWMutex
	stateLock sync.Mutex
	running   bool
	stopCh    chan struct{}
	stopOnce  sync.Once
	listener  net.Listener
}

var (
	server *Server
	once   sync.Once
)

func GetServer() *Server {
	once.Do(func() {
		server = &Server{rulesLock: sync.RWMutex{}, stateLock: sync.Mutex{}}
	})
	return server
}

func (s *Server) getRules() []*Rule {
	defer s.rulesLock.RUnlock()
	s.rulesLock.RLock()
	return s.rules
}

func (s *Server) UpdateRules() error {
	db := database.DB.Model(new(Rule))
	defer s.rulesLock.Unlock()
	s.rulesLock.Lock()
	return errors.Wrap(db.Order("base_rank desc").Find(&s.rules).Error, "RMI update rules error")
}

func parseRMIPath(data []byte, n int) (string, bool) {
	if n <= 0 || len(data) < n {
		return "", false
	}
	frags := bytes.Split(data[:n], []byte{0xdf, 0x74})
	if len(frags) == 0 {
		return "", false
	}
	last := frags[len(frags)-1]
	if len(last) < 2 {
		return "", false
	}
	return strings.TrimRight(string(last[2:]), "\x00"), true
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		_ = conn.Close()
		if err := recover(); err != nil {
			recycler.Recycle(err)
		}
	}()

	if err := conn.SetDeadline(time.Now().Add(time.Second * 30)); err != nil {
		log.Warn("RMI set connection deadline error:%v", err)
		return
	}

	ip, port, _ := net.SplitHostPort(conn.RemoteAddr().String())

	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		log.Warn("RMI read connection error:%v", err)
		return
	}

	if !bytes.Contains(buf, []byte{0x4a, 0x52, 0x4d, 0x49}) {
		return
	}

	send := []byte{0x4e}
	bs := make([]byte, 8)
	binary.BigEndian.PutUint16(bs, uint16(len(ip)))
	send = append(send, bs...)
	send = append(send, []byte(ip)...)
	send = append(send, []byte{0x00, 0x00}...)
	uintPort, _ := strconv.Atoi(port)
	bs = make([]byte, 8)
	binary.BigEndian.PutUint16(bs, uint16(uintPort))
	send = append(send, bs...)

	_, err = conn.Write(send)
	if err != nil {
		log.Warn("RMI write connection error: %v", err)
		return
	}

	data := make([]byte, 512)
	length := 0

	for length < 50 && length < len(data) {
		n, err := conn.Read(data)
		if err != nil {
			log.Warn("RMI read connection error: %v", err)
			return
		}
		length += n
	}
	path, ok := parseRMIPath(data, length)
	if !ok {
		log.Warn("RMI invalid packet from %s", conn.RemoteAddr())
		return
	}

	for _, _rule := range s.getRules() {
		flag, flagGroup, _ := _rule.Match(path)
		if flag == "" {
			continue
		}

		area := ipinfo.Area(ip)

		// create new record
		r, err := NewRecord(_rule, flag, path, ip, area)
		if err != nil {
			log.Warn("RMI record[rule_id:%d] created failed :%s", _rule.ID, err)
			return
		}
		log.Info("RMI record[id:%d rule:%s remote_ip:%s] has been created", r.ID, _rule.Name, ip)

		//only send to client when this connection recorded first time.
		if _rule.PushToClient {
			if flagGroup != "" {
				var count int64
				database.DB.Where("rule_name=? and path like ?", _rule.Name, "%"+flagGroup+"%").Model(&Record{}).Count(&count)
				if count <= 1 {
					r.PushToClient()
					log.Trace("RMI record[id:%d, flagGroup:%s] has been put to client message queue", r.ID, flagGroup)
				}
			} else {
				r.PushToClient()
				log.Trace("RMI record[id:%d, flag:%s] has been put to client message queue", r.ID, flag)
			}
		}

		//send notice
		if _rule.Notice {
			go func() {
				r.Notice()
				log.Trace("RMI record[id:%d] notice has been sent", r.ID)
			}()
		}
		return
	}
}

func (s *Server) Stop() {
	log.Info("RMI Server is stopping...")
	s.stateLock.Lock()
	if !s.running {
		s.Enable = false
		s.stateLock.Unlock()
		return
	}
	s.running = false
	s.Enable = false
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	listener := s.listener
	s.stateLock.Unlock()
	if listener != nil {
		_ = listener.Close()
	}
}

func (s *Server) Restart() {
	s.Stop()
	time.Sleep(time.Second * 2)
	go s.Run()
}

func (s *Server) Run() {
	s.stateLock.Lock()
	if s.running {
		s.stateLock.Unlock()
		return
	}
	s.running = true
	s.Enable = true
	s.stopCh = make(chan struct{})
	s.stopOnce = sync.Once{}
	s.stateLock.Unlock()
	defer func() {
		s.stateLock.Lock()
		s.running = false
		s.Enable = false
		s.listener = nil
		s.stateLock.Unlock()
	}()

	if err := s.UpdateRules(); err != nil {
		log.Error("%v", err)
		return
	}

	// run server
	log.Info("Starting RMI Server at %v", s.Addr)

	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		log.Error("%v", err)
		return
	}
	s.stateLock.Lock()
	s.listener = listener
	s.stateLock.Unlock()

	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				log.Warn("RMI accept connection error: %v", err)
			} else {
				break
			}
			continue
		}
		go s.handleConnection(tcpConn)
	}
}
