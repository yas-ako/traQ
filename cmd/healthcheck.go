package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// healthcheckCommand ヘルスチェックコマンド
func healthcheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "healthcheck",
		Short: "Run healthcheck",
		Run: func(_ *cobra.Command, _ []string) {
			logger, _ := getCLILoggers()
			defer logger.Sync()

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(fmt.Sprintf("http://localhost:%d/api/ping", c.Port))
			if err != nil {
				logger.Fatal("HTTP Client Error", zap.Error(err))
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				logger.Fatal("Unexpected status", zap.Int("status", resp.StatusCode))
			}
		},
	}
}
