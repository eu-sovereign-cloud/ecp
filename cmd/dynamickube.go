package cmd

import (
	"github.com/spf13/cobra"

	"github.com/eu-sovereign-cloud/ecp/pkg/kubeclient"
	"github.com/eu-sovereign-cloud/ecp/pkg/logger"
)

var dynamicKubeCMD = &cobra.Command{
	Use:   "dynamickube",
	Short: "Instantiate a DynamicKube client and read kubeconfig",
	Run: func(cmd *cobra.Command, args []string) {
		ctrl, err := kubeclient.New()
		if err != nil {
			logger.Log.Fatalw("Failed to instantiate DynamicKubeController:", "err", err)
		}
		logger.Log.Infow("DynamicKubeController instantiated:", "ctrl", ctrl)
	},
}

func init() {
	rootCmd.AddCommand(dynamicKubeCMD)
}
