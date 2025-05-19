/*
Copyright © 2023 Comcast Cable Communications Management, LLC
*/
package cmd

import (
	"context"
	"encoding/base64"
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
	// ErrorCodeInput Exit Code for Bad Input
	ErrorCodeInput = 7
	// ErrorCodeOutput Exit Code for Failed Serialization/Output
	ErrorCodeOutput = 8
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
			os.Exit(0)
		}

		ctx := context.Background()
		debug, _ := cmd.Flags().GetBool("debug")

		enableMlock, _ := cmd.Flags().GetBool("mlock")
		if enableMlock {
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

		skip_vault, _ := cmd.Flags().GetBool("skip-vault")

		// Setup the Reader
		rdr, err := reader.NewReader(reader.WithSkipVault(skip_vault))
		if err != nil {
			fmt.Printf("Failure creating Reader: %v", err)
			os.Exit(ErrorCodeVault)
		}

		// Get the Output
		env, _ := cmd.Flags().GetString("environment")
		run, _ := cmd.Flags().GetString("run")
		dc, _ := cmd.Flags().GetString("datacenter")

		out, err := rdr.Read(ctx, &data, env, dc)
		if err != nil {
			fmt.Printf("Failure reading data: %v", err)
			os.Exit(ErrorCodeVault)
		}

		if debug {
			outData, _ := json.MarshalIndent(out, "", "  ")
			fmt.Printf("Output:\n%s\n\n", outData)
		}

		var use_vars reader.EnvVars

		use, err := cmd.Flags().GetStringArray("use")
		if err != nil {
			fmt.Printf("Could not get \"use\" (-u) flag values: %v", err)
			os.Exit(ErrorCodeInput)
		}

		for _, use_inst := range use {
			blob := os.Getenv(use_inst)
			decoded := make([]byte, base64.StdEncoding.DecodedLen(len(blob)))
			len, err := base64.StdEncoding.Decode(decoded, []byte(blob))
			if err != nil {
				fmt.Printf("Could not decode input to flag \"use\" (-u): %v", err)
				os.Exit(ErrorCodeInput)
			}
			decoded = decoded[:len]
			/* It adds to the structure, merging matching keys */
			err = json.Unmarshal([]byte(decoded), &use_vars)
			if err != nil {
				fmt.Printf("Could not decode input to flag \"use\" (-u): %v", err)
				os.Exit(ErrorCodeInput)
			}
		}

		vars_out := use_vars.GetOutput()
		out = append(out, vars_out...)

		// Output the Exports
		comments, _ := cmd.Flags().GetBool("comments")
		if cmd.Flags().Lookup("run").Changed {
			os.Exit(out.Exec(run))
		} else {
			encoded_export, err := cmd.Flags().GetBool("export")
			if err != nil {
				fmt.Printf("Failure reading export flag: %v", err)
				os.Exit(ErrorCodeInput)
			}

			if encoded_export {
				err = out.PrintB64Json()
				if err != nil {
					fmt.Printf("Failure printing output: %v", err)
					os.Exit(ErrorCodeOutput)
				}
			} else {
				out.Print(comments)
			}
		}
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
	rootCmd.Flags().StringP("run", "r", "", "Shell command to execute with environment")
	rootCmd.Flags().StringP("datacenter", "d", "", "Datacenter (ndc_as_a, us-east-1 etc)")
	rootCmd.Flags().StringP("variables_file", "f", "variables.yml", "Variables Source YAML file")

	rootCmd.Flags().BoolP("skip-vault", "v", false, "Skip Vault and use only variables file")
	rootCmd.Flags().BoolP("mlock", "m", false, "Will enable system mlock if set (prevent write to swap on linux)")
	rootCmd.Flags().BoolP("comments", "c", false, "Comments will be included in output")
	rootCmd.Flags().Bool("debug", false, "Turn on debugging output")
	rootCmd.Flags().Bool("version", false, "Print the version number")
	rootCmd.Flags().StringArrayP("use", "u", []string{}, "Use Stored Vars from named environment variable. Contents should be base64 encoded JSON.")
	rootCmd.Flags().BoolP("export", "x", false, "Print Vars as base64 encoded json")
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
