package selector

import "time"

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func NewClock() realClock {
	return realClock{}
}
