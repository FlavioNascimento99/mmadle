package domain

import (
	"errors"
	"fmt"
)

// Pool narrows which fighters take part in a game. Each pool has its own
// daily target, drawn from the fighters whose current division belongs to it.
type Pool string

const (
	PoolAll Pool = "all"
	PoolMen Pool = "men"
)

// ErrUnknownPool is returned for pool names outside the supported set.
var ErrUnknownPool = errors.New("unknown pool")

// ParsePool maps a request value to a Pool; empty means PoolAll.
func ParsePool(s string) (Pool, error) {
	switch s {
	case "", string(PoolAll):
		return PoolAll, nil
	case string(PoolMen):
		return PoolMen, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnknownPool, s)
	}
}
