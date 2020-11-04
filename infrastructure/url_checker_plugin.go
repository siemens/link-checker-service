package infrastructure

import "context"

// URLCheckerPlugin represents one low-level URL checker in a chain of checkers
type URLCheckerPlugin interface {
	// CheckURL gets the urlToCheck and lastResult, which it can process, and return the next result
	// and a boolean flag, whether the chain should be interrupted, and the last result - simply returned
	// ctx can be used to cancel the request prematurely
	CheckURL(ctx context.Context, urlToCheck string, lastResult *URLCheckResult) (*URLCheckResult, bool)
}
