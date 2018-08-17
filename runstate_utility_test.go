package flow

import (
	"github.com/google/go-cmp/cmp"
)

var runStateEqualOpts = cmp.Comparer(func(x, y RunState) bool {
	if x.value != y.value {
		return false
	}
	if x.branch != nil && y.branch != nil {
		if *x.branch != *y.branch {
			return false
		}
	} else if x.branch != nil || y.branch != nil {
		return false
	}
	if x.err != nil && y.err != nil {
		if x.err.Error() != y.err.Error() {
			return false
		}
	} else if x.err != nil || y.err != nil {
		return false
	}
	return true
})