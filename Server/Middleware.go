package Server

import "time"

var (

	RateLimitPage *RateLimiter
	RateLimitWSConnect *RateLimiter
	RateLimitSuggestions *RateLimiter
	RateLimitAuthLogin *RateLimiter
	RateLimitAuthCallback *RateLimiter
	RateLimitAuthMe *RateLimiter

)

func init() {

	RateLimitPage = NewRateLimiter(120, time.Minute)
	RateLimitWSConnect = NewRateLimiter(30, time.Minute)
	RateLimitSuggestions = NewRateLimiter(120, time.Minute)
	RateLimitAuthLogin = NewRateLimiter(20, time.Minute)
	RateLimitAuthCallback = NewRateLimiter(20, time.Minute)
	RateLimitAuthMe = NewRateLimiter(180, time.Minute)

}
