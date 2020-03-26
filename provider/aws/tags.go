package aws

import "sort"

type Tags map[string]string

func (tt Tags) CloudFormation() (interface{}, error) {
	type tag struct {
		Key   string `json:"Key"`
		Value string `json:"Value"`
	}
	list := make([]tag, 0, len(tt))
	for k, v := range tt {
		list = append(list, tag{Key: k, Value: v})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Key < list[j].Key
	})
	return list, nil
}
