package cmd

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/bttnns/hover-dns/internal/api"
	"github.com/bttnns/hover-dns/internal/ddns"
	"github.com/bttnns/hover-dns/internal/hover"
	"github.com/spf13/cobra"
)

var (
	listenAddr string
	apiFlag    bool
	ddnsFlag   bool
)

const defaultListen = "127.0.0.1:8088"

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the daemon: the DDNS loop and/or the internal HTTP API",
	Long: "Run the long-running daemon. Which services start is driven by the\n" +
		"config file (ddns.enabled, api.enabled), each defaulting to off. The\n" +
		"--ddns and --api flags override the config per-invocation.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := hover.LoadConfig(cfgFile)
		if err != nil {
			return err
		}

		// Resolve each service: config is the baseline; an explicitly-passed flag
		// wins. cmd.Flags().Changed distinguishes "not set" from "set to false".
		ddnsEnabled := cfg.DDNS.Enabled
		if cmd.Flags().Changed("ddns") {
			ddnsEnabled = ddnsFlag
		}
		apiEnabled := cfg.API.Enabled
		if cmd.Flags().Changed("api") {
			apiEnabled = apiFlag
		}

		if !ddnsEnabled && !apiEnabled {
			return fmt.Errorf("no services enabled: set ddns.enabled and/or api.enabled in config (or pass --ddns/--api)")
		}

		var apiKey, addr string
		if apiEnabled {
			apiKey = os.Getenv("HOVER_API_KEY")
			if apiKey == "" {
				return fmt.Errorf("api enabled but HOVER_API_KEY is not set; refusing to start")
			}
			addr = resolveListen(cfg)
		}

		c, err := hover.NewClient(cfg, verbose)
		if err != nil {
			return err
		}

		ctx, stop := notifyContext()
		defer stop()

		var wg sync.WaitGroup
		var ddnsErr error
		if ddnsEnabled {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := ddns.Run(ctx, c, cfg); err != nil {
					ddnsErr = err
					log.Printf("ddns stopped: %v", err)
					stop() // a fatal ddns error brings the whole process down
				}
			}()
		}

		var srvErr error
		if apiEnabled {
			srv := api.NewServer(c, apiKey, addr)
			srvErr = srv.Run(ctx)
		} else {
			// API-less: block on the ddns goroutine until ctx is cancelled.
			<-ctx.Done()
		}

		// Cancel ctx so the ddns goroutine unblocks even when the API server
		// returned a startup error before any signal arrived, then wait for it.
		stop()
		wg.Wait()

		if srvErr != nil {
			return srvErr
		}
		return ddnsErr
	},
}

// resolveListen picks the API listen address: the --listen flag wins, then the
// HOVER_LISTEN env var, then config api.listen, then the built-in default. Env is
// above config so a deployment can override the file-pinned address.
func resolveListen(cfg *hover.Config) string {
	if listenAddr != "" {
		return listenAddr
	}
	if env := os.Getenv("HOVER_LISTEN"); env != "" {
		return env
	}
	if cfg.API.Listen != "" {
		return cfg.API.Listen
	}
	return defaultListen
}

func init() {
	serveCmd.Flags().StringVar(&listenAddr, "listen", "", "API listen address (overrides config api.listen, env HOVER_LISTEN; default "+defaultListen+")")
	serveCmd.Flags().BoolVar(&apiFlag, "api", false, "run the HTTP API (overrides config api.enabled)")
	serveCmd.Flags().BoolVar(&ddnsFlag, "ddns", false, "run the DDNS loop (overrides config ddns.enabled)")
	rootCmd.AddCommand(serveCmd)
}
