package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	loggergo "github.com/Alonza0314/logger-go/v2"
	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
	"github.com/free-ran-ue/free-ran-ue/constant"
	"github.com/free-ran-ue/free-ran-ue/logger"
	"github.com/free-ran-ue/free-ran-ue/model"
	"github.com/free-ran-ue/free-ran-ue/ue"
	"github.com/free-ran-ue/free-ran-ue/util"
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

	ueCmd.Flags().IntP("num", "n", constant.BASIC_UE_NUM, "number of UEs")
	ueCmd.Flags().IntP("concurrent", "p", constant.BASIC_UE_MAX_CONCURRENT, "max concurrent UEs to start simultaneously")
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

	maxConcurrent, err := cmd.Flags().GetInt("concurrent")
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
	wg, startStopWg, ues, uesMtx, errChan, semaphore := sync.WaitGroup{}, sync.WaitGroup{}, make([]*ue.Ue, 0, num), sync.Mutex{}, make(chan error, num), util.NewSemaphore(maxConcurrent)

	defer func() {
		cancel()
		wg.Wait()

		for _, u := range ues {
			startStopWg.Add(1)
			go func(ueInstance *ue.Ue) {
				defer startStopWg.Done()

				semaphore.Acquire()
				defer func() { semaphore.Release() }()

				ueInstance.Stop()
			}(u)
		}
		startStopWg.Wait()
	}()

	baseMsinInt, err := strconv.Atoi(ueConfig.Ue.Msin)
	if err != nil {
		panic(err)
	}
	baseUeTunnelDevice := ueConfig.Ue.UeTunnelDevice

	for i := 0; i < num; i += 1 {
		startStopWg.Add(1)
		go func(index int) {
			defer startStopWg.Done()

			semaphore.Acquire()
			defer func() { semaphore.Release() }()

			ueConfigCopy := ueConfig
			updateUeConfig(&ueConfigCopy, baseMsinInt, baseUeTunnelDevice, index)

			logger := logger.NewUeLogger(loggergoUtil.LogLevelString(ueConfigCopy.Logger.Level), "", true)
			ue := ue.NewUe(&ueConfigCopy, &logger)
			if ue == nil {
				errChan <- fmt.Errorf("error creating UE %d", index)
				return
			}

			if err := ue.Start(ctx, &wg); err != nil {
				errChan <- fmt.Errorf("error starting UE %d: %v", index, err)
				return
			}

			uesMtx.Lock()
			ues = append(ues, ue)
			uesMtx.Unlock()
		}(i)
	}

	startStopWg.Wait()
	close(errChan)
	for err := range errChan {
		loggergo.Error("UE", err.Error())
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}

func updateUeConfig(ueConfig *model.UeConfig, baseMsinInt int, baseUeTunnelDevice string, num int) {
	ueConfig.Ue.Msin = fmt.Sprintf("%010d", baseMsinInt+num)
	ueConfig.Ue.UeTunnelDevice = fmt.Sprintf("%s%d", baseUeTunnelDevice, num)
}
