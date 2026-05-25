package main

import (
	. "Topicgram/common"
	"Topicgram/config"
	"Topicgram/database"
	_ "Topicgram/i18n/languages"
	"Topicgram/pkg/proxy"
	"Topicgram/services/bots"
	"Topicgram/services/cron"
	_ "Topicgram/services/cron/jobs"
	"encoding/json"
	"flag"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/CoiaPrant/clog"
)

var (
	version = "dev"
)

func main() {
	var conf Config
	{
		var cfg string
		{
			flag.StringVar(&cfg, "config", "config.json", "The config file location")
			debug := flag.Bool("debug", false, "Show debug logs")
			help := flag.Bool("h", false, "Show help")
			v := flag.Bool("version", false, "Show version")
			flag.Parse()

			if *help {
				flag.PrintDefaults()
				return
			}

			if *v {
				println(version)
				return
			}

			if *debug {
				clog.SetLevel(clog.LevelDebug)
			}
		}

		file, err := os.ReadFile(cfg)
		if err != nil {
			clog.Fatal("[Config] Unable to read config file, error: ", err)
			return
		}

		err = json.Unmarshal(file, &conf)
		if err != nil {
			clog.Fatal("[Config] Unable to parse config file, error: ", err)
			return
		}
	}

	{
		TLSConfig.InsecureSkipVerify = conf.Security.InsecureSkipVerify
	}

	clog.Infof("Bot Version: %s", version)

	{
		if conf.Proxy != "" {
			u, err := url.Parse(conf.Proxy)
			if err != nil {
				clog.Fatal("[Initial] failed to parse proxy, error: ", err)
				return
			}

			err = proxy.Register(u)
			if err != nil {
				clog.Fatal("[Initial] failed to register proxy, error: ", err)
				return
			}
		}
	}

	{
		var dbConf config.Database

		switch conf.Database.Type {
		case "sqlite3":
			dbConf = conf.Database.SQLite3
		case "mysql":
			dbConf = conf.Database.MySQL
		case "postgres":
			dbConf = conf.Database.Postgres
		case "oracle":
			dbConf = conf.Database.Oracle
		default:
			clog.Fatal("[Config] Unknown database type")
			return
		}

		if dbConf == nil {
			clog.Fatal("[Config] Bad database config")
			return
		}

		err := database.InitDB(dbConf)
		if err != nil {
			clog.Fatal("[Database] failed to connect database, error: ", err)
			return
		}
		clog.Success("[Database] connected database")
	}

	{
		if conf.Bot == nil {
			clog.Fatal("[Bot] Invalid config")
			return
		}

		if conf.Bot.Token == "" {
			clog.Fatal("[Bot] Invalid Bot Token")
			return
		}

		if conf.Bot.GroupId == 0 {
			clog.Fatal("[Bot] Invalid Bot Group Id")
			return
		}
	}

	err := bots.Load(conf.Bot)
	if err != nil {
		clog.Fatal("[Bot][Initial] failed to init bot, error: ", err)
		return
	}
	cron.Start()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTRAP)

	<-sigs
	bots.Shutdown()
	clog.Message("Exiting")
}
