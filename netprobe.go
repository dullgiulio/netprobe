package netprobe

import (
	"context"
	"fmt"
	"net"
	"time"
)

type result struct {
	err  error
	conn net.Conn
}

type probe struct {
	addrs    []string
	parallel int
	network  string
}

func newProbe(network string, addrs []string) *probe {
	return &probe{
		addrs:    addrs,
		parallel: 50,
		network:  network,
	}
}

func (p *probe) dial(ctx context.Context, timeout time.Duration, addr string) *result {
	var (
		cancel context.CancelFunc
		dialer net.Dialer
	)
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	c, err := dialer.DialContext(ctx, p.network, addr)
	if err != nil {
		return &result{err: err}
	}
	return &result{conn: c}
}

func (p *probe) start(ctx context.Context, timeout time.Duration, addrs <-chan string, res chan<- *result) {
	for addr := range addrs {
		r := p.dial(ctx, timeout, addr)
		res <- r
		// if a connection was found or everything is cancelled, no point in trying more addresses
		if r.err == nil || r.err == context.Canceled {
			break
		}
	}
	for addr := range addrs {
		res <- &result{err: fmt.Errorf("skipping %s", addr)}
	}
}

func (p *probe) run(ctx context.Context, timeout time.Duration) (net.Conn, error) {
	parallel := p.parallel
	if parallel > len(p.addrs) {
		parallel = len(p.addrs)
	}
	addrs := make(chan string)
	res := make(chan *result)

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	for i := 0; i < parallel; i++ {
		go p.start(ctx, timeout, addrs, res)
	}

	go func() {
		for i := range p.addrs {
			addrs <- p.addrs[i]
		}
		close(addrs)
	}()

	var err error
	for i := 0; i < len(p.addrs); i++ {
		r := <-res
		if r.err == nil {
			// catch all cancellation and timeout errors from other addresses
			go func(i int) {
				for ; i < len(p.addrs); i++ {
					r := <-res
					if r.conn != nil {
						r.conn.Close()
					}
				}
			}(i)
			// cancel ongoing dials for other addresses
			cancel()

			return r.conn, nil
		}
		err = r.err
	}

	cancel()
	return nil, fmt.Errorf("cannot probe %v: %w", p.addrs, err)
}

// Probe dials multiple addresses in parallel and returns the first available connection
func Dial(ctx context.Context, network string, addrs []string, timeout time.Duration) (net.Conn, error) {
	p := newProbe(network, addrs)
	return p.run(ctx, timeout)
}
