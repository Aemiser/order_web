package wuyin

import (
	"errors"
	"fmt"
	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/models/enum"
	alog "order_server/go/server/log"
	"strings"

	"gorm.io/gorm"
)

// DefaultTag 未匹配任何标签时的兜底标签名（不入库，展示与筛选时按需拼出）
const DefaultTag = "未分配"

// LoadTagConfigs 从独立库加载标签配置到缓存（启动与每次配置变更后调用）。
// 空表时播种默认标签（按吾饮良品群真实发送人整理，用户可在设置页改）。
func LoadTagConfigs() error {
	if global.DbWuyin == nil {
		return nil // 未配置库：缓存留空，端点入口另有 CheckDB 拦截
	}
	var list []models.WuyinTagConfig
	if err := global.DbWuyin.Order("sort").Order("id").Find(&list).Error; err != nil {
		return fmt.Errorf("加载吴饮标签配置失败: %w", err)
	}
	if len(list) == 0 {
		seed := []models.WuyinTagConfig{
			{Name: "合肥仓", Keywords: models.WuyinKeywordsJSON{"合肥"}, Sort: 1},
			{Name: "广西仓", Keywords: models.WuyinKeywordsJSON{"广西", "南昌"}, Sort: 2},
			{Name: "云南仓", Keywords: models.WuyinKeywordsJSON{"云南", "长春"}, Sort: 3},
			{Name: "成都仓", Keywords: models.WuyinKeywordsJSON{"成都", "河南"}, Sort: 4},
			{Name: "河北仓", Keywords: models.WuyinKeywordsJSON{"河北", "沈阳", "湖南"}, Sort: 5},
		}
		if err := global.DbWuyin.Create(&seed).Error; err != nil {
			return fmt.Errorf("播种默认标签失败: %w", err)
		}
		list = seed
	}
	global.WuyinTagConfigs = list
	return nil
}

// reloadTags 变更后重建缓存（加载失败不影响已落库的变更，只影响下次打标精度）
func reloadTags() {
	if err := LoadTagConfigs(); err != nil {
		fmt.Printf("重建吴饮标签缓存失败: %v\n", err)
	}
}

// MatchTag 按发送人昵称匹配标签：遍历缓存（已按 Sort、ID 升序），
// 任一关键词被发送人昵称包含即命中。未命中返回 DefaultTag。
func MatchTag(sender string) string {
	if sender == "" {
		return DefaultTag
	}
	for _, tag := range global.WuyinTagConfigs {
		for _, kw := range tag.Keywords {
			if kw != "" && strings.Contains(sender, kw) {
				return tag.Name
			}
		}
	}
	return DefaultTag
}

// applyTags 批量打标（识别结果后置处理；AI 已提取发送人，此处按配置裁决）
func applyTags(list []models.WuyinAddressInfo) []models.WuyinAddressInfo {
	for i := range list {
		list[i].Tag = MatchTag(list[i].Sender)
	}
	return list
}

// ListTagConfigs 标签列表（缓存即可，配置变更都经本包落库+重建）
func ListTagConfigs() []models.WuyinTagConfig {
	if global.WuyinTagConfigs == nil {
		return []models.WuyinTagConfig{}
	}
	return global.WuyinTagConfigs
}

// TagConfigRequest 标签增/改入参（Keywords 为空数组合法：仅人工指定的标签）
type TagConfigRequest struct {
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
	Sort     int      `json:"sort"`
}

func (r TagConfigRequest) validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return errors.New("标签名不能为空")
	}
	if len([]rune(r.Name)) > 64 {
		return errors.New("标签名过长（≤64 字符）")
	}
	if r.Name == DefaultTag {
		return errors.New("该名称为系统保留的兜底标签")
	}
	return nil
}

// CreateTagConfig 新增标签
func CreateTagConfig(req TagConfigRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if err := req.validate(); err != nil {
		return err
	}
	tag := models.WuyinTagConfig{
		Name:     req.Name,
		Keywords: models.WuyinKeywordsJSON(req.Keywords),
		Sort:     req.Sort,
	}
	if err := global.DbWuyin.Create(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "Duplicate entry") {
			return fmt.Errorf("标签 %s 已存在", req.Name)
		}
		return fmt.Errorf("创建标签失败: %w", err)
	}
	alog.Logaction(enum.LogInfoLevel, "吴饮标签新增", fmt.Sprintf("name=%s keywords=%v", req.Name, req.Keywords))
	reloadTags()
	return nil
}

// UpdateTagConfig 修改标签（名称与关键词整体替换）。
// 改名不回刷已落库订单的 tag 值（历史数据保持当时口径），仅影响后续打标。
func UpdateTagConfig(id uint, req TagConfigRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if err := req.validate(); err != nil {
		return err
	}
	var tag models.WuyinTagConfig
	if err := global.DbWuyin.First(&tag, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("标签不存在")
		}
		return fmt.Errorf("查询失败: %w", err)
	}
	updates := map[string]interface{}{
		"name":     req.Name,
		"keywords": models.WuyinKeywordsJSON(req.Keywords),
		"sort":     req.Sort,
	}
	if err := global.DbWuyin.Model(&tag).Updates(updates).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return fmt.Errorf("标签 %s 已存在", req.Name)
		}
		return fmt.Errorf("更新标签失败: %w", err)
	}
	alog.Logaction(enum.LogInfoLevel, "吴饮标签修改", fmt.Sprintf("id=%d name=%s keywords=%v", id, req.Name, req.Keywords))
	reloadTags()
	return nil
}

// DeleteTagConfig 删除标签。已打该标签的历史订单不受影响（值仍保留在行上）。
func DeleteTagConfig(id uint) error {
	var tag models.WuyinTagConfig
	if err := global.DbWuyin.First(&tag, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("标签不存在")
		}
		return fmt.Errorf("查询失败: %w", err)
	}
	if err := global.DbWuyin.Delete(&tag).Error; err != nil {
		return fmt.Errorf("删除标签失败: %w", err)
	}
	alog.Logaction(enum.LogInfoLevel, "吴饮标签删除", fmt.Sprintf("id=%d name=%s", id, tag.Name))
	reloadTags()
	return nil
}
