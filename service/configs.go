package service

import (
	"net/http"
	"strings"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
)

// configList 返回全部站点配置，按分组下发
func configList(c *adminCtx, r *http.Request) (interface{}, error) {
	list, err := dao.ListConfigs()
	if err != nil {
		return nil, serverError("读取配置失败: " + err.Error())
	}
	groups := make([]map[string]interface{}, 0)
	index := map[string]int{}
	for _, item := range list {
		if _, ok := index[item.GroupName]; !ok {
			index[item.GroupName] = len(groups)
			groups = append(groups, map[string]interface{}{
				"group": item.GroupName,
				"items": []model.SysConfig{},
			})
		}
		pos := index[item.GroupName]
		groups[pos]["items"] = append(groups[pos]["items"].([]model.SysConfig), item)
	}
	values, _ := dao.ConfigMap()
	return map[string]interface{}{"groups": groups, "values": values, "total": len(list)}, nil
}

// configSave 批量保存配置
func configSave(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Items []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"items"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if len(req.Items) == 0 {
		return nil, badRequest("没有需要保存的配置项")
	}

	// 先取现有配置的类型，用于规范化与防止写入未知 key
	known := map[string]string{}
	if list, err := dao.ListConfigs(); err == nil {
		for _, item := range list {
			known[item.Key] = item.Type
		}
	}

	saved, skipped := 0, 0
	for _, item := range req.Items {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		configType, ok := known[key]
		if !ok {
			// 未知配置项直接跳过，避免前端脏 key 建出无效记录
			skipped++
			continue
		}
		value := item.Value
		if configType == "switch" {
			// 开关值统一为 true / false，兼容 1、on、yes 等写法
			switch strings.ToLower(strings.TrimSpace(value)) {
			case "true", "1", "on", "yes":
				value = "true"
			default:
				value = "false"
			}
		}
		if err := dao.SaveConfig(key, value); err != nil {
			return nil, serverError("保存 " + key + " 失败: " + err.Error())
		}
		saved++
	}
	values, _ := dao.ConfigMap()
	return map[string]interface{}{"saved": saved, "skipped": skipped, "values": values}, nil
}
