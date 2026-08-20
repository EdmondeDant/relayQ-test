package repository

import (
	"context"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"time"
)

var signupBonusIPScript = redis.NewScript(`local n=redis.call('INCR',KEYS[1]);if n==1 then redis.call('PEXPIRE',KEYS[1],ARGV[1]) end;return n`)

type signupBonusIPLimiter struct{ r *redis.Client }

func NewSignupBonusIPLimiter(r *redis.Client) service.SignupBonusIPLimiter {
	return &signupBonusIPLimiter{r: r}
}
func (s *signupBonusIPLimiter) Allow(ctx context.Context, ip string, now time.Time) (bool, error) {
	if s == nil || s.r == nil {
		return true, nil
	}
	loc := time.FixedZone("Asia/Shanghai", 8*60*60)
	n := now.In(loc)
	key := fmt.Sprintf("signup_bonus_ip:%s:%s", n.Format("20060102"), ip)
	ttl := time.Date(n.Year(), n.Month(), n.Day()+1, 0, 0, 0, 0, loc).Sub(n)
	v, e := signupBonusIPScript.Run(ctx, s.r, []string{key}, ttl.Milliseconds()).Int64()
	return e == nil && v <= 2, e
}
