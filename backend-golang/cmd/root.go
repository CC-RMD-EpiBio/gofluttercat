/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/internal"
	"github.com/spf13/cobra"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cat",
	Short: "This example app serves the RWAS instrument",
	Long: `GoFlutterCat is a computer adaptive testing implementation of
	modern selection techniques`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%v\n", internal.Logo)
		cmd.Help()
	},
}

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "RWAS",
	Long:  internal.Logo,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("args: %v\n", args)
		launchCat()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.backend-golang.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.AddCommand(ServerCmd)
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
