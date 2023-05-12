package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/adamwg/memphis-chat/api"
	"github.com/jroimartin/gocui"
	"github.com/memphisdev/memphis.go"
	"google.golang.org/protobuf/proto"
)

func main() {
	var (
		username = flag.String("username", "", "username for chat")
	)
	flag.Parse()
	if username == nil || *username == "" {
		log.Fatal("must provide username with the -username flag")
	}

	gui, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Fatalf("creating gui: %v", err)
	}
	defer gui.Close()
	gui.Cursor = true
	gui.Mouse = true
	gui.SetManager(
		gocui.ManagerFunc(channelView),
		gocui.ManagerFunc(messageView),
		gocui.ManagerFunc(editView),
	)

	gui.Update(func(g *gocui.Gui) error {
		v, err := gui.View("channels")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(v, "#memphis\n#general\n#offtopic\n")
		return err
	})

	if err := gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Fatalf("setting up quit: %v", err)
	}

	conn, err := memphis.Connect(os.Getenv("MEMPHIS_ADDR"), os.Getenv("MEMPHIS_USER"), memphis.Password(os.Getenv("MEMPHIS_PASSWORD")))
	if err != nil {
		log.Fatalf("creating memphis connection: %v", err)
	}
	defer conn.Close()

	producerName := fmt.Sprintf("chat-%s", *username)
	producer, err := conn.CreateProducer("awg-test-station", producerName)
	if err != nil {
		log.Fatalf("creating producer %q: %v", producerName, err)
	}

	sendMessage := func(g *gocui.Gui, v *gocui.View) error {
		channelsView, err := g.View("channels")
		if err != nil {
			return err
		}
		channelName := getSelection(channelsView)

		var msg api.ChatMessage
		input := v.Buffer()
		v.Clear()
		v.SetCursor(0, 0)

		msg.From = *username
		msg.Channel = channelName
		msg.Body = input

		return producer.Produce(&msg)
	}

	showChannel := func(g *gocui.Gui, v *gocui.View) error {
		channelName := getSelection(v)
		viewName := fmt.Sprintf("messages-%s", channelName)
		maxX, maxY := g.Size()
		_, err := g.SetView(viewName, 20, 0, maxX-1, maxY-5)
		if err != nil && err != gocui.ErrUnknownView {
			return err
		}

		_, err = g.SetViewOnTop(viewName)
		return err
	}

	if err := gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, sendMessage); err != nil {
		log.Fatalf("setting up quit: %v", err)
	}
	if err := gui.SetKeybinding("channels", gocui.MouseLeft, gocui.ModNone, showChannel); err != nil {
		log.Fatalf("setting up click: %v", err)
	}

	consumer, err := conn.CreateConsumer(
		"awg-test-station",
		producerName,
		memphis.PullInterval(100*time.Millisecond),
	)
	if err != nil {
		log.Fatalf("creating consumer %q: %v", producerName, err)
	}

	handler := func(msgs []*memphis.Msg, err error, ctx context.Context) {
		if err != nil {
			return
		}

		for _, rawMsg := range msgs {
			var msg api.ChatMessage
			if err := proto.Unmarshal(rawMsg.Data(), &msg); err != nil {
				log.Printf("invalid message: %v", err)
				continue
			}
			if err := rawMsg.Ack(); err != nil {
				log.Printf("failed to ack: %v", err)
				continue
			}
			gui.Update(func(g *gocui.Gui) error {
				channelsView, err := g.View("channels")
				if err != nil {
					return err
				}
				selectedChannel := getSelection(channelsView)

				maxX, maxY := g.Size()
				viewName := fmt.Sprintf("messages-%s", msg.Channel)
				v, err := g.SetView(viewName, 20, 0, maxX-1, maxY-5)
				if err != nil && err != gocui.ErrUnknownView {
					return err
				}
				if selectedChannel != msg.Channel {
					_, err = g.SetViewOnBottom(viewName)
					if err != nil {
						return err
					}
				}
				_, err = fmt.Fprintf(v, "<%s>\t%s", msg.From, msg.Body)
				return err
			})
		}
	}

	go consumer.Consume(handler)

	if err := gui.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatalf("main loop: %v", err)
	}
}

func channelView(g *gocui.Gui) error {
	_, maxY := g.Size()
	v, err := g.SetView("channels", 0, 0, 19, maxY-1)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}

	v.Title = "Channels"
	v.Highlight = true
	v.SelBgColor = gocui.ColorGreen
	v.SelFgColor = gocui.ColorBlack

	return nil
}

func messageView(g *gocui.Gui) error {
	channelsView, err := g.View("channels")
	if err != nil {
		return err
	}
	channelName := getSelection(channelsView)

	maxX, maxY := g.Size()
	_, err = g.SetView(fmt.Sprintf("messages-%s", channelName), 20, 0, maxX-1, maxY-5)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	return nil
}

func editView(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	v, err := g.SetView("edit", 20, maxY-4, maxX-1, maxY-1)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	v.Editable = true
	g.SetCurrentView("edit")
	return nil
}

func getSelection(v *gocui.View) string {
	var (
		l   string
		err error
	)
	_, cy := v.Cursor()
	if l, err = v.Line(cy); err != nil {
		l = ""
	}
	return l
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
