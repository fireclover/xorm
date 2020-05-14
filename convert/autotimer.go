package convert

import "time"

type AutoTimer interface {
	AutoTime(t time.Time) (interface{}, error)
}
