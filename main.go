package main

import (
	"context"
	"fmt"
	"linkding-telegram/internal/config"
	"linkding-telegram/internal/linkding"
	"linkding-telegram/internal/telegram"
	"os"

	"github.com/sirupsen/logrus"
)

const (
	MsgLinkAdded = "✅ Link successfully added! 🎉"

	ErrInvalidURL = "❌ That doesn't look like a valid URL. Double-check and try again! 🔍"
)

var log = logrus.New()

func init() {
	log.SetOutput(os.Stdout)

	log.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	log.SetLevel(logrus.InfoLevel)
}

func setupLogger(logLevel, logPath string) error {
	if logPath != "" {
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open/create log file: %w", err)
		}

		log.SetOutput(file)
	}

	switch logLevel {
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	case "fatal":
		log.SetLevel(logrus.FatalLevel)
	case "panic":
		log.SetLevel(logrus.PanicLevel)
	}

	return nil
}

func main() {
	config, err := config.GetConfig()
	if err != nil {
		log.Fatalf("failed to read configuration file: %s", err.Error())
	}

	if err := setupLogger(config.LogLevel, config.LogFile); err != nil {
		log.Fatalf("failed to setup logger: %s", err.Error())
	}

	lnkdng := linkding.New(config, log)
	tg := telegram.New(config, lnkdng, log)

	ctx := context.Background()

	go tg.PollUpdates(ctx)

	for update := range tg.GetUpdate() {
		var tags []string
		if update.Message.MessageOrigin.Chat.Username != "" {
			tags = append(tags, update.Message.MessageOrigin.Chat.Username)
		}

		if config.LinkdingConf.DefaultTag != "" {
			tags = append(tags, config.LinkdingConf.DefaultTag)
		}

		var req *linkding.CreateBookmarkReqBody
		if len(tags) > 0 {
			req = &linkding.CreateBookmarkReqBody{
				URL:      update.Message.LinkPrev.Url,
				Unread:   true,
				TagNames: tags,
			}
		} else {
			req = &linkding.CreateBookmarkReqBody{
				URL:    update.Message.LinkPrev.Url,
				Unread: true,
			}
		}

		_, error := lnkdng.CreateBookmark(ctx, req)
		if error != nil {
			log.Error(error)
			message := fmt.Sprintf("%s\nError: %s", ErrInvalidURL, error)
			tg.SendMessage(ctx, update.Message.Chat.ID, message)
		} else {
			log.Info("Link added successfully")
			tg.SendMessage(ctx, update.Message.Chat.ID, MsgLinkAdded)
		}

	}
}
