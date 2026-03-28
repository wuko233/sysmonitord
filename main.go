package main

import (
	"fmt"
	"os"
	"sysmonitord/cmd/start"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "sysmonitord",
		Short: "Sysmonitord 是一个 Linux 系统安全监控工具",
	}

	rootCmd.AddCommand(start.StartCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
