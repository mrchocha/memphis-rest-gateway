package main

import (
	"fmt"
	"rest-gateway/conf"
	"rest-gateway/logger"
	"rest-gateway/router"
	"time"

	"github.com/memphisdev/memphis.go"
)

func main() {
	configuration := conf.GetConfig()
	var conn *memphis.Conn
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			var err error
			opts := []memphis.Option{memphis.Reconnect(true), memphis.MaxReconnect(10), memphis.ReconnectInterval(3 * time.Second)}
			username := configuration.ROOT_USER
			if configuration.USER_PASS_BASED_AUTH {
				opts = append(opts, memphis.Password(configuration.CONNECTION_TOKEN))
			} else {
				opts = append(opts, memphis.ConnectionToken(configuration.CONNECTION_TOKEN))
			}
			if configuration.CLIENT_CERT_PATH != "" && configuration.CLIENT_KEY_PATH != "" && configuration.ROOT_CA_PATH != "" {
				opts = append(opts, memphis.Tls(configuration.CLIENT_CERT_PATH, configuration.CLIENT_KEY_PATH, configuration.ROOT_CA_PATH))
			}
			conn, err = memphis.Connect(configuration.MEMPHIS_HOST, username, opts...)
			if err == nil {
				ticker.Stop()
				goto serverInit
			} else {
				fmt.Printf("Awaiting to establish connection with Memphis - %v\n", err.Error())
			}
		}
	}

serverInit:
	l, err := logger.CreateLogger(configuration.MEMPHIS_HOST, configuration.ROOT_USER, configuration.CONNECTION_TOKEN)
	if err != nil {
		panic("Logger creation failed - " + err.Error())
	}

	app := router.SetupRoutes(conn, l)
	l.Noticef("Memphis REST gateway is up and running")
	l.Noticef("Version %s", configuration.VERSION)
	app.Listen(":" + configuration.HTTP_PORT)
}
