/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var (
	cfgFile      string
	env          string
	dc           string
	varsFile     string
	backend      string
	mlockEnabled bool
)

//Supported Backends
var supportedBackends = []string{
	"vault",
}

// EnvErrorCode Exit Code for Missing Environment
const EnvErrorCode = 2

// YamlErrorCode Exit Code for YAML Errors
const YamlErrorCode = 5

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "buildenv",
	Short: "Take a yml File and populate the Environment",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		exportSecrets()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.buildenv.yaml)")
	rootCmd.PersistentFlags().StringVarP(&env, "environment", "e", "", "-e <ENVNAME>")
	rootCmd.MarkPersistentFlagRequired("environment")
	rootCmd.PersistentFlags().StringVarP(&dc, "datacenter", "d", "", "-d <DATACENTER>")
	rootCmd.MarkPersistentFlagRequired("datacenter")
	rootCmd.PersistentFlags().StringVarP(&varsFile, "var-file", "f", "variables.yml", "-f <MYVARFILE>.yml")
	rootCmd.PersistentFlags().BoolVarP(&mlockEnabled, "mlock", "m", false, "-m")
	rootCmd.PersistentFlags().StringVarP(&backend, "backend", "b", "vault", "Supported Backend Declaration: vault")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".buildenv" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".buildenv")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func enableMlock(mlock bool) error {
	if mlock {
		mlockError := unix.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE)
		if mlockError != nil {
			errString := fmt.Sprintf("mlock error: %s", mlockError)
			return errors.New(errString)
		}
	}
	return nil
}

func exportSecrets() error {
	type EnvVars map[string]string

	type Secrets map[string]string

	type Config struct {
		Vars         EnvVars
		Secrets      Secrets
		Environments map[string]struct {
			Vars    EnvVars
			Secrets Secrets
			Dcs     map[string]struct {
				Vars    EnvVars
				Secrets Secrets
			}
		}
	}

	var config Config

	enableMlock(mlockEnabled)

	filename, _ := filepath.Abs(varsFile)
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return err
	}

	fmt.Println("# Setting Variables for:")
	fmt.Printf("# Environment: %s\n", env)
	if dc != "" {
		fmt.Printf("# Datacenter: %s\n", dc)
	}

	// Print The Globals
	fmt.Println("# Global Vars:")
	for k, v := range config.Vars {
		fmt.Printf("export %s=%q\n", k, v)
	}

	fmt.Println("# Global Secrets:")
	for k, path := range config.Secrets {
		secret, err := GetVaultSecret(path)
		if err != nil {
			return err
		} else {
			fmt.Printf("export %s=%q # %s\n", k, secret.Data["value"], path)
		}
	}

	// Print The Environment Specific Vars
	fmt.Printf("# Environment (%s) Vars:\n", env)
	for k, v := range config.Environments[env].Vars {
		fmt.Printf("export %s=%q\n", k, v)
	}

	fmt.Printf("# Environment (%s) Secrets:\n", env)
	for k, path := range config.Environments[env].Secrets {
		secret, err := GetVaultSecret(path)
		if err != nil {
			return err
		} else {
			fmt.Printf("export %s=%q # %s\n", k, secret.Data["value"], path)
		}
	}

	return nil
}

func GetVaultSecret(path string) (*vaultapi.Secret, error) {
	// Get Config Completely From Environment
	var c *vaultapi.Config

	vault, err := vaultapi.NewClient(c)

	if err != nil {
		return nil, fmt.Errorf("Vault - Client Error: %s", err)
	}

	vaultSecret, err := vault.Logical().Read(path)

	if err != nil {
		return nil, fmt.Errorf("Vault - Read Error: %s", err)
	}
	if vaultSecret == nil {
		return nil, fmt.Errorf("Vault - No secret at path: %s", path)
	}
	return vaultSecret, nil
}
