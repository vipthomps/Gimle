package routines

import (
	"log/slog"

	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/subnet"
)

// SeedSubnets - when no subnets are configured, add the ones found on local interfaces
func SeedSubnets() {

	if len(gdb.SelectSubnets()) > 0 {
		return
	}

	for _, s := range subnet.Detect(conf.AppConfig.Ifaces) {
		err := gdb.SaveSubnet(&s)
		if !check.IfError(err) {
			slog.Info("Added detected subnet", "cidr", s.CIDR, "iface", s.Iface)
		}
	}
}
