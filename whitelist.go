package main

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/dgraph-io/badger/v4"
	"log"
)

func AddImgToWhiteList(b *gotgbot.Bot, ctx *ext.Context) error {
	log.Println("Adding whitelist")
	msg := ctx.Message
	if msg.Photo == nil && msg.ReplyToMessage != nil {
		msg = msg.ReplyToMessage
	}
	if msg.Photo == nil {
		_, err := ctx.Message.Reply(b, "bot 没有识别到任何图片", nil)
		return err
	}
	mu := GetLockByKey(msg.Chat.Id)
	mu.Lock()
	mu.Unlock()
	h, err := getImgPHash(b, msg)
	if err != nil {
		return err
	}
	key := MarsInfoKey{ChatId: msg.Chat.Id, Hash: h}
	oldIsIgnored := true
	err = db.Update(func(txn *badger.Txn) error {
		info, err := key.GetInfo(txn)
		if err != nil {
			return err
		}
		oldIsIgnored = info.IsIgnored
		info.IsIgnored = true
		return key.SetInfo(txn, info)
	})
	if err != nil {
		log.Println(err)
		_, err = ctx.Message.Reply(b, fmt.Sprintf("出现错误%s", err), nil)
		return err
	}
	if oldIsIgnored {
		_, err = ctx.Message.Reply(b, "这张图片已经在白名单中了", nil)
		return err
	}
	_, err = ctx.Message.Reply(b, "添加白名单成功", nil)
	return err
}

func RemoveImgFromWhiteList(b *gotgbot.Bot, ctx *ext.Context) error {
	log.Println("Removing whitelist")
	msg := ctx.Message
	if msg.Photo == nil && msg.ReplyToMessage != nil {
		msg = msg.ReplyToMessage
	}
	if msg.Photo == nil {
		_, err := ctx.Message.Reply(b, "bot 没有识别到任何图片", nil)
		return err
	}
	mu := GetLockByKey(msg.Chat.Id)
	mu.Lock()
	mu.Unlock()
	h, err := getImgPHash(b, msg)
	if err != nil {
		return err
	}
	key := MarsInfoKey{ChatId: msg.Chat.Id, Hash: h}
	oldIsIgnored := true
	err = db.Update(func(txn *badger.Txn) error {
		info, err := key.GetInfo(txn)
		if err != nil {
			return err
		}
		oldIsIgnored = info.IsIgnored
		info.IsIgnored = false
		return key.SetInfo(txn, info)
	})
	if err != nil {
		_, err = ctx.Message.Reply(b, fmt.Sprintf("出现错误%s", err), nil)
		return err
	}
	if !oldIsIgnored {
		_, err = ctx.Message.Reply(b, "不在白名单中", nil)
		return err
	}
	_, err = ctx.Message.Reply(b, "移除白名单成功", nil)
	return err
}
