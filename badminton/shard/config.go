package shard

import (
	"encoding/json"
	"os"
)

type Setting struct {
	Port    int               `json:"port"`
	Keyword string            `json:"keyword"`
	Lark    Lark              `json:"lark"`
	Wechat  Wechat            `json:"wechat"`
	Stash   map[string]string `json:"stash"`
	Drivers []string          `json:"drivers"`
}

type Lark struct {
	Webhook  string `json:"webhook"`
	Interval int    `json:"interval"`
}

type Wechat struct {
	Iyuu        map[string]string `json:"iyuu"`
	IyuuWebhook string            `json:"iyuuWebhook"`
}

var SettingInstance *Setting

func init() {
	SettingReload()
}

func SettingReload() {
	bytes, err := os.ReadFile("setting.json")

	if err != nil {
		return
	}

	SettingInstance = &Setting{}

	err = json.Unmarshal(bytes, &SettingInstance)
	if err != nil {
		return
	}
}

func SettingExport() string {
	b, _ := json.MarshalIndent(SettingInstance, "", "  ")

	return string(b)
}
