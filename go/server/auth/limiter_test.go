package auth

import (
	"fmt"
	"testing"
	"time"
)

// newTestLimiter 固定参数：窗口 10 分钟、失败 3 次锁 5 分钟。
// 全部用例用注入时间轴驱动，不依赖真实等待。
func newTestLimiter() *LoginLimiter {
	return NewLoginLimiter(3, 10*time.Minute, 5*time.Minute)
}

func TestLimiterLocksAfterMaxFailures(t *testing.T) {
	l := newTestLimiter()
	now := time.Unix(1700000000, 0)

	// 前两次失败不锁定。
	if locked, _ := l.RecordFailure("k", now); locked {
		t.Fatalf("第 1 次失败即触发锁定")
	}
	if locked, _ := l.RecordFailure("k", now.Add(time.Second)); locked {
		t.Fatalf("第 2 次失败即触发锁定")
	}
	if l.Remaining("k", now.Add(2*time.Second)) != 1 {
		t.Fatalf("剩余次数应为 1")
	}

	// 第 3 次触发锁定。
	locked, retryAfter := l.RecordFailure("k", now.Add(2*time.Second))
	if !locked || retryAfter != 5*time.Minute {
		t.Fatalf("第 3 次失败应锁定 5 分钟，got locked=%v retryAfter=%v", locked, retryAfter)
	}

	// 锁定期内不再消耗剩余次数判断，直接拒绝。
	if ok, remaining := l.Locked("k", now.Add(3*time.Second)); !ok || remaining <= 0 {
		t.Fatalf("锁定期内 Locked 应为 true，got %v %v", ok, remaining)
	}
}

func TestLimiterUnlocksAfterDuration(t *testing.T) {
	l := newTestLimiter()
	now := time.Unix(1700000000, 0)

	lockAt := now
	for i := 0; i < 3; i++ {
		_, _ = l.RecordFailure("k", lockAt.Add(time.Duration(i)*time.Second))
	}
	lockAt = lockAt.Add(2 * time.Second) // 第 3 次失败的时刻即锁定起点

	// 锁定满 5 分钟即解除，条目就地清除（以 lockAt 为起点推算）。
	unlockAt := lockAt.Add(5 * time.Minute).Add(time.Second)
	if ok, _ := l.Locked("k", unlockAt); ok {
		t.Fatalf("锁定期过后应解除")
	}
	if l.Remaining("k", unlockAt) != 3 {
		t.Fatalf("解除后失败计数应清零，剩余应为 3")
	}
}

func TestLimiterResetOnSuccess(t *testing.T) {
	l := newTestLimiter()
	now := time.Unix(1700000000, 0)

	l.RecordFailure("k", now)
	l.RecordFailure("k", now.Add(time.Second))
	l.Reset("k")

	// 成功登录清零后，又要 3 次才锁。
	if locked, _ := l.RecordFailure("k", now.Add(2*time.Second)); locked {
		t.Fatalf("清零后第 1 次失败不应锁定")
	}
	l.RecordFailure("k", now.Add(3*time.Second))
	if locked, _ := l.RecordFailure("k", now.Add(4*time.Second)); !locked {
		t.Fatalf("清零后累计 3 次应重新锁定")
	}
}

func TestLimiterWindowResetsStaleCount(t *testing.T) {
	l := newTestLimiter()
	now := time.Unix(1700000000, 0)

	l.RecordFailure("k", now)
	l.RecordFailure("k", now.Add(time.Second))

	// 窗口（10 分钟）外的旧失败作废，重新起算。
	if locked, _ := l.RecordFailure("k", now.Add(11*time.Minute)); locked {
		t.Fatalf("窗口外的旧失败不应计入，单次新失败不该锁定")
	}
	if l.Remaining("k", now.Add(11*time.Minute)) != 2 {
		t.Fatalf("窗口重置后剩余应为 2，got %d", l.Remaining("k", now.Add(11*time.Minute)))
	}
}

// TestLimiterKeysAreIndependent 组合键「IP+用户名」彼此独立：
// 同一用户换来源 IP 不被连坐，同一来源换用户名也不被连坐。
func TestLimiterKeysAreIndependent(t *testing.T) {
	l := newTestLimiter()
	now := time.Unix(1700000000, 0)

	for i := 0; i < 3; i++ {
		l.RecordFailure("ip1|alice", now.Add(time.Duration(i)*time.Second))
	}

	if ok, _ := l.Locked("ip1|bob", now.Add(4*time.Second)); ok {
		t.Fatalf("同一 IP 换用户名不应被连坐")
	}
	if ok, _ := l.Locked("ip2|alice", now.Add(4*time.Second)); ok {
		t.Fatalf("同一用户换 IP 不应被连坐")
	}
	if ok, _ := l.Locked("ip1|alice", now.Add(4*time.Second)); !ok {
		t.Fatalf("原键应处于锁定期")
	}
}

// TestLimiterSweepBoundsMap 条目数超上限时清扫，使内存占用有界——
// 攻击者可用海量随机用户名把 map 撑爆，这是防线。
func TestLimiterSweepBoundsMap(t *testing.T) {
	l := NewLoginLimiter(100, time.Minute, time.Minute)
	now := time.Unix(1700000000, 0)

	// 灌满上限 + 10 条。
	for i := 0; i < l.maxTracked+10; i++ {
		l.RecordFailure(fmt.Sprintf("ip|user-%d", i), now.Add(time.Duration(i)*time.Millisecond))
	}

	// 一小时后再记一条失败：触发 sweep，此时全部旧条目都在窗口外，
	// 应被整体清掉，map 长度回落到个位数而非继续增长。
	l.RecordFailure("ip|post-sweep", now.Add(time.Hour))

	l.mu.Lock()
	size := len(l.records)
	l.mu.Unlock()
	if size > 2 {
		t.Fatalf("清扫后 map 应只剩新条目，仍剩 %d 条", size)
	}
}
