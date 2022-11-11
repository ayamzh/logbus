package fluentd

import (
	"github.com/fluent/fluent-logger-golang/fluent"

	"bitbucket.org/funplus/sandwich/base/slog"
)

var client *Client

type Client struct {
	*fluent.Fluent
}

func Close() {
	if client == nil {
		return
	}
	slog.LogErrorAndEatError(client.Close(), "fluent_client_close")
}

func Init(c *fluent.Config) {
	f, err := fluent.New(*c)
	if err != nil {
		panic(err)
	}
	client = &Client{
		f,
	}
	return
}

func GetClient() *Client {
	if client == nil {
		panic("fluentd client not Init")
	}
	return client
}
