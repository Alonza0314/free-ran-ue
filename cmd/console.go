package cmd

import (
	"os"
	"os/signal"
	"syscall"

	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
	"github.com/free-ran-ue/free-ran-ue/console/backend"
	"github.com/free-ran-ue/free-ran-ue/logger"
	"github.com/free-ran-ue/free-ran-ue/model"
	"github.com/free-ran-ue/free-ran-ue/util"
	"github.com/spf13/cobra"
)

var consoleCmd = &cobra.Command{
	Use:     "console",
	Short:   "This is a console for free-ran-ue.",
	Long:    "This is a console for free-ran-ue. It is used to manage the free-ran-ue.",
	Example: "free-ran-ue console",
	Run:     consoleFunc,
}

func init() {
	consoleCmd.Flags().StringP("config", "c", "config/console.yaml", "config file path")
	if err := consoleCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(consoleCmd)
}

func consoleFunc(cmd *cobra.Command, args []string) {
	consoleConfigFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		panic(err)
	}

	consoleConfig := model.ConsoleConfig{}
	if err := util.LoadFromYaml(consoleConfigFilePath, &consoleConfig); err != nil {
		panic(err)
	}

	logger := logger.NewConsoleLogger(loggergoUtil.LogLevelString(consoleConfig.Logger.Level), "", true)

	console := backend.NewConsole(&consoleConfig, &logger)
	if console == nil {
		return
	}

	console.Start()
	defer console.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}
