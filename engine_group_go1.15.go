// +build go1.15

package xorm

import (
	"time"
)

// SetConnMaxIdleTime sets the maximum amount of time a connection may be idle.
//
// Expired connections may be closed lazily before reuse.
//
// If d <= 0, connections are not closed due to a connection's idle time.
func (eg *EngineGroup) SetConnMaxIdleTime(d time.Duration) {
	eg.Engine.SetConnMaxIdleTime(d)
	for i := 0; i < len(eg.slaves); i++ {
		eg.slaves[i].SetConnMaxIdleTime(d)
	}
}
