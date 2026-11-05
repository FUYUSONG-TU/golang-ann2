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
	saved := 0
	for _, item := range req.Items {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		if err := dao.SaveConfig(key, item.Value); err != nil {
			return nil, serverError("保存 " + key + " 失败: " + err.Error())
		}
		saved++
	}
	values, _ := dao.ConfigMap()
	return map[string]interface{}{"saved": saved, "values": values}, nil
}
