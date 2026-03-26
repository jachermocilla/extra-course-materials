package types

import "github.com/pixperk/plethora/vclock"

type Key string

type Value struct {
	Data  string
	Clock vclock.VClock
}
