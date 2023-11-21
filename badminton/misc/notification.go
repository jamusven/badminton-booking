package misc

import (
	"badminton-booking/badminton/shard"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type LarkText struct {
	Text string `json:"text"`
}

type LarkPost struct {
	MsgType string   `json:"msg_type"`
	Content LarkText `json:"content"`
}

func LarkMarkdown(msg string) {
	post := LarkPost{
		MsgType: "text",
		Content: LarkText{
			Text: shard.SettingInstance.Keyword + " : " + msg,
		},
	}

	b, err := json.Marshal(post)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(shard.SettingInstance.Lark.Webhook, "application/json", bytes.NewBuffer(b))
	if err != nil {
		panic(err)
	}

	//fmt.Printf("content: %s, resp: %v\n", post.Content.Text, resp)

	defer resp.Body.Close()
}

func Wechat(msg string) {
	if len(shard.SettingInstance.Wechat.Iyuu) <= 0 {
		return
	}

	msg = url.QueryEscape(msg)

	for _, token := range shard.SettingInstance.Wechat.Iyuu {
		webhook := fmt.Sprintf(shard.SettingInstance.Wechat.IyuuWebhook, token, msg)

		_, err := http.Get(webhook)

		if err != nil {
			continue
		}
	}
}

func WechatSingle(name, msg string) {
	if len(shard.SettingInstance.Wechat.Iyuu) <= 0 {
		return
	}

	var token string
	var ok bool

	if token, ok = shard.SettingInstance.Wechat.Iyuu[name]; !ok {
		return
	}

	msg = url.QueryEscape(msg)

	webhook := fmt.Sprintf(shard.SettingInstance.Wechat.IyuuWebhook, token, msg)

	resp, err := http.Get(webhook)

	if err != nil {
		return
	}

	defer resp.Body.Close()
}
