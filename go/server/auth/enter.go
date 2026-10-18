// Package auth 登录业务：账号查找、口令校验、失败限流与令牌签发。
//
// 与其余 server 包一样只经 application 门面被 handler 调用（B1/B3）。
// 账号注册不设入口——由本地 tools/userctl 脚本直连数据库写入，
// 因此本包没有创建/改密逻辑。
package auth

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/models/enum"
	alog "order_server/go/server/log"
	authutil "order_server/go/utils/auth"

	"github.com/sirupsen/logrus"
)

const (
	// loginMaxFailures 滑动窗口内允许的连续失败次数。
	loginMaxFailures = 5
	// loginWindow / loginLockDuration 失败计数窗口与触发后的锁定时长。
	loginWindow       = 15 * time.Minute
	loginLockDuration = 15 * time.Minute
)

var (
	// ErrBadCredentials 口令错误或账号不存在。刻意不区分二者：
	// 区分会把「哪些用户名有效」直接告诉攻击者。
	ErrBadCredentials = errors.New("账号或口令错误")
	// ErrDisabled 账号被禁用。仅在口令已验证正确后才可能返回。
	ErrDisabled = errors.New("账号已被禁用，请联系管理员")
)

// LockedError 登录失败次数过多。单独成类型是为了让 handler 拿到
// 剩余锁定时长去拼提示语，而 errors.Is 只能判不能读。
type LockedError struct {
	RetryAfter time.Duration
}

func (e *LockedError) Error() string {
	return fmt.Sprintf("失败次数过多，请 %s 后重试", formatDuration(e.RetryAfter))
}

// loginLimiter 包级单例。参数是固定值而非配置项：限流阈值属于
// 防爆破的基本盘，跟着配置走反而给了误调空间。
var loginLimiter = NewLoginLimiter(loginMaxFailures, loginWindow, loginLockDuration)

// dummyHash 用户不存在时仍要跑一次 bcrypt 比较用的哈希。
// bcrypt 单次约 60ms，若用户不存在就提前返回，「用户名是否存在」
// 会从响应耗时差里泄露；固定哈希把两条路径的耗时拉平。
var dummyHash = func() string {
	h, err := authutil.HashPassword("order-web-timing-equalizer")
	if err != nil {
		// 对固定短输入 GenerateFromPassword 不会失败；万一失败（熵源不可用），
		// 空哈希会让 VerifyPassword 立即返回 false，仅剩时序差异，不影响正确性。
		return ""
	}
	return h
}()

var (
	fallbackSecret     string
	fallbackSecretOnce sync.Once
)

// LoginResult 登录成功产物。
type LoginResult struct {
	Token     string
	ExpiresIn time.Duration
	User      *models.User
}

// Secret 返回生效的签名密钥。
//
// 正常路径下 main 启动时 settings.yaml 已填 auth.secret。兜底分支处理
// 密钥留空（或测试直接构造 Config 跳过配置文件）的情况：生成进程内
// 随机密钥并只生成一次——随机密钥不可预测，所以不存在「空密钥可伪造
// 令牌」的漏洞，代价只是重启后所有人需要重新登录。
func Secret() string {
	if global.Config != nil && global.Config.Auth.Secret != "" {
		return global.Config.Auth.Secret
	}
	fallbackSecretOnce.Do(func() {
		s, err := authutil.GenerateSecret()
		if err != nil {
			// 熵源不可用意味着无法签出安全令牌，带病运行不如直接终止。
			logrus.Fatalf("生成鉴权密钥失败: %v", err)
		}
		fallbackSecret = s
		logrus.Warn("auth.secret 未配置，已生成进程内随机密钥：重启后所有登录失效，请在 settings.yaml 配置固定密钥")
	})
	return fallbackSecret
}

// Issuer 返回生效的令牌签发者标识。
func Issuer() string {
	if global.Config != nil && global.Config.Auth.Issuer != "" {
		return global.Config.Auth.Issuer
	}
	return authutil.DefaultIssuer
}

// TTL 返回生效的令牌有效期。
func TTL() time.Duration {
	hours := authutil.DefaultExpireHours
	if global.Config != nil && global.Config.Auth.ExpireHours > 0 {
		hours = global.Config.Auth.ExpireHours
	}
	return time.Duration(hours) * time.Hour
}

// Login 校验口令并签发令牌。调用方（handler）已保证 username/password 非空。
//
// 校验顺序刻意为「口令 → 启用位」：先对口令再对启用位，禁用账号的错误口令
// 得到的是含糊的 ErrBadCredentials，不泄露「这个账号存在且被禁用」。
func Login(username, password, ip string) (*LoginResult, error) {
	key := ip + "|" + username
	now := time.Now()

	if locked, retryAfter := loginLimiter.Locked(key, now); locked {
		return nil, &LockedError{RetryAfter: retryAfter}
	}

	var user models.User
	err := global.Db.Where("username = ?", username).First(&user).Error
	if err != nil {
		// 用户不存在：仍执行一次 bcrypt 比较抹平耗时差（见 dummyHash 说明），
		// 记一次失败（键含用户名，攻击者换用户名不影响单一账号的计数）。
		authutil.VerifyPassword(dummyHash, password)
		loginLimiter.RecordFailure(key, now)
		return nil, ErrBadCredentials
	}

	if !authutil.VerifyPassword(user.Password, password) {
		loginLimiter.RecordFailure(key, now)
		remaining := loginLimiter.Remaining(key, now)
		if remaining > 0 && remaining < loginMaxFailures {
			// 剩余次数随错误一起返回，让正常用户在锁定前就知道还剩几次机会。
			return nil, fmt.Errorf("%w（再错 %d 次将锁定 %s）",
				ErrBadCredentials, remaining, formatDuration(loginLockDuration))
		}
		return nil, ErrBadCredentials
	}

	if !user.Enabled {
		alog.Logaction(enum.LogWarnLevel, "登录被拒",
			fmt.Sprintf("账号 %s 已被禁用，来源 %s", username, ip))
		return nil, ErrDisabled
	}

	ttl := TTL()
	token, err := authutil.Sign(Secret(), Issuer(), authutil.Identity{
		UserID:   user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
		Role:     user.Role,
	}, ttl)
	if err != nil {
		return nil, fmt.Errorf("签发令牌失败: %w", err)
	}

	loginLimiter.Reset(key)
	touchLogin(user.ID, ip)
	alog.Logaction(enum.LogInfoLevel, "登录成功",
		fmt.Sprintf("账号 %s（%s）自 %s 登录", user.Username, user.Role, ip))

	return &LoginResult{Token: token, ExpiresIn: ttl, User: &user}, nil
}

// FindEnabledByID 按主键取用户。中间件每个已认证请求都会调用：
// 令牌是无状态的，签发之后账号被禁用或删除无法从令牌本身感知，
// 用一次主键查询换取「禁用即时生效」。内部工具流量低，代价可忽略。
// 返回的 error 语义是「查不到或查库失败」，启用与否由调用方判断。
func FindEnabledByID(id uint) (*models.User, error) {
	var user models.User
	if err := global.Db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Logout 记审计日志。JWT 无服务端状态，退出在服务端只是留痕，
// 真正的「失效」由前端丢弃令牌完成；令牌到期前服务端无法单方面撤销，
// 这是无状态鉴权的固有取舍，有效期上限（TTL）就是为此设的。
func Logout(username, role, ip string) {
	alog.Logaction(enum.LogInfoLevel, "退出登录",
		fmt.Sprintf("账号 %s（%s）自 %s 退出", username, role, ip))
}

// touchLogin 更新最近登录时间与来源 IP。仅运维参考用途，
// 写失败只记日志不阻断登录。
func touchLogin(id uint, ip string) {
	err := global.Db.Model(&models.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_login_at": time.Now(),
		"last_login_ip": ip,
	}).Error
	if err != nil {
		logrus.Errorf("更新最近登录信息失败: %v", err)
	}
}

// formatDuration 把时长格式化成中文提示用的「X 分 Y 秒」形态。
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "稍候"
	}
	mins := int(d.Minutes())
	if mins >= 1 {
		return fmt.Sprintf("%d 分钟", mins)
	}
	return fmt.Sprintf("%d 秒", int(d.Seconds()))
}
