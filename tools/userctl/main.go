// userctl 是本地可交互式的用户管理工具，直连数据库管理登录账号。
//
// 账号注册不对外开放（前端/后端均无注册入口），管理员在本地用本脚本
// 向服务器数据库插入、启停、删除或重置用户。密码统一走 bcrypt 哈希，
// 与后端 server/auth 的校验逻辑一致。
//
// 用法：
//   go run ./tools/userctl          # 读取默认配置 ./go/config/settings.yaml
//   go run ./tools/userctl -f path  # 指定配置文件
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"order_server/go/conf"
	"order_server/go/global"
	"order_server/go/models"
	authutil "order_server/go/utils/auth"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// 与后端一致的配置读取：-f 默认 ./go/config/settings.yaml
	file := flag.String("f", "./go/config/settings.yaml", "配置文件路径")
	flag.Parse()

	config := readConfig(*file)
	global.Config = config
	global.Db = connectDB(config)

	// 确保 users 表存在；不存在时先建表再继续（生产库由 go run main.go -db 负责迁移）
	if err := global.Db.AutoMigrate(&models.User{}); err != nil {
		fmt.Println("users 表检查/创建失败:", err)
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		printMenu()
		line := readLine(reader, "")
		switch strings.TrimSpace(line) {
		case "1":
			createUser(reader)
		case "2":
			listUsers()
		case "3":
			setEnabled(reader, true)
		case "4":
			setEnabled(reader, false)
		case "5":
			deleteUser(reader)
		case "6":
			resetPassword(reader)
		case "0", "q", "Q":
			fmt.Println("退出。")
			return
		default:
			fmt.Println("无效选择，请重新输入。")
		}
	}
}

func readConfig(file string) *conf.Config {
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Printf("读取配置文件失败 %s: %v\n", file, err)
		os.Exit(1)
	}
	c := new(conf.Config)
	if err := yamlUnmarshal(data, c); err != nil {
		fmt.Printf("配置文件格式错误: %v\n", err)
		os.Exit(1)
	}
	return c
}

// yamlUnmarshal 直接使用 yaml.v3 解析配置。
func yamlUnmarshal(data []byte, c *conf.Config) error {
	return yaml.Unmarshal(data, c)
}

func connectDB(c *conf.Config) *gorm.DB {
	if len(c.DB) == 0 {
		fmt.Println("配置文件中没有 db 配置")
		os.Exit(1)
	}
	dc := c.DB[0]
	db, err := gorm.Open(mysql.Open(dc.GetDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Printf("数据库连接失败: %v\n", err)
		os.Exit(1)
	}
	return db
}

func printMenu() {
	fmt.Println()
	fmt.Println("========== 用户管理 (userctl) ==========")
	fmt.Println(" 1. 创建用户")
	fmt.Println(" 2. 列出用户")
	fmt.Println(" 3. 启用用户")
	fmt.Println(" 4. 禁用用户")
	fmt.Println(" 5. 删除用户(硬删)")
	fmt.Println(" 6. 重置密码")
	fmt.Println(" 0. 退出")
	fmt.Print("请选择: ")
}

// readLine 读取一行；trim 为 true 时去掉首尾空白。
func readLine(reader *bufio.Reader, prompt string) string {
	if prompt != "" {
		fmt.Print(prompt)
	}
	line, err := reader.ReadString('\n')
	if err != nil && err.Error() == "EOF" {
		// 管道/EOF 输入也返回已读部分
		return strings.TrimSpace(line)
	}
	if err != nil {
		return ""
	}
	return strings.TrimSpace(line)
}

// readRequired 读取非空输入；为空时反复提示。
func readRequired(reader *bufio.Reader, prompt string) string {
	for {
		v := readLine(reader, prompt)
		if v != "" {
			return v
		}
		fmt.Println("输入不能为空，请重新输入。")
	}
}

func createUser(reader *bufio.Reader) {
	fmt.Println()
	fmt.Println("-- 创建用户 --")
	username := readRequired(reader, "用户名: ")
	if len(username) > 64 {
		fmt.Println("用户名长度不能超过 64。")
		return
	}

	password := readRequired(reader, "密码: ")
	if len(password) > authutil.MaxPasswordLen {
		fmt.Printf("密码长度不能超过 %d。\n", authutil.MaxPasswordLen)
		return
	}

	nickname := readLine(reader, "昵称(回车默认同用户名): ")
	if nickname == "" {
		nickname = username
	}

	fmt.Printf("角色 (1=admin, 2=operator, 回车默认 operator): ")
	roleInput := readLine(reader, "")
	role := models.RoleOperator
	if strings.TrimSpace(roleInput) == "1" {
		role = models.RoleAdmin
	}

	hash, err := authutil.HashPassword(password)
	if err != nil {
		fmt.Printf("密码哈希失败: %v\n", err)
		return
	}

	user := models.User{
		Username: username,
		Password: hash,
		Nickname: nickname,
		Role:     role,
		Enabled:  true,
	}
	if err := global.Db.Create(&user).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "1062") {
			fmt.Printf("创建失败：用户名 %q 已存在。\n", username)
		} else {
			fmt.Printf("创建失败: %v\n", err)
		}
		return
	}
	fmt.Printf("用户 %q 创建成功 (昵称=%s, 角色=%s)。\n", username, nickname, role)
}

func listUsers() {
	fmt.Println()
	fmt.Println("-- 用户列表 --")
	var users []models.User
	if err := global.Db.Order("id asc").Find(&users).Error; err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}
	if len(users) == 0 {
		fmt.Println("(空)")
		return
	}
	fmt.Printf("%-5s %-20s %-20s %-10s %-8s %-22s %s\n", "ID", "用户名", "昵称", "角色", "状态", "最近登录", "最近IP")
	for _, u := range users {
		status := "启用"
		if !u.Enabled {
			status = "禁用"
		}
		last := "-"
		if u.LastLoginAt != nil {
			last = u.LastLoginAt.Format("2006-01-02 15:04:05")
		}
		fmt.Printf("%-5d %-20s %-20s %-10s %-8s %-22s %s\n",
			u.ID, u.Username, u.Nickname, u.Role, status, last, u.LastLoginIP)
	}
}

// promptUserID 交互式选择用户 ID；列出候选后要求输入数字。
func promptUserID(reader *bufio.Reader, action string) (uint, bool) {
	fmt.Println()
	listUsers()
	idStr := readLine(reader, fmt.Sprintf("输入要%s的用户ID (回车取消): ", action))
	if idStr == "" {
		fmt.Println("已取消。")
		return 0, false
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		fmt.Println("无效的 ID，已取消。")
		return 0, false
	}
	return uint(id), true
}

func setEnabled(reader *bufio.Reader, enabled bool) {
	verb := "禁用"
	if enabled {
		verb = "启用"
	}
	id, ok := promptUserID(reader, verb)
	if !ok {
		return
	}
	result := global.Db.Model(&models.User{}).Where("id = ?", id).Update("enabled", enabled)
	if result.Error != nil {
		fmt.Printf("操作失败: %v\n", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		fmt.Printf("未找到 ID=%d 的用户。\n", id)
		return
	}
	fmt.Printf("ID=%d 用户已%s。\n", id, verb)
}

func deleteUser(reader *bufio.Reader) {
	id, ok := promptUserID(reader, "删除")
	if !ok {
		return
	}
	fmt.Printf("确认硬删除 ID=%d 的用户？此操作不可恢复。输入 yes 确认: ", id)
	confirm := readLine(reader, "")
	if confirm != "yes" {
		fmt.Println("已取消。")
		return
	}
	// 硬删除（Unscoped）：释放 username 唯一索引，允许之后重建同名账号
	if err := global.Db.Unscoped().Delete(&models.User{}, id).Error; err != nil {
		fmt.Printf("删除失败: %v\n", err)
		return
	}
	fmt.Printf("ID=%d 用户已删除。\n", id)
}

func resetPassword(reader *bufio.Reader) {
	id, ok := promptUserID(reader, "重置密码")
	if !ok {
		return
	}
	password := readRequired(reader, "新密码: ")
	if len(password) > authutil.MaxPasswordLen {
		fmt.Printf("密码长度不能超过 %d。\n", authutil.MaxPasswordLen)
		return
	}
	hash, err := authutil.HashPassword(password)
	if err != nil {
		fmt.Printf("密码哈希失败: %v\n", err)
		return
	}
	result := global.Db.Model(&models.User{}).Where("id = ?", id).Update("password", hash)
	if result.Error != nil {
		fmt.Printf("重置失败: %v\n", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		fmt.Printf("未找到 ID=%d 的用户。\n", id)
		return
	}
	fmt.Printf("ID=%d 用户密码已重置。\n", id)
}
