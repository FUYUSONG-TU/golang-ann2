package dao

import (
	"wxcloudrun-golang/db"
	"wxcloudrun-golang/db/model"
)

// ListCounters 查询全部计数记录，按 id 升序返回（后台管理页使用）
func (imp *CounterInterfaceImp) ListCounters() ([]model.CounterModel, error) {
	cli := db.Get()
	var list []model.CounterModel
	err := cli.Table(tableName).Order("id asc").Find(&list).Error
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []model.CounterModel{}
	}
	return list, nil
}
