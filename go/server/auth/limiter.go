package auth

import (
	"sync"
	"time"
)

// LoginLimiter 登录失败限流。
//
// 键是「IP + 用户名」的组合，而不是单独取其一：
//   - 只按 IP：同一 NAT 出口后一个人乱试会连累整间办公室；
//   - 只按用户名：攻击者轮换用户名即可绕过对单一账号的爆破防护。
//
// 组合键让两条路径各自独立计数。代价是攻击者可用大量不同用户名撑大 map，
// 故条目数超过 maxTracked 时主动清扫过期项（sweep），使内存占用有上界。
//
// 所有方法都显式接收 now，不内部调用 time.Now()——这样单测无需真实等待，
// 可以直接构造时间轴验证锁定与解除。
type LoginLimiter struct {
	mu           sync.Mutex
	records      map[string]*failureRecord
	maxFailures  int
	lockDuration time.Duration
	window       time.Duration
	maxTracked   int
}

type failureRecord struct {
	count    int
	firstAt  time.Time
	lockedAt time.Time // 零值表示尚未触发锁定
}

// NewLoginLimiter 构造限流器。window 是失败计数的滑动窗口：
// 窗口内累计 maxFailures 次失败即锁定 lockDuration，窗口外的旧失败不再计入。
func NewLoginLimiter(maxFailures int, window, lockDuration time.Duration) *LoginLimiter {
	if maxFailures <= 0 {
		maxFailures = 1
	}
	return &LoginLimiter{
		records:      make(map[string]*failureRecord),
		maxFailures:  maxFailures,
		window:       window,
		lockDuration: lockDuration,
		maxTracked:   4096,
	}
}

// Locked 报告该键是否仍在锁定期内，并返回剩余时长。
// 锁定期已过的条目就地删除，避免长期堆积。
func (l *LoginLimiter) Locked(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	rec, ok := l.records[key]
	if !ok || rec.lockedAt.IsZero() {
		return false, 0
	}
	if remaining := l.lockDuration - now.Sub(rec.lockedAt); remaining > 0 {
		return true, remaining
	}
	delete(l.records, key)
	return false, 0
}

// RecordFailure 记一次登录失败，返回是否已触发锁定及建议的重试等待时长。
func (l *LoginLimiter) RecordFailure(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.records) > l.maxTracked {
		l.sweep(now)
	}

	rec, ok := l.records[key]
	// 窗口外的旧计数作废，重新起算：否则用户几个月前的零星失败会永久累积。
	if !ok || now.Sub(rec.firstAt) > l.window {
		rec = &failureRecord{firstAt: now}
		l.records[key] = rec
	}
	rec.count++

	if rec.count >= l.maxFailures {
		rec.lockedAt = now
		return true, l.lockDuration
	}
	return false, 0
}

// Remaining 返回触发锁定前还剩几次尝试机会，用于给前端拼提示语。
func (l *LoginLimiter) Remaining(key string, now time.Time) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	rec, ok := l.records[key]
	if !ok || now.Sub(rec.firstAt) > l.window {
		return l.maxFailures
	}
	if r := l.maxFailures - rec.count; r > 0 {
		return r
	}
	return 0
}

// Reset 登录成功后清除该键的失败计数。
func (l *LoginLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.records, key)
}

// sweep 丢弃窗口外的计数与锁定期已过的条目，调用方必须持有锁。
func (l *LoginLimiter) sweep(now time.Time) {
	for k, rec := range l.records {
		if !rec.lockedAt.IsZero() {
			if now.Sub(rec.lockedAt) >= l.lockDuration {
				delete(l.records, k)
			}
			continue
		}
		if now.Sub(rec.firstAt) > l.window {
			delete(l.records, k)
		}
	}
}
