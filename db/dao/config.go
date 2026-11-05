package dao

import (
	"time"

	"wxcloudrun-golang/db"
	"wxcloudrun-golang/db/model"
)

// ListConfigs 查询全部站点配置，按分组与主键排序
func ListConfigs() ([]model.SysConfig, error) {
	list := make([]model.SysConfig, 0)
	err := db.Get().Order("groupName asc, id asc").Find(&list).Error
	return list, err
}

// FindConfig 按配置键查询
func FindConfig(key string) (*model.SysConfig, error) {
	var item model.SysConfig
	err := db.Get().Where("`key` = ?", key).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// SaveConfig 写入或更新单条配置
func SaveConfig(key, value string) error {
	now := time.Now()
	var item model.SysConfig
	err := db.Get().Where("`key` = ?", key).First(&item).Error
	if err != nil && !IsNotFound(err) {
		return err
	}
	if IsNotFound(err) {
		return db.Get().Create(&model.SysConfig{
			Key: key, Value: value, GroupName: "custom", Type: "text",
			UpdatedAt: now,
		}).Error
	}
	return db.Get().Model(&model.SysConfig{}).Where("id = ?", item.Id).
		Updates(map[string]interface{}{"value": value, "updatedAt": now}).Error
}

// ConfigMap 把配置转成 map，便于一次性下发前端
func ConfigMap() (map[string]string, error) {
	list, err := ListConfigs()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(list))
	for _, item := range list {
		out[item.Key] = item.Value
	}
	return out, nil
}
