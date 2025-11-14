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
	"Topicgram/services/webhook"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"gitlab.com/CoiaPrant/clog"
)

var (
	version = "dev"
)

func main() {
	var conf Config
	{
		var config_file string
		{
			flag.StringVar(&config_file, "config", "config.json", "The config file location")
			flag.BoolVar(clog.DebugFlag(), "debug", false, "Show debug logs")
			help := flag.Bool("h", false, "Show help")
			v := flag.Bool("version", false, "Show version")
			flag.Parse()

			if *help {
				flag.PrintDefaults()
				return
			}

			if *v {
				clog.Print(version)
				return
			}
		}

		file, err := os.ReadFile(config_file)
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

		gin.SetMode(gin.ReleaseMode)
		if version == "dev" || clog.IsDebug() {
			gin.SetMode(gin.DebugMode)
		}
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

		if conf.Bot.WebHook.Host == "" {
			clog.Fatal("[Bot] Invalid WebHook Host")
			return
		}

		err := bots.Load(conf.Bot)
		if err != nil {
			clog.Fatal("[Bot][Initial] failed to init bot, error: ", err)
			return
		}
	}

	cron.Start()

	var srv http.Server
	{
		if conf.Web.Type == "unix" {
			os.Remove(conf.Web.Listen)
		}

		lis, err := net.Listen(conf.Web.Type, conf.Web.Listen)
		if err != nil {
			clog.Fatal("[Web] failed to listen, error: ", err)
			return
		}

		if conf.Web.Type == "unix" {
			os.Chmod(conf.Web.Listen, 0777)
		}

		srv = http.Server{Handler: webhook.Handler(), ErrorLog: log.New(io.Discard, "", 0)}

		if conf.Web.Cert == "" || conf.Web.Key == "" {
			go srv.Serve(lis)
		} else {
			{
				_, err = tls.LoadX509KeyPair(conf.Web.Cert, conf.Web.Key)
				if err != nil {
					clog.Fatal("[Web] failed to load tls certificate, error: ", err)
					return
				}
			}
			go srv.ServeTLS(lis, conf.Web.Cert, conf.Web.Key)
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTRAP)

	<-sigs
	srv.Shutdown(context.Background())
	clog.Print("Exiting")
}
