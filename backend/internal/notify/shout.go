package notify

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/nicholas-fedor/shoutrrr"

	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/models"
)

// Unknown - send message to log and shoutrrr
func Unknown(host models.Host) {

	msg := fmt.Sprintf("Unknown host found. Name: '%s', IP: '%s', MAC: '%s', Hw: '%s', Iface: '%s'", host.DNS, host.IP, host.Mac, host.Hw, host.Iface)

	slog.Warn(msg)
	shout(msg)
}

// Test Shoutrrr notification
func Test() {

	msg := "test notification"
	slog.Info("Sending " + msg)
	shout(msg)
}

// shout - send msg to Shoutrrr
func shout(msg string) {

	hostname, _ := os.Hostname()
	wyl := "Gimlé on '" + hostname + "': "

	if conf.AppConfig.ShoutURL != "" {
		err := shoutrrr.Send(conf.AppConfig.ShoutURL, wyl+msg)
		if err != nil {
			slog.Error("Notification failed (shoutrrr): ", "", err)
		}
	}
}

// Ports - report ports that opened or closed on a host
func Ports(host models.Host, opened, closed []int) {

	msg := fmt.Sprintf("Ports changed on '%s' (%s). Opened: %v. Closed: %v", host.Name, host.IP, opened, closed)

	slog.Warn(msg)
	shout(msg)
}
