package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/realip"
	"golang.org/x/time/rate"
)

type perIP struct {
	sync.Mutex
	every    time.Duration
	burst    int
	limiters map[string]*rate.Limiter
}

// Middleware implementing a Token Bucket rate limiter indexed on the IP
func PerIP(every time.Duration, burst int, next http.Handler) http.Handler {
	p := &perIP{every: every, burst: burst, limiters: map[string]*rate.Limiter{}}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !p.limiter(realip.From(r)).Allow() {
			http.Error(w, "Too many requests, please wait a moment and try again", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (p *perIP) limiter(ip string) *rate.Limiter {
	p.Lock()
	defer p.Unlock()
	for k, l := range p.limiters {
		if k != ip && l.Tokens() >= float64(p.burst) {
			delete(p.limiters, k)
		}
	}
	l, ok := p.limiters[ip]
	if !ok {
		l = rate.NewLimiter(rate.Every(p.every), p.burst)
		p.limiters[ip] = l
	}
	return l
}
