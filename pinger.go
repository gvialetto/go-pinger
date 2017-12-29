package pinger

import (
	"context"
	"encoding/binary"
	"math/rand"
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"
)

// ReplyHandlerFunc is the type of handler that is called every time a
// ping response is received (or alternatively an error occurs)
type ReplyHandlerFunc func(string, time.Duration, error)

type destination struct {
	host             string
	addr             net.Addr
	timeout          time.Duration
	interval         time.Duration
	cancelFunc       context.CancelFunc
	receivedMessages chan []byte
}

type Pinger struct {
	conn          *icmp.PacketConn
	proto         string
	ctx           context.Context
	dests         map[string]*destination
	mtx           *sync.Mutex
	wg            *sync.WaitGroup
	isRunning     bool
	buffers       *bufferPool
	replyHandlers []ReplyHandlerFunc
}

func New() (*Pinger, error) {
	conn, proto, err := getDefaultICMPListener()
	if err != nil {
		return nil, err
	}
	return &Pinger{
		conn:          conn,
		proto:         proto,
		ctx:           nil,
		dests:         make(map[string]*destination),
		mtx:           &sync.Mutex{},
		wg:            &sync.WaitGroup{},
		buffers:       newBufPool(512),
		replyHandlers: nil,
	}, nil
}

// Run the pinger
func (p *Pinger) Run(ctx context.Context) {
	p.wg.Add(1)
	// IDEA: Maybe use a child context for this also?
	go p.startICMPListener(ctx, 100*time.Millisecond)
	p.mtx.Lock()
	for _, dest := range p.dests {
		destCtx, cancel := context.WithCancel(ctx)
		dest.cancelFunc = cancel
		p.wg.Add(1)
		go p.startICMPHandler(destCtx, dest)
	}
	p.ctx = ctx
	p.isRunning = true
	p.mtx.Unlock()
}

func (p *Pinger) AddHandler(handler ReplyHandlerFunc) {
	if p.replyHandlers == nil {
		p.replyHandlers = make([]ReplyHandlerFunc, 1)
		p.replyHandlers[0] = handler
	}
}

// Wait after cancelling the context passed to Run() for all goroutines to
// terminate. Useful if Run() was launched in another goroutine.
func (p *Pinger) Wait() {
	p.isRunning = false
	p.wg.Wait()
}

// AddHost appends an host to the list of hosts to be pinged
func (p *Pinger) AddHost(host string, interval time.Duration, timeout time.Duration) error {
	ip, addr, err := getAddr(p.conn, host)
	if err != nil {
		return err
	}

	messages := make(chan []byte, 1)

	p.mtx.Lock()
	defer p.mtx.Unlock()
	dest := &destination{
		addr:             addr,
		host:             host,
		cancelFunc:       nil, // this will be set when we actually start a routine
		timeout:          timeout,
		interval:         interval,
		receivedMessages: messages,
	}
	p.dests[ip.String()] = dest

	// If we're already running, start another handler on the fly
	if p.isRunning {
		destCtx, cancel := context.WithCancel(p.ctx)
		dest.cancelFunc = cancel
		p.wg.Add(1)
		go p.startICMPHandler(destCtx, dest)
	}

	return nil
}

func (p *Pinger) startICMPListener(ctx context.Context, readTimeout time.Duration) {
	defer p.wg.Done()
	defer p.conn.Close()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		bytes := p.buffers.Get()
		p.conn.SetReadDeadline(time.Now().Add(readTimeout))
		_, addr, err := p.conn.ReadFrom(bytes)
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				// this is caused by our deadline set above, ignore and continue
				if neterr.Timeout() {
					continue
				}
			}
			// TODO: fail here?
			continue
		}
		echoMessage := parseICMPMessage(bytes)
		if echoMessage == nil {
			// TODO: fail here?
			continue
		}
		var ip net.IP
		switch netAddr := addr.(type) {
		case *net.UDPAddr:
			ip = netAddr.IP
		case *net.IPAddr:
			ip = netAddr.IP
		}
		p.mtx.Lock()
		destination := p.dests[ip.String()]
		p.mtx.Unlock()
		if destination != nil && destination.receivedMessages != nil {
			dataBuf := p.buffers.Get()
			copy(dataBuf, echoMessage.Data)
			select {
			case <-ctx.Done():
				p.buffers.Release(bytes)
				return
			case destination.receivedMessages <- dataBuf:
			default:
			}
		}
		p.buffers.Release(bytes)
	}
}

func (p *Pinger) startICMPHandler(ctx context.Context, dest *destination) {
	defer p.wg.Done()
	rand.Seed(time.Now().UnixNano())
	destID := rand.Intn(0xffff)
	buf := make([]byte, 8)
	seq := 1
	sendEcho := func() {
		seq++
		binary.LittleEndian.PutUint64(buf, uint64(time.Now().UnixNano()))
		sendICMPEchoMessage(p.conn, dest.addr, destID, seq, buf)
	}
	sendEcho()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(dest.interval):
			sendEcho()
		case msg := <-dest.receivedMessages:
			t := time.Unix(0, int64(binary.LittleEndian.Uint64(msg)))
			p.buffers.Release(msg)
			for _, handler := range p.replyHandlers {
				handler(dest.host, time.Since(t), nil)
			}
		}
	}
}
