package main

import (
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/puzpuzpuz/xsync/v4"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
)

var db = func() *badger.DB {
	dbInner, err := badger.Open(badger.DefaultOptions("./marsbot"))
	if err != nil {
		log.Fatal(err)
	}
	return dbInner
}()

func main() {
	// Get token from the environment variable
	token := os.Getenv("BOT_TOKEN")
	apiUrl := os.Getenv("BOT_API_URL")
	if token == "" {
		log.Fatal("BOT_TOKEN environment variable not set")
	}
	if apiUrl == "" {
		apiUrl = gotgbot.DefaultAPIURL
	}
	// Create bot from environment value.
	b, err := gotgbot.NewBot(token, &gotgbot.BotOpts{
		BotClient: &gotgbot.BaseBotClient{
			Client: http.Client{},
			DefaultRequestOpts: &gotgbot.RequestOpts{
				Timeout: gotgbot.DefaultTimeout, // Customise the default request timeout here
				APIURL:  apiUrl,                 // As well as the Default API URL here (in case of using local bot API servers)
			},
		},
	})
	if err != nil {
		panic("failed to create new bot: " + err.Error())
	}

	// Create updater and dispatcher.
	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		// If an error is returned by a handler, log it and continue going.
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Println("an error occurred while handling update:", err.Error())
			return ext.DispatcherActionNoop
		},
		Panic: func(b *gotgbot.Bot, ctx *ext.Context, r interface{}) {
			log.Println("a panic occurred while handling update:", r)
			fmt.Println("Stack trace:")
			fmt.Printf("%s\n", debug.Stack()) // 打印堆栈信息
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	})
	updater := ext.NewUpdater(dispatcher, nil)
	dispatcher.AddHandler(handlers.NewCommand("add_whitelist", AddImgToWhiteList))
	dispatcher.AddHandler(handlers.NewCommand("remove_whitelist", RemoveImgFromWhiteList))
	dispatcher.AddHandler(handlers.NewCommand("show_info", ShowImageInfo))
	dispatcher.AddHandler(handlers.NewMessage(isOnePhotoMessage, ReplyToOneImgDupHash))
	dispatcher.AddHandler(handlers.NewMessage(isGroupPhotoMessage, ReplyToGroupedImgDup))

	// Start receiving updates.
	err = updater.StartPolling(b, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 9,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 10,
			},
		},
	})
	if err != nil {
		panic("failed to start polling: " + err.Error())
	}
	log.Printf("%s has been started...\n", b.User.Username)

	// Idle, to keep updates coming in, and avoid bot stopping.
	updater.Idle()
}

var downloadingImage = xsync.NewMap[string, *sync.WaitGroup]()

func getImgPHash(b *gotgbot.Bot, msg *gotgbot.Message) (h PHash, err error) {
	fileObj := msg.Photo[len(msg.Photo)-1]
retry:
	h, err = GetDHashByUniqueId(fileObj.FileUniqueId)
	if err == nil {
		// 这里是找到了
		return h, nil
	}
	if !errors.Is(err, badger.ErrKeyNotFound) {
		// 没找到，而且还不是Key未找到的错误，那就是奇怪的错误，不管
		return h, err
	}
	wg, loaded := downloadingImage.LoadOrCompute(fileObj.FileUniqueId,
		func() (newValue *sync.WaitGroup, cancel bool) {
			wgInner := &sync.WaitGroup{}
			wgInner.Add(1)
			return wgInner, false
		})
	if loaded {
		log.Printf("same file is downloading, waiting for result (*fileData.Id:%s)", fileObj.FileUniqueId)
		wg.Wait()
		downloadingImage.Delete(fileObj.FileUniqueId)
		goto retry
	}
	defer downloadingImage.Delete(fileObj.FileUniqueId)
	defer wg.Done()
	file, err := b.GetFile(fileObj.FileId, nil)
	if err != nil {
		return
	}
	u := file.URL(b, nil)
	log.Printf("get bot http url %s", u)
	defer func() {
		if err != nil {
			return
		}
		e := SetDHashByUniqueId(fileObj.FileUniqueId, h)
		if e != nil {
			log.Printf("set dHash error: %s", e)
		}
	}()
	if strings.HasPrefix(u, "http") {
		resp, err := http.Get(u)
		if err != nil {
			return h, err
		}
		defer resp.Body.Close()
		f, err := os.CreateTemp(os.TempDir(), "dhash-img*")
		if err != nil {
			return h, err
		}
		defer os.Remove(f.Name())
		_, err = io.Copy(f, resp.Body)
		_ = f.Close()
		return PHashFromFile(f.Name())
	} else if strings.HasPrefix(u, "file") {
		u1, err := url.Parse(u)
		if err != nil {
			return h, err
		}
		return PHashFromFile(u1.Path)
	}
	return h, err
}

func ShowImageInfo(b *gotgbot.Bot, ctx *ext.Context) error {
	log.Println("show image info")
	msg := ctx.Message
	if msg.Photo == nil && msg.ReplyToMessage != nil {
		msg = msg.ReplyToMessage
	}
	if msg.Photo == nil {
		_, err := ctx.Message.Reply(b, "bot 没有识别到任何图片", nil)
		return err
	}
	h, err := getImgPHash(b, msg)
	if err != nil {
		return err
	}
	photo := msg.Photo[len(msg.Photo)-1]
	_, err = b.SendMessage(
		msg.Chat.Id, fmt.Sprintf("Unique Id: `%s`\npHash: `%s`", photo.FileUniqueId, h), &gotgbot.SendMessageOpts{
			ParseMode: "Markdown",
		},
	)
	return err
}

func isOnePhotoMessage(msg *gotgbot.Message) bool {
	return len(msg.Photo) != 0 && msg.MediaGroupId == ""
}

func isGroupPhotoMessage(msg *gotgbot.Message) bool {
	return len(msg.Photo) != 0 && msg.MediaGroupId != ""
}

func ReplyToOneImgDupHash(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.Message
	dHash, err := getImgPHash(b, msg)
	if err != nil {
		return err
	}
	key := MarsInfoKey{
		ChatId: msg.Chat.Id,
		Hash:   dHash,
	}
	mu := GetLockByKey(msg.Chat.Id)
	mu.Lock()
	defer mu.Unlock()
	var value MarsInfo
	err = db.Update(func(txn *badger.Txn) error {
		info, err := key.GetInfo(txn)
		if err != nil {
			return err
		}
		value = info
		info.Count++
		err = key.SetInfo(txn, info)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !value.IsIgnored && value.Count != 0 {
		link := GetMsgLink(ctx.Message)
		text := GetSingleMarsText(link, value.Count, 3)
		_, err = ctx.Message.Reply(b, text, nil)
	}
	return err
}

type groupPhotoKey struct {
	ChatId int64
	Key    string
}

type marsInfoKeyWithMsg struct {
	MarsInfoKey
	msg *gotgbot.Message
}

var groupMedia = xsync.NewMap[groupPhotoKey, chan marsInfoKeyWithMsg]()

func ReplyToGroupedImgDup(b *gotgbot.Bot, ctx *ext.Context) error {

	groupKey := groupPhotoKey{
		ChatId: ctx.Message.Chat.Id,
		Key:    ctx.Message.MediaGroupId,
	}

	ctxChan, loaded := groupMedia.LoadOrCompute(groupKey, func() (newValue chan marsInfoKeyWithMsg, cancel bool) {
		return make(chan marsInfoKeyWithMsg, 10), false
	})

	if loaded {
		dh, err := getImgPHash(b, ctx.Message)
		if err != nil {
			return err
		}
		pm := marsInfoKeyWithMsg{
			MarsInfoKey: MarsInfoKey{
				ChatId: ctx.Message.Chat.Id,
				Hash:   dh,
			},
			msg: ctx.Message,
		}
		ctxChan <- pm
		return nil
	}
	keyList := make([]marsInfoKeyWithMsg, 0, 10)
	dh, err := getImgPHash(b, ctx.Message)
	if err == nil {
		keyList = append(keyList, marsInfoKeyWithMsg{
			MarsInfoKey: MarsInfoKey{
				ChatId: ctx.Message.Chat.Id,
				Hash:   dh,
			},
			msg: ctx.Message})
	}
process:
	for {
		t := time.NewTimer(2 * time.Second)
		select {
		case newCtx := <-ctxChan:
			keyList = append(keyList, newCtx)
			t.Stop() // 这里只有2s，对gc压力不大，而且go1.23以后也修了，纯粹是低版本go的编码习惯
		case <-t.C:
			break process
		}
	}
	groupMedia.Delete(groupKey)
	mu := GetLockByKey(groupKey.ChatId)
	mu.Lock()
	defer mu.Unlock()
	hasMarsPhoto := false
	maxCount := int64(0)
	var replyToMsg *gotgbot.Message
	e := db.Update(func(txn *badger.Txn) error {
		var valueList = make([]MarsInfo, 0, len(keyList))
		for _, k := range keyList {
			value, err := k.GetInfo(txn)
			if err != nil {
				return err
			}
			valueList = append(valueList, value)
			if !value.IsIgnored && value.Count > 0 {
				if value.Count >= maxCount {
					maxCount = value.Count
					replyToMsg = k.msg
				}
				hasMarsPhoto = true
			}
		}
		for i, value := range valueList {
			err := keyList[i].SetInfo(txn, value)
			if err != nil {
				log.Printf("update key:%s failed, error:%s", keyList[i], err)
			}
		}
		return nil
	})
	if e != nil {
		return e
	}
	if !hasMarsPhoto {
		return nil
	}
	link := GetMsgLink(replyToMsg)
	text := GetGroupedMarsText(link, maxCount, 3)
	_, err = b.SendMessage(ctx.Message.Chat.Id, text, &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{MessageId: replyToMsg.MessageId, ChatId: ctx.Message.Chat.Id},
	})
	if err != nil {
		log.Printf("send message failed, error:%s", err)
	}
	return nil
}
