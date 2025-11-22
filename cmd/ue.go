package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/Alonza0314/free-ran-ue/logger"
	"github.com/Alonza0314/free-ran-ue/model"
	"github.com/Alonza0314/free-ran-ue/ue"
	"github.com/Alonza0314/free-ran-ue/util"
	loggergo "github.com/Alonza0314/logger-go/v2"
	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
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

	ueCmd.Flags().IntP("num", "n", 1, "number of UEs")
	rootCmd.AddCommand(ueCmd)
}

func ueFunc(cmd *cobra.Command, args []string) {
	if os.Geteuid() != 0 {
		loggergo.Error("UE", "This program requires root privileges to bring up tunnel device.")
		return
	}

	ueConfigFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		panic(err)
	}

	num, err := cmd.Flags().GetInt("num")
	if err != nil {
		panic(err)
	}

	ueConfig := model.UeConfig{}
	if err := util.LoadFromYaml(ueConfigFilePath, &ueConfig); err != nil {
		panic(err)
	}

	if err := util.ValidateUe(&ueConfig); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg, ues := sync.WaitGroup{}, make([]*ue.Ue, 0, num)

	defer func() {
		cancel()
		wg.Wait()

		for _, ue := range ues {
			ue.Stop()
		}
	}()

	baseMsinInt, err := strconv.Atoi(ueConfig.Ue.Msin)
	if err != nil {
		panic(err)
	}
	baseUeTunnelDevice := ueConfig.Ue.UeTunnelDevice

	for i := 0; i < num; i += 1 {
		updateUeConfig(&ueConfig, baseMsinInt, baseUeTunnelDevice, i)

		logger := logger.NewUeLogger(loggergoUtil.LogLevelString(ueConfig.Logger.Level), "", true)
		ue := ue.NewUe(&ueConfig, &logger)
		if ue == nil {
			return
		}

		if err := ue.Start(ctx, &wg); err != nil {
			return
		}

		ues = append(ues, ue)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}

func updateUeConfig(ueConfig *model.UeConfig, baseMsinInt int, baseUeTunnelDevice string, num int) {
	ueConfig.Ue.Msin = fmt.Sprintf("%010d", baseMsinInt+num)
	ueConfig.Ue.UeTunnelDevice = fmt.Sprintf("%s%d", baseUeTunnelDevice, num)
}
