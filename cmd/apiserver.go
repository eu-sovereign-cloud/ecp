package cmd

import (
	"github.com/spf13/cobra"

	"ecp/pkg/apiserver"
)

var (
	host string
	port string
)

// apiServerCMD represents the apiserver command
var apiServerCMD = &cobra.Command{
	Use:   "apiserver",
	Short: "A brief description of your command",
	Long:  `The API server command starts the backend server for the ECP application`,
	Run: func(cmd *cobra.Command, args []string) {
		apiserver.Start(host + ":" + port)
	},
}

func init() {
	apiServerCMD.Flags().StringVar(&host, "host", "0.0.0.0", "Host to bind the server to")
	apiServerCMD.Flags().StringVarP(&port, "port", "p", "8080", "Port to bind the server to")
	rootCmd.AddCommand(apiServerCMD)
}
