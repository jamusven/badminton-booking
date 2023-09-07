package misc

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type LarkText struct {
	Text string `json:"text"`
}

type LarkPost struct {
	MsgType string   `json:"msg_type"`
	Content LarkText `json:"content"`
}

func LarkMarkdown(msg string) {
	webhook := "https://open.feishu.cn/open-apis/bot/v2/hook/1" // sven
	//webhook := "https://open.feishu.cn/open-apis/bot/v2/hook/2" // 羽林军
	post := LarkPost{
		MsgType: "text",
		Content: LarkText{
			Text: "sven: " + msg,
		},
	}

	b, err := json.Marshal(post)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(webhook, "application/json", bytes.NewBuffer(b))
	if err != nil {
		panic(err)
	}

	//fmt.Printf("content: %s, resp: %v\n", post.Content.Text, resp)

	defer resp.Body.Close()
}
