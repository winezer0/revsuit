package ldap

import (
	"bytes"
	"net"
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
	return errors.Wrap(db.Order("base_rank desc").Find(&s.rules).Error, "LDAP update rules error")
}

func parsePath(buf []byte, n int) (string, bool) {
	if n < 10 || len(buf) < n || len(buf) < 9 {
		return "", false
	}
	length := int(buf[8])
	if length <= 0 || 9+length > n {
		return "", false
	}
	return string(buf[9 : 9+length]), true
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		_ = conn.Close()
		if err := recover(); err != nil {
			recycler.Recycle(err)
		}
	}()

	if err := conn.SetDeadline(time.Now().Add(time.Second * 30)); err != nil {
		log.Warn("LDAP set connection deadline error:%v", err)
		return
	}

	ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Warn("LDAP read connection error:%v", err)
		return
	}

	if !bytes.Contains(buf[:n], []byte{
		0x30, 0x0c, 0x02, 0x01, 0x01, 0x60, 0x07,
		0x02, 0x01, 0x03, 0x04, 0x00, 0x80, 0x00}) {
		return
	}

	send := []byte{
		0x30, 0x0c, 0x02, 0x01, 0x01, 0x61, 0x07,
		0x0a, 0x01, 0x00, 0x04, 0x00, 0x04, 0x00,
	}
	_, err = conn.Write(send)
	if err != nil {
		log.Warn("LDAP write connection error: %v", err)
		return
	}
	n, err = conn.Read(buf)
	if err != nil {
		log.Warn("LDAP read connection error:%v", err)
		return
	}
	path, ok := parsePath(buf, n)
	if !ok {
		log.Warn("LDAP invalid packet from %s", conn.RemoteAddr())
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
			log.Warn("LDAP record[rule_id:%d] created failed :%s", _rule.ID, err)
			return
		}
		log.Info("LDAP record[id:%d rule:%s remote_ip:%s] has been created", r.ID, _rule.Name, ip)

		//only send to client when this connection recorded first time.
		if _rule.PushToClient {
			if flagGroup != "" {
				var count int64
				database.DB.Where("rule_name=? and path like ?", _rule.Name, "%"+flagGroup+"%").Model(&Record{}).Count(&count)
				if count <= 1 {
					r.PushToClient()
					log.Trace("LDAP record[id:%d, flagGroup:%s] has been put to client message queue", r.ID, flagGroup)
				}
			} else {
				r.PushToClient()
				log.Trace("LDAP record[id:%d, flag:%s] has been put to client message queue", r.ID, flag)
			}
		}

		//send notice
		if _rule.Notice {
			go func() {
				r.Notice()
				log.Trace("LDAP record[id:%d] notice has been sent", r.ID)
			}()
		}
		return
	}
}

func (s *Server) Stop() {
	log.Info("LDAP Server is stopping...")
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
	log.Info("Starting LDAP Server at %v", s.Addr)

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
				log.Warn("LDAP accept connection error: %v", err)
			} else {
				break
			}
			continue
		}
		go s.handleConnection(tcpConn)
	}
}
