// package siglistener provides signal listening as a module.
package siglistener

import (
	"os"
	"os/signal"
)

type Listener struct {
	ch   chan os.Signal
	sigs []os.Signal
}

func New(signals ...os.Signal) *Listener {
	return &Listener{
		sigs: signals,
	}
}

func (l *Listener) Init() error {
	l.ch = make(chan os.Signal, 1)
	signal.Notify(l.ch, l.sigs...)
	return nil
}

func (l *Listener) Run() error {
	<-l.ch
	return nil
}

func (l *Listener) Stop() error {
	defer close(l.ch)
	signal.Stop(l.ch)
	return nil
}

func (l *Listener) Name() string {
	return "siglistener.Listener"
}
