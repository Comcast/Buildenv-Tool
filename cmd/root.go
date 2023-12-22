/*
Copyright © 2023 Comcast Cable Communications Management, LLC
*/
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Comcast/Buildenv-Tool/reader"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	// ErrorCodeMLock Exit Code for MLock Errors
	ErrorCodeMLock = 1
	// ErrorCodeEnv Exit Code for Missing Environment
	ErrorCodeEnv = 2
	// ErrorCodeYaml Exit Code for YAML Errors
	ErrorCodeYaml = 5
	// ErrorCodeVault Exit Code for Vault Errors
	ErrorCodeVault = 6
)

var cfgFile string

var Version string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "buildenv",
	Short: "Set environment variables from a configuation file",
	Long: `Set environment variables based on environment and datacenter. 
Values can be specified in plain text, or set from a vault server.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {

		version, _ := cmd.Flags().GetBool("version")
		if version {
			fmt.Printf("buildenv version %s\n", Version)
		}

		ctx := context.Background()
		debug, _ := cmd.Flags().GetBool("debug")

		enableMlock, _ := cmd.Flags().GetBool("mlock")
		if !enableMlock {
			err := EnableMlock()
			if err != nil {
				fmt.Printf("Failure locking memory: %v", err)
				os.Exit(ErrorCodeMLock)
			}
		}

		// Read the Data File
		variablesFile, _ := cmd.Flags().GetString("variables_file")
		var data reader.Variables

		yamlFile, err := os.ReadFile(variablesFile)
		if err != nil {
			fmt.Printf("Unable to read file %s: %v", variablesFile, err)
			os.Exit(ErrorCodeYaml)
		}
		err = yaml.Unmarshal(yamlFile, &data)
		if err != nil {
			fmt.Printf("Unable to parse YAML file %s: %v", variablesFile, err)
		}
		if debug {
			inData, _ := json.MarshalIndent(data, "", "  ")
			fmt.Printf("Data:\n%s\n\n", inData)
		}

		// Setup the Reader
		reader, err := reader.NewReader()
		if err != nil {
			fmt.Printf("Failure creating Reader: %v", err)
			os.Exit(ErrorCodeVault)
		}

		// Get the Output
		env, _ := cmd.Flags().GetString("environment")
		dc, _ := cmd.Flags().GetString("datacenter")

		out, err := reader.Read(ctx, &data, env, dc)
		if err != nil {
			fmt.Printf("Failure reading data: %v", err)
			os.Exit(ErrorCodeVault)
		}

		if debug {
			outData, _ := json.MarshalIndent(out, "", "  ")
			fmt.Printf("Output:\n%s\n\n", outData)
		}

		// Output the Exports
		comments, _ := cmd.Flags().GetBool("comments")
		out.Print(comments)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version string) {
	Version = version
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.buildenv.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringP("environment", "e", "", "Environment (qa, dev, stage, prod, etc)")
	rootCmd.Flags().StringP("datacenter", "d", "", "Datacenter (ndc_as_a, us-east-1 etc)")
	rootCmd.Flags().StringP("variables_file", "f", "variables.yml", "Variables Source YAML file")

	rootCmd.Flags().BoolP("skip-vault", "v", false, "Skip Vault and use only variables file")
	rootCmd.Flags().BoolP("mlock", "m", false, "Will enable system mlock if set (prevent write to swap on linux)")
	rootCmd.Flags().BoolP("comments", "c", false, "Comments will be included in output")
	rootCmd.Flags().Bool("debug", false, "Turn on debugging output")
	rootCmd.Flags().Bool("version", false, "Print the version number")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".buildenv" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".buildenv")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
