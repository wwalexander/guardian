package guardian

import (
	"context"
	"math"
)

const RequestsRemainingMax = math.MaxUint32

// RequestBlockerFunc is a function that evaluates a given request and determines if it should be blocked or not and how many requests are remaining.
type RequestBlockerFunc func(context.Context, Request) (bool, uint32, error)

type RateLimiter interface {
	Limit(context.Context, Request) (bool, uint32, error)
}

// CondRequestBlockerFunc is the same as a RequestBlockerFunc with the added ability to indicate that the evaluation of a chain should stop
type CondRequestBlockerFunc func(context.Context, Request) (stop, blocked bool, remaining uint32, err error)

// DefaultCondChain is the default condition chain used by Guardian. This performs the following checks when
// processing a request: whitelist, blacklist, rate limiters.
func DefaultCondChain(whitelister *IPWhitelister, blacklister *IPBlacklister, jailer Jailer, rateLimiters ...RateLimiter) RequestBlockerFunc {
	condWhitelistFunc := CondStopOnWhitelistFunc(whitelister)
	condBlacklistFunc := CondStopOnBlacklistFunc(blacklister)
	condJailerFunc := CondStopOnBanned(jailer)
	rbfs := []CondRequestBlockerFunc{condWhitelistFunc, condBlacklistFunc, condJailerFunc}
	for _, rl := range rateLimiters {
		rbfs = append(rbfs, CondStopOnBlockOrError(rl))
	}
	return CondChain(rbfs...)
}

// CondChain chains a series of CondRequestBlockerFunc running each until one indicates the chain should stop processing, returning that functions results
func CondChain(cf ...CondRequestBlockerFunc) RequestBlockerFunc {
	return func(c context.Context, r Request) (bool, uint32, error) {
		minRemaining := uint32(math.MaxUint32)
		for _, f := range cf {
			stop, blocked, remaining, err := f(c, r)
			if err != nil && stop {
				return blocked, 0, err
			}

			if remaining < minRemaining {
				minRemaining = remaining
			}
			if stop || blocked {
				return blocked, minRemaining, nil
			}
		}

		return false, minRemaining, nil
	}
}

// CondStopOnBlockOrError wraps a RateLimiter Limit call and returns true for stop if the request was blocked or errored out
func CondStopOnBlockOrError(rl RateLimiter) CondRequestBlockerFunc {
	return func(c context.Context, r Request) (bool, bool, uint32, error) {
		blocked, remaining, err := rl.Limit(c, r)
		stop := (blocked || err != nil)

		return stop, blocked, remaining, err
	}
}
