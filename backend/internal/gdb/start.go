package gdb

import (
	"log"
	"log/slog"
	"os"
	"time"

	sqlite "github.com/aceberg/gorm-sqlite"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/models"
)

var db *gorm.DB
var gormConf *gorm.Config

// Start working with DB
func Start() {
	var tab *gorm.DB
	var err error

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             5 * time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
	gormConf = &gorm.Config{
		Logger: newLogger,
		NamingStrategy: schema.NamingStrategy{
			NoLowerCase: true,
			// So upper case Columns could work in both PostgreSQL and SQLite
		},
	}

	Connect()

	// Migrate the schema
	tab = db.Table("now")
	err = tab.AutoMigrate(&models.Host{})
	check.IfError(err)

	tab = db.Table("history")
	err = tab.AutoMigrate(&models.Host{})
	check.IfError(err)

	tab = db.Table("subnets")
	err = tab.AutoMigrate(&models.Subnet{})
	check.IfError(err)

	tab = db.Table("ports")
	err = tab.AutoMigrate(&models.Port{})
	check.IfError(err)

	tab = db.Table("port_scans")
	err = tab.AutoMigrate(&models.PortScan{})
	check.IfError(err)

	tab = db.Table("map_pos")
	err = tab.AutoMigrate(&models.MapPos{})
	check.IfError(err)

	tab = db.Table("views")
	err = tab.AutoMigrate(&models.View{})
	check.IfError(err)

	tab = db.Table("items")
	err = tab.AutoMigrate(&models.Item{})
	check.IfError(err)

	tab = db.Table("icons")
	err = tab.AutoMigrate(&models.Icon{})
	check.IfError(err)

	tab = db.Table("host_days")
	err = tab.AutoMigrate(&models.HostDay{})
	check.IfError(err)

	tab = db.Table("subnet_hours")
	err = tab.AutoMigrate(&models.SubnetHour{})
	check.IfError(err)

	tab = db.Table("first_seen")
	err = tab.AutoMigrate(&models.FirstSeen{})
	check.IfError(err)

	tab = db.Table("connectors")
	err = tab.AutoMigrate(&models.Connector{})
	check.IfError(err)

	tab = db.Table("containers")
	err = tab.AutoMigrate(&models.Container{})
	check.IfError(err)

	tab = db.Table("host_categories")
	err = tab.AutoMigrate(&models.HostCategory{})
	check.IfError(err)

	tab = db.Table("bookmarks")
	err = tab.AutoMigrate(&models.Bookmark{})
	check.IfError(err)
	migrateLinks()

	tab = db.Table("guests")
	err = tab.AutoMigrate(&models.Guest{})
	check.IfError(err)

	TidyNames()
}

// Connect - choose DB and connect
func Connect() {
	var err error
	var pgFail bool

	if conf.AppConfig.UseDB == "postgres" {
		db, err = gorm.Open(postgres.Open(conf.AppConfig.PGConnect), gormConf)

		if err != nil {
			pgFail = true

			slog.Error("PostgreSQL connection error:", "err", err)
			slog.Warn("Falling back to SQLite")
		} else {
			slog.Info("Connected to DB: PostgreSQL")
		}
	}

	if pgFail || conf.AppConfig.UseDB != "postgres" {

		db, err = gorm.Open(sqlite.Open(conf.AppConfig.DBPath), gormConf)

		if !check.IfError(err) {
			slog.Info("Connected to DB: SQLite")
			// One connection: the scan, port scans and connector syncs write at
			// the same time, and SQLite allows one writer. Pragmas are per connection.
			if sqlDB, err := db.DB(); err == nil {
				sqlDB.SetMaxOpenConns(1)
			}
			db.Exec("PRAGMA journal_mode = wal;")
			db.Exec("PRAGMA busy_timeout = 5000;")
		}
	}
}
