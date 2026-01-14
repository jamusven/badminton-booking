package shard

import (
	"encoding/json"
	"os"
)

type Setting struct {
	Port            int                     `json:"port"`
	Keyword         string                  `json:"keyword"`
	Lark            Lark                    `json:"lark"`
	Wechat          Wechat                  `json:"wechat"`
	Stash           map[string]string       `json:"stash"`
	Alias           map[string]string       `json:"alias"`
	VenueBookingMap map[string]VenueBooking `json:"venueBookingMap"`
}

type Lark struct {
	Webhook       string `json:"webhook"`
	RecordWebhook string `json:"recordWebhook"`
	AlertWebhook  string `json:"alertWebhook"`
	Interval      int    `json:"interval"`
}

type Wechat struct {
	Iyuu        map[string]string `json:"iyuu"`
	IyuuWebhook string            `json:"iyuuWebhook"`
}

type VenueBooking struct {
	Name     string `json:"name"`
	Desc     string `json:"desc"`
	Amount   int32  `json:"amount"`
	Limit    int32  `json:"limit"`
	VenueFee int32  `json:"venueFee"`
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
