package reader

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/hashicorp/vault-client-go"
)

type Reader struct {
	client *vault.Client
	mounts Mounts
}

type EnvVars map[string]string

func (e EnvVars) GetOutput() OutputList {
	output := OutputList{}
	for k, v := range e {
		output = append(output, Output{
			Key:   k,
			Value: v,
		})
	}
	return output
}

type Secrets map[string]string

func (s Secrets) GetOutput(ctx context.Context, r *Reader) (OutputList, error) {
	// Read it like a kv secrets where all keys are "value"
	kvSecrets := KVSecrets{}
	for outVar, path := range s {
		kvSecret := KVSecretBlock{
			Path: path,
			Vars: KVSecret{
				outVar: "value",
			},
		}
		kvSecrets = append(kvSecrets, kvSecret)
	}
	return kvSecrets.GetOutput(ctx, r)
}

type KVSecret map[string]string

type KVSecretBlock struct {
	Path string
	Vars KVSecret
}

type KVSecrets []KVSecretBlock

func (s KVSecretBlock) GetOutput(ctx context.Context, r *Reader) (OutputList, error) {
	output := OutputList{}

	// Initialize the Vault Client if Necessary
	if r.client == nil {
		err := r.InitVault()
		if err != nil {
			return nil, err
		}
	}

	// The first thing we need to do is get the mount point for the KV engine
	mountPoint, secretPath := r.MountAndPath(s.Path)
	if mountPoint == "" {
		return nil, fmt.Errorf("no mount point found for path %s", s.Path)
	}

	// V2 KV Secrets
	if r.mounts[mountPoint].Type == "kv" && r.mounts[mountPoint].Version == "2" {
		// Get Secret
		resp, err := r.client.Secrets.KvV2Read(ctx, secretPath, vault.WithMountPath(mountPoint))
		if err != nil {
			if vault.IsErrorStatus(err, http.StatusNotFound) {
				return nil, fmt.Errorf("secret does not exist: '%s'", s.Path)
			}
			return nil, fmt.Errorf("error reading path '%s': %w", s.Path, err)
		}
		// For testing purposes, we want to order this
		envVars := []string{}
		for varName := range s.Vars {
			envVars = append(envVars, varName)
		}
		slices.Sort(envVars)
		for _, varName := range envVars {
			varKey := s.Vars[varName]
			if _, hasValue := resp.Data.Data[varKey]; !hasValue {
				return nil, fmt.Errorf("key %s not found in path %s", varKey, s.Path)
			}
			val := fmt.Sprintf("%s", resp.Data.Data[varKey])
			output = append(output, Output{
				Key:     varName,
				Value:   val,
				Comment: fmt.Sprintf("Path: %s, Key: %s", s.Path, varKey),
			})
		}
	} else {
		// Treat it as a KVv1 secret
		resp, err := r.client.Secrets.KvV1Read(ctx, secretPath, vault.WithMountPath(mountPoint))
		if err != nil {
			return nil, fmt.Errorf("error reading path %s: %w", s.Path, err)
		}
		for varName, varKey := range s.Vars {
			if _, hasValue := resp.Data[varKey]; !hasValue {
				return nil, fmt.Errorf("key %s not found in path %s", varKey, s.Path)
			}
			val := fmt.Sprintf("%s", resp.Data[varKey])
			output = append(output, Output{
				Key:     varName,
				Value:   val,
				Comment: fmt.Sprintf("Path: %s, Key: %s", s.Path, varKey),
			})
		}
	}

	return output, nil
}

func (s KVSecrets) GetOutput(ctx context.Context, r *Reader) (OutputList, error) {
	output := OutputList{}
	for _, block := range s {
		blockOutput, err := block.GetOutput(ctx, r)
		if err != nil {
			return nil, err
		}
		output = append(output, blockOutput...)
	}
	return output, nil
}

type DC struct {
	Vars      EnvVars   `yaml:"vars,omitempty"`
	Secrets   Secrets   `yaml:"secrets,omitempty"`
	KVSecrets KVSecrets `yaml:"kv_secrets,omitempty"`
}

type Environment struct {
	Vars      EnvVars       `yaml:"vars,omitempty"`
	Secrets   Secrets       `yaml:"secrets,omitempty"`
	KVSecrets KVSecrets     `yaml:"kv_secrets,omitempty"`
	Dcs       map[string]DC `yaml:"dcs,omitempty"`
}

type Variables struct {
	Vars         EnvVars                `yaml:"vars,omitempty"`
	Secrets      Secrets                `yaml:"secrets,omitempty"`
	KVSecrets    KVSecrets              `yaml:"kv_secrets,omitempty"`
	Environments map[string]Environment `yaml:"environments,omitempty"`
}

type Output struct {
	Key     string
	Value   string
	Comment string
}
type OutputList []Output

func (o OutputList) Print(showComments bool) {
	for _, out := range o {
		keySpace := ""
		nl := false
		if out.Key != "" {
			fmt.Printf("export %s=%q", out.Key, out.Value)
			keySpace = " "
			nl = true
		}
		if out.Comment != "" && showComments {
			fmt.Printf("%s# %s", keySpace, out.Comment)
			nl = true
		}
		if nl {
			fmt.Println()
		}
	}
}

type MountInfo struct {
	Type    string
	Version string
}

type Mounts map[string]MountInfo

func (r *Reader) InitVault() error {
	vaultClient, err := vault.New(vault.WithEnvironment())
	if err != nil {
		return err
	}
	r.client = vaultClient

	// Get mount info
	resp, err := vaultClient.System.MountsListSecretsEngines(context.Background())
	if err != nil {
		return fmt.Errorf("failure reading secret mounts: %w", err)
	}

	mounts := Mounts{}
	for mount, details := range resp.Data {
		detailMap := details.(map[string]interface{})
		thisMount := MountInfo{
			Type: detailMap["type"].(string),
		}
		if options, hasOptions := detailMap["options"]; hasOptions && options != nil {
			optionMap := options.(map[string]interface{})
			if version, hasVersion := optionMap["version"]; hasVersion {
				thisMount.Version = version.(string)
			}
		}
		mounts[mount] = thisMount
	}

	r.mounts = mounts
	return nil
}

func NewReader() (*Reader, error) {
	return &Reader{}, nil
}

func (r *Reader) MountAndPath(path string) (string, string) {
	for mount := range r.mounts {
		if strings.HasPrefix(path, mount) {
			return mount, strings.TrimPrefix(path, mount)
		}
	}
	return "", ""
}

func (r *Reader) Read(ctx context.Context, input *Variables, env string, dc string) (OutputList, error) {
	output := OutputList{}

	// Global Variables
	output = append(output, Output{
		Comment: "Global Variables",
	})
	output = append(output, input.Vars.GetOutput()...)

	// Global Secrets
	kvOut, err := input.KVSecrets.GetOutput(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("kv secret error: %w", err)
	}
	output = append(output, kvOut...)
	secretOut, err := input.Secrets.GetOutput(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("secret error: %w", err)
	}
	output = append(output, secretOut...)

	// Environment Variablers
	if env != "" {
		output = append(output, Output{
			Comment: fmt.Sprintf("Environment: %s", env),
		})
		output = append(output, input.Environments[env].Vars.GetOutput()...)
		kvOut, err := input.Environments[env].KVSecrets.GetOutput(ctx, r)
		if err != nil {
			return nil, fmt.Errorf("kv secret error: %w", err)
		}
		output = append(output, kvOut...)
		secretOut, err := input.Environments[env].Secrets.GetOutput(ctx, r)
		if err != nil {
			return nil, fmt.Errorf("secret error: %w", err)
		}
		output = append(output, secretOut...)
	}

	// DC Variables
	if dc != "" {
		output = append(output, Output{
			Comment: fmt.Sprintf("Datacenter: %s", dc),
		})
		output = append(output, input.Environments[env].Dcs[dc].Vars.GetOutput()...)
		kvOut, err := input.Environments[env].Dcs[dc].KVSecrets.GetOutput(ctx, r)
		if err != nil {
			return nil, fmt.Errorf("kv secret error: %w", err)
		}
		output = append(output, kvOut...)
		secretOut, err := input.Environments[env].Dcs[dc].Secrets.GetOutput(ctx, r)
		if err != nil {
			return nil, fmt.Errorf("secret error: %w", err)
		}
		output = append(output, secretOut...)
	}

	return output, nil
}
