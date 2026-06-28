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

var listenAddr string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the DDNS daemon and the internal HTTP API together",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := os.Getenv("HOVER_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("HOVER_API_KEY is not set; refusing to start")
		}
		if listenAddr == "" {
			listenAddr = os.Getenv("HOVER_LISTEN")
		}
		if listenAddr == "" {
			listenAddr = "127.0.0.1:8088"
		}

		cfg, err := hover.LoadConfig(cfgFile)
		if err != nil {
			return err
		}
		c, err := hover.NewClient(cfg, verbose)
		if err != nil {
			return err
		}

		ctx, stop := notifyContext()
		defer stop()

		var wg sync.WaitGroup
		var ddnsErr error
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := ddns.Run(ctx, c, cfg); err != nil {
				ddnsErr = err
				log.Printf("ddns stopped: %v", err)
				stop() // a fatal ddns error brings the whole process down
			}
		}()

		srv := api.NewServer(c, apiKey, listenAddr)
		srvErr := srv.Run(ctx)

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

func init() {
	serveCmd.Flags().StringVar(&listenAddr, "listen", "", "address to listen on (default 127.0.0.1:8088, env HOVER_LISTEN)")
	rootCmd.AddCommand(serveCmd)
}
