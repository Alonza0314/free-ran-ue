package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Alonza0314/free-ran-ue/logger"
	"github.com/Alonza0314/free-ran-ue/model"
	"github.com/Alonza0314/free-ran-ue/ue"
	"github.com/Alonza0314/free-ran-ue/util"
	"github.com/spf13/cobra"
)

var ueCmd = &cobra.Command{
	Use:     "ue",
	Short:   "This is a UE simulator.",
	Long:    "This is a UE simulator for NR-DC feature in free5GC.",
	Example: "free-ran-ue ue",
	Run:     ueFunc,
}

func init() {
	ueCmd.Flags().StringP("config", "c", "config/ue.yaml", "config file path")
	if err := ueCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(ueCmd)
}

func ueFunc(cmd *cobra.Command, args []string) {
	ueConfigFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		panic(err)
	}

	ueConfig := model.UeConfig{}
	if err := util.LoadFromYaml(ueConfigFilePath, &ueConfig); err != nil {
		panic(err)
	}

	logger, err := logger.NewLogger(ueConfig.Logger.Level)
	if err != nil {
		panic(err)
	}

	ue := ue.NewUe(&ueConfig, logger)
	if ue == nil {
		return
	}

	if err := ue.Start(); err != nil {
		return
	}
	defer ue.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}
