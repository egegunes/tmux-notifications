package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	dbusInterface     = "org.freedesktop.Notifications"
	dbusObjPath       = "/org/freedesktop/Notifications"
	dbusMember        = "Notify"
	dbusBecomeMonitor = "org.freedesktop.DBus.Monitoring.BecomeMonitor"
	dbusMatchSignal   = "type='signal',member='" + dbusMember + "',path='" + dbusObjPath + "',interface='" + dbusInterface + "'"
	dbusMatchCall     = "type='method_call',member='" + dbusMember + "',path='" + dbusObjPath + "',interface='" + dbusInterface + "'"
	dbusMatchReturn   = "type='method_return',member='" + dbusMember + "',path='" + dbusObjPath + "',interface='" + dbusInterface + "'"
	dbusMatchError    = "type='error',member='" + dbusMember + "',path='" + dbusObjPath + "',interface='" + dbusInterface + "'"
)

type notification struct {
	Sender  string
	Message string
}

func sendTmuxNotifications(nc chan notification) {
	for n := range nc {
		time.Sleep(5 * time.Second)
		cmd := exec.Command("tmux", "display-message", n.Sender+": "+n.Message)
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "Couldn't send notification to tmux:", err)
		}
	}
}

func handleMessage(nc chan notification, m *dbus.Message) {
	if len(m.Body) < 2 {
		return
	}

	var sender string
	var msg string

	switch m.Body[0] {
	case "Google Chrome":
		sender = m.Body[3].(string)
		body := strings.Split(m.Body[4].(string), "\n\n")
		msg = strings.ReplaceAll(body[1], "\n", "")
	case "Slack":
		sender = m.Body[3].(string)
		body := strings.Split(m.Body[4].(string), "\n")
		msg = strings.ReplaceAll(body[0], "\n", "")
	default:
		sender = m.Body[0].(string)
		msg = m.Body[3].(string)
	}

	nc <- notification{sender, msg}
}

func main() {
	conn, err := dbus.SessionBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to connect to session bus:", err)
		os.Exit(1)
	}

	var rules = []string{dbusMatchSignal, dbusMatchCall, dbusMatchReturn, dbusMatchError}
	var flag uint = 0
	call := conn.BusObject().Call(dbusBecomeMonitor, 0, rules, flag)
	if call.Err != nil {
		fmt.Fprintln(os.Stderr, "Failed to add match:", call.Err)
		os.Exit(1)
	}

	ch := make(chan *dbus.Message, 10)
	conn.Eavesdrop(ch)

	nc := make(chan notification)

	go sendTmuxNotifications(nc)

	for m := range ch {
		go handleMessage(nc, m)
	}
}
