package commands

import (
	"log"
	"net/url"

	"github.com/JSainsburyPLC/ui-dev-proxy/domain"
	"github.com/JSainsburyPLC/ui-dev-proxy/proxy"
	"github.com/urfave/cli"
)

func StartCommand(logger *log.Logger, confProvider domain.ConfigProvider) cli.Command {
	return cli.Command{
		Name:  "start",
		Usage: "Start the proxy",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "default-backend-url, u",
				Usage:    "the default backend to use",
				Required: true,
			},
			cli.StringFlag{
				Name:     "config, c",
				Usage:    "Load configuration from 'FILE'",
				Required: true,
			},
			cli.IntFlag{
				Name:  "port, p",
				Usage: "The port to start proxy on",
				Value: 8080,
			},
			cli.BoolFlag{
				Name:  "enable-mocks, m",
				Usage: "Turn on mocks",
			},
			cli.BoolFlag{
				Name:  "tls-enabled",
				Usage: "Turn on TLS (tls-certfile and tls-keyfile both required if this is true)",
			},
			cli.StringFlag{
				Name:  "tls-certfile",
				Usage: "Path to TLS certificate file",
			},
			cli.StringFlag{
				Name:  "tls-keyfile",
				Usage: "Path to TLS key file",
			},
			cli.BoolFlag{
				Name:  "tls-skip-verify",
				Usage: "Disable TLS certificate verification",
			},
		},
		Action: startAction(logger, confProvider),
	}
}

func startAction(logger *log.Logger, confProvider domain.ConfigProvider) cli.ActionFunc {
	return func(c *cli.Context) error {
		logger.Println("Starting UI Dev Proxy...")

		defaultBackendUrl := c.String("default-backend-url")
		confFile := c.String("config")
		port := c.Int("port")
		mocksEnabled := c.Bool("enable-mocks")
		tlsEnabled := c.Bool("tls-enabled")
		tlsCertfile := c.String("tls-certfile")
		tlsKeyfile := c.String("tls-keyfile")
		tlsSkipVerify := c.Bool("tls-skip-verify")

		logger.Printf("Default backend URL: %s\n", defaultBackendUrl)
		logger.Printf("Config file: %s\n", confFile)
		logger.Printf("Port: %d\n", port)
		logger.Printf("Mocks enabled: %t\n", mocksEnabled)
		logger.Printf("TLS enabled: %t\n", tlsEnabled)
		if tlsEnabled {
			logger.Printf("TLS certfile: %s\n", tlsCertfile)
			logger.Printf("TLS keyfile: %s\n", tlsKeyfile)
		}

		conf, err := confProvider(confFile)
		if err != nil {
			return cli.NewExitError(err, 1)
		}

		defaultBackend, err := url.Parse(defaultBackendUrl)
		if err != nil {
			return cli.NewExitError(err, 1)
		}

		p := proxy.NewProxy(port, conf, defaultBackend, mocksEnabled, logger, tlsSkipVerify)

		if tlsEnabled {
			p.TlsEnabled = true
			p.TlsCertFile = tlsCertfile
			p.TlsKeyFile = tlsKeyfile
		}

		p.Start()

		return nil
	}
}
