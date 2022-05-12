package testutils

// NOTE: Not really an assert (since the check is actually performed). Maybe rename?

// Assert(condition) panics if condition is false; Assert(condition, error) panics if condition is false with panic(error).
func Assert(condition bool, err ...interface{}) {
	if len(err) > 1 {
		panic("bandersnatch / testutils: Assert can only handle 1 extra error argument")
	}
	if !condition {
		if len(err) == 0 {
			panic("This is not supposed to be possible")
		} else {
			panic(err[0])
		}
	}
}
