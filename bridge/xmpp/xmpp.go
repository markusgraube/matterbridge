package bxmpp

import (
	"github.com/42wim/matterbridge/bridge/config"
	log "github.com/Sirupsen/logrus"
	"github.com/mattn/go-xmpp"

	"strings"
	"time"
)

type Bxmpp struct {
	xc       *xmpp.Client
	xmppMap  map[string]string
	Config   *config.Protocol
	origin   string
	protocol string
	Remote   chan config.Message
}

var flog *log.Entry
var protocol = "xmpp"

func init() {
	flog = log.WithFields(log.Fields{"module": protocol})
}

func New(config config.Protocol, origin string, c chan config.Message) *Bxmpp {
	b := &Bxmpp{}
	b.xmppMap = make(map[string]string)
	b.Config = &config
	b.protocol = protocol
	b.origin = origin
	b.Remote = c
	return b
}

func (b *Bxmpp) Connect() error {
	var err error
	flog.Infof("Connecting %s", b.Config.Server)
	b.xc, err = b.createXMPP()
	if err != nil {
		flog.Debugf("%#v", err)
		return err
	}
	flog.Info("Connection succeeded")
	go b.handleXmpp()
	return nil
}

func (b *Bxmpp) FullOrigin() string {
	return b.protocol + "." + b.origin
}

func (b *Bxmpp) JoinChannel(channel string) error {
	b.xc.JoinMUCNoHistory(channel+"@"+b.Config.Muc, b.Config.Nick)
	return nil
}

func (b *Bxmpp) Name() string {
	return b.protocol + "." + b.origin
}

func (b *Bxmpp) Protocol() string {
	return b.protocol
}

func (b *Bxmpp) Origin() string {
	return b.origin
}

func (b *Bxmpp) Send(msg config.Message) error {
	flog.Debugf("Receiving %#v", msg)
	b.xc.Send(xmpp.Chat{Type: "groupchat", Remote: msg.Channel + "@" + b.Config.Muc, Text: msg.Username + msg.Text})
	return nil
}

func (b *Bxmpp) createXMPP() (*xmpp.Client, error) {
	options := xmpp.Options{
		Host:     b.Config.Server,
		User:     b.Config.Jid,
		Password: b.Config.Password,
		NoTLS:    true,
		StartTLS: true,
		//StartTLS:      false,
		Debug:                        true,
		Session:                      true,
		Status:                       "",
		StatusMessage:                "",
		Resource:                     "",
		InsecureAllowUnencryptedAuth: false,
		//InsecureAllowUnencryptedAuth: true,
	}
	var err error
	b.xc, err = options.NewClient()
	return b.xc, err
}

func (b *Bxmpp) xmppKeepAlive() {
	go func() {
		ticker := time.NewTicker(90 * time.Second)
		for {
			select {
			case <-ticker.C:
				b.xc.Send(xmpp.Chat{})
			}
		}
	}()
}

func (b *Bxmpp) handleXmpp() error {
	for {
		m, err := b.xc.Recv()
		if err != nil {
			return err
		}
		switch v := m.(type) {
		case xmpp.Chat:
			var channel, nick string
			if v.Type == "groupchat" {
				s := strings.Split(v.Remote, "@")
				if len(s) == 2 {
					channel = s[0]
				}
				s = strings.Split(s[1], "/")
				if len(s) == 2 {
					nick = s[1]
				}
				if nick != b.Config.Nick {
					flog.Debugf("Sending message from %s on %s to gateway", nick, b.FullOrigin())
					b.Remote <- config.Message{Username: nick, Text: v.Text, Channel: channel, Origin: b.origin, Protocol: b.protocol, FullOrigin: b.FullOrigin()}
				}
			}
		case xmpp.Presence:
			// do nothing
		}
	}
}
