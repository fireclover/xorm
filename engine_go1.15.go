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
func (engine *Engine) SetConnMaxIdleTime(d time.Duration) {
	engine.DB().SetConnMaxIdleTime(d)
}
