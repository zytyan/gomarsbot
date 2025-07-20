package main

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

func GetMsgLink(msg *gotgbot.Message) string {

	if msg.Chat.Username != "" && msg.Chat.Type != "private" {
		return fmt.Sprintf("https://t.me/%s/%d", msg.Chat.Username, msg.MessageId)
	}
	if msg.Chat.Id > -1000000000000 {
		return ""
	}
	return fmt.Sprintf("https://t.me/c/%d/%d", -(msg.Chat.Id + 1000000000000), msg.Chat.Id)
}

func GetHTMLTag(link string) (string, string) {
	if link == "" {
		return "", ""
	}
	return fmt.Sprintf(`<a href="%s">`, link), "</a>"
}

func GetSingleMarsText(link string, count int64, threshold int64) string {
	start, end := GetHTMLTag(link)
	switch {
	case count < threshold:
		return fmt.Sprintf("这张图火星了%s火星%d次%s了！", start, count, end)
	case count == threshold:
		return fmt.Sprintf("这张图已经%s火星了%d次%s了，现在本车送你 ”火星之王“ 称号！", start, count, end)
	default:
		return fmt.Sprintf("火星之王，收了你的神通吧，这张图都让您%s火星%d次%s了！", start, count, end)
	}
}

func GetGroupedMarsText(link string, count int64, threshold int64) string {
	start, end := GetHTMLTag(link)
	switch {
	case count < threshold:
		return fmt.Sprintf("这一组图片火星了%s火星%d次%s了！", start, count, end)
	case count == threshold:
		return fmt.Sprintf("您这一组图片已经%s火星了%d次%s了，现在本车送你 ”火星之王“ 称号！", start, count, end)
	default:
		return fmt.Sprintf("火星之王，收了你的神通吧，这些图都让您%s火星%d次%s了！", start, count, end)
	}
}
