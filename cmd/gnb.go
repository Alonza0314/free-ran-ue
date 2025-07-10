package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Alonza0314/free-ran-ue/gnb"
	"github.com/Alonza0314/free-ran-ue/logger"
	"github.com/Alonza0314/free-ran-ue/model"
	"github.com/Alonza0314/free-ran-ue/util"
	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
	"github.com/spf13/cobra"
)

var gnbCmd = &cobra.Command{
	Use:     "gnb",
	Short:   "This is a gNB simulator.",
	Long:    "This is a gNB simulator for NR-DC feature in free5GC.",
	Example: "free-ran-ue gnb",
	Run:     gnbFunc,
}

func init() {
	gnbCmd.Flags().StringP("config", "c", "config/gnb.yaml", "config file path")
	if err := gnbCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(gnbCmd)
}

func gnbFunc(cmd *cobra.Command, args []string) {
	gnbConfigFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		panic(err)
	}

	gnbConfig := model.GnbConfig{}
	if err := util.LoadFromYaml(gnbConfigFilePath, &gnbConfig); err != nil {
		panic(err)
	}

	logger := logger.NewGnbLogger(loggergoUtil.LogLevelString(gnbConfig.Logger.Level), "", true)

	gnb := gnb.NewGnb(&gnbConfig, &logger)
	if gnb == nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err := gnb.Start(ctx); err != nil {
		return
	}
	defer gnb.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	cancel()
}
