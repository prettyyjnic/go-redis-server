// go-redis-server is a helper library for building server software capable of speaking the redis protocol.
// This could be an alternate implementation of redis, a custom proxy to redis,
// or even a completely different backend capable of "masquerading" its API as a redis database.

package redis

import (
	"fmt"
	"net"
	"reflect"
)

type Server struct {
	Proto        string
	Addr         string // TCP address to listen on, ":6389" if empty
	MonitorChans []chan string
	methods      map[string]HandlerFn
	exitChan	 chan struct{}
}

func (srv *Server) ListenAndServe() error {
	addr := srv.Addr
	if srv.Proto == "" {
		srv.Proto = "tcp"
	}
	if srv.Proto == "unix" && addr == "" {
		addr = "/tmp/redis.sock"
	} else if addr == "" {
		addr = ":6389"
	}
	l, e := net.Listen(srv.Proto, addr)
	if e != nil {
		return e
	}
	return srv.Serve(l)
}

// Serve accepts incoming connections on the Listener l, creating a
// new service goroutine for each.  The service goroutines read requests and
// then call srv.Handler to reply to them.
func (srv *Server) Serve(l net.Listener) error {
	defer l.Close()
	srv.MonitorChans = []chan string{}
	for {
		select{
		case <-srv.exitChan:
			return nil
		default:
			rw, err := l.Accept()
			if err != nil {
				return err
			}
			go srv.ServeClient(rw)
		}
	}
}

// Serve starts a new redis session, using `conn` as a transport.
// It reads commands using the redis protocol, passes them to `handler`,
// and returns the result.
func (srv *Server) ServeClient(conn net.Conn) (err error) {
	clientChan := make(chan struct{})
	defer func() {
		if err != nil {
			fmt.Fprintf(conn, "-%s\n", err)
		}
		Debugf("Client disconnected")
		close(clientChan)
		//close()
		conn.Close()
	}()

	var clientAddr string

	switch co := conn.(type) {
	case *net.UnixConn:
		f, err := conn.(*net.UnixConn).File()
		if err != nil {
			return err
		}
		clientAddr = f.Name()
	default:
		clientAddr = co.RemoteAddr().String()
	}

	for {
		select{
		case <-srv.exitChan:
			return nil
		default:
			request, err := parseRequest(conn)
			if err != nil {
				return err
			}
			request.Host = clientAddr
			reply, err := srv.Apply(request)
			if err != nil {
				return err
			}
			if _, err = reply.WriteTo(conn); err != nil {
				return err
			}
		}
	}
	return nil
}

func (srv *Server) Shutdown()  {
	Debugf("server exiting...")
	close(srv.exitChan)
}

func NewServer(c *Config) (*Server, error) {
	srv := &Server{
		Proto:        c.proto,
		MonitorChans: []chan string{},
		methods:      make(map[string]HandlerFn),
	}

	if srv.Proto == "unix" {
		srv.Addr = c.host
	} else {
		srv.Addr = fmt.Sprintf("%s:%d", c.host, c.port)
	}

	if c.handler == nil {
		c.handler = NewDefaultHandler()
	}

	rh := reflect.TypeOf(c.handler)
	for i := 0; i < rh.NumMethod(); i++ {
		method := rh.Method(i)
		if method.Name[0] > 'a' && method.Name[0] < 'z' {
			continue
		}
		Debugf(method.Name)
		handlerFn, err := srv.createHandlerFn(c.handler, &method.Func)
		if err != nil {
			return nil, err
		}
		srv.Register(method.Name, handlerFn)
	}
	srv.exitChan = make(chan struct{}, 1)
	return srv, nil
}
