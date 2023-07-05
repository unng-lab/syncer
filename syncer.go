package syncer

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrStopped       = errors.New("syncer stopped")
	ErrPaused        = errors.New("syncer paused")
	ErrNotPaused     = errors.New("syncer not paused")
	ErrFuncNotFound  = errors.New("func not found")
	ErrSpeedModifier = errors.New("speed modifier cant be between -1 and 1")
	ErrUnprocessable = errors.New("cur speed zero unprocessable")
)

type Syncer struct {
	stopChan chan struct{}
	stopped  atomic.Bool

	mutex       sync.RWMutex
	deletedList []int
	unitList    []runner

	balanceChan chan struct{}
	nextChan    chan struct{}

	paused      atomic.Bool
	pauseChan   chan struct{}
	unPauseChan chan struct{}

	curSpeed  int
	speedChan chan int
}

func NewSyncer() *Syncer {
	s := Syncer{
		stopChan:    make(chan struct{}),
		unitList:    make([]runner, 0, 64),
		balanceChan: make(chan struct{}, 2),
		nextChan:    make(chan struct{}, 1),
		pauseChan:   make(chan struct{}, 1),
		unPauseChan: make(chan struct{}, 1),
		curSpeed:    1,
		speedChan:   make(chan int, 1),
	}
	go s.run()
	return &s
}

func (s *Syncer) run() {
	s.nextChan <- struct{}{}
	for {
		select {
		case <-s.nextChan:
			s.runUnits()
		case <-s.stopChan:
			return
		case <-s.pauseChan:
			<-s.unPauseChan
		case v := <-s.speedChan:
			err := s.changeSpeed(v)
			if err != nil {
				return
			}
		}
	}
}

func (s *Syncer) ChangeSpeed(v int) {
	s.speedChan <- v
}

func (s *Syncer) changeSpeed(v int) error {
	if v == 0 || v == -1 || v == 1 {
		return ErrSpeedModifier
	}

	if s.curSpeed == 0 {
		return ErrUnprocessable
	}

	if v > 1 {
		if s.curSpeed >= 1 {
			s.curSpeed = s.curSpeed * v
			s.balanceChan = make(chan struct{}, s.curSpeed)
		} else {
			if s.curSpeed/v > -1 {
				s.curSpeed = v + s.curSpeed
				s.balanceChan = make(chan struct{}, s.curSpeed)
			} else {
				s.curSpeed = s.curSpeed / v
			}
		}
	} else {
		if s.curSpeed <= -1 {
			s.curSpeed = s.curSpeed * v * -1
		} else {
			if s.curSpeed/v < 1 {
				s.curSpeed = v + s.curSpeed
				s.balanceChan = make(chan struct{}, 1)
			} else {
				s.curSpeed = s.curSpeed / v * -1
			}
		}
	}
	return nil
}

func (s *Syncer) runUnits() {
	t := time.Now()
	var wg sync.WaitGroup
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for k := range s.unitList {
		if s.unitList[k].deleted {
			continue
		}
		s.balanceChan <- struct{}{}
		wg.Add(1)
		s.unitList[k].exec(&wg)
	}
	wg.Wait()
	if s.curSpeed < 0 {
		time.Sleep(time.Since(t) * time.Duration(s.curSpeed*-1))
	}
	s.nextChan <- struct{}{}
}

func (s *Syncer) Add(u Unit) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var key int
	if l := len(s.deletedList); l > 0 {
		key = s.deletedList[l-1]
		s.unitList[key].unit = u
		s.unitList[key].deleted = false
		s.deletedList = s.deletedList[:l-1]
	} else {
		s.unitList = append(s.unitList, runner{
			unit:     u,
			runChan:  make(chan *sync.WaitGroup, 1),
			syncer:   s,
			stopChan: make(chan struct{}, 1),
		})
		key = len(s.unitList) - 1
	}

	go s.unitList[key].run()
	return key, nil
}

func (s *Syncer) Remove(key int) error {
	if key < 0 && key >= len(s.unitList) {
		return ErrFuncNotFound
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.unitList[key].deleted = true
	s.deletedList = append(s.deletedList, key)
	return nil
}

func (s *Syncer) Pause() error {
	if s.paused.Load() {
		return ErrPaused
	}
	s.paused.Store(true)
	s.pauseChan <- struct{}{}
	return nil
}

func (s *Syncer) Play() error {
	if !s.paused.Load() {
		return ErrNotPaused
	}
	s.paused.Store(false)
	s.unPauseChan <- struct{}{}
	return nil
}

func (s *Syncer) Slower() {
	s.ChangeSpeed(-2)
}

func (s *Syncer) Faster() {
	s.ChangeSpeed(2)
}

func (s *Syncer) Stop() error {
	s.stopped.Store(true)
	s.stopChan <- struct{}{}
	return nil
}

func (s *Syncer) check() error {
	if s.stopped.Load() {
		return ErrStopped
	}
	return nil
}
