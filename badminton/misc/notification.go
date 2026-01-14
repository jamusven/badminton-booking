package misc

import (
	"badminton-booking/badminton/shard"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type LarkText struct {
	Text string `json:"text"`
}

type LarkPost struct {
	MsgType string   `json:"msg_type"`
	Content LarkText `json:"content"`
}

var larkChan = make(chan string, 100)

func init() {
	go larkWorker()
}

func larkWorker() {
	ticker := time.NewTicker(time.Duration(shard.SettingInstance.Lark.Interval) * time.Second)
	defer ticker.Stop()

	var larkMessage []string

loop:
	for {
		select {
		case val, ok := <-larkChan:
			if !ok {
				break loop
			}

			larkMessage = append(larkMessage, val)
		case <-ticker.C:
			time.Sleep(5 * time.Second)
			break loop
		}
	}

	if len(larkMessage) > 0 {
		LarkMarkdown(strings.Join(larkMessage, "\n"))
	}

	larkWorker()
}

func LarkMarkdownChan(msg string) {
	if shard.SettingInstance.Lark.Webhook == "" {
		return
	}

	larkChan <- msg
}

func LarkMarkdown(msg string) {
	if shard.SettingInstance.Lark.Webhook == "" {
		return
	}

	post := LarkPost{
		MsgType: "text",
		Content: LarkText{
			Text: shard.SettingInstance.Keyword + msg,
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

func LarkAlert(names, title, text string) {
	if shard.SettingInstance.Lark.AlertWebhook == "" {
		return
	}

	Http(shard.SettingInstance.Lark.AlertWebhook, []byte(ToJsonPrettify(map[string]string{
		"names": names,
		"title": title,
		"text":  text,
	})))
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

func Http(url string, data []byte) {
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}

	//fmt.Printf("content: %s, resp: %v\n", post.Content.Text, resp)

	defer resp.Body.Close()
}
