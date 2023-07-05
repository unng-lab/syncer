package syncer

import (
	"log"
	"sync"
)

type Unit interface {
	Process() error
}

type runner struct {
	unit     Unit
	runChan  chan *sync.WaitGroup
	stopChan chan struct{}
	deleted  bool
	syncer   *Syncer
}

func (fr *runner) run() {
	for {
		select {
		case wg := <-fr.runChan:
			err := fr.unit.Process()
			if err != nil {
				log.Println(err)
			}
			wg.Done()
			<-fr.syncer.balanceChan
		case <-fr.stopChan:
			return
		}
	}
}

func (fr *runner) stop() {
	close(fr.stopChan)
}

func (fr *runner) exec(wg *sync.WaitGroup) {
	fr.runChan <- wg
}
