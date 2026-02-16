package reader

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/vault-client-go"
)

func TestEnvVars_GetOutput(t *testing.T) {
	tests := []struct {
		name string
		e    EnvVars
		want OutputList
	}{
		{
			name: "Test Output",
			e: EnvVars{
				"a": "b",
			},
			want: OutputList{
				{
					Key:   "a",
					Value: "b",
				},
			},
		},
		{
			name: "Empty Output",
			e:    EnvVars{},
			want: OutputList{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.GetOutput(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EnvVars.GetOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReader_Read(t *testing.T) {
	type fields struct {
		client *vault.Client
	}
	type args struct {
		input *Variables
		env   string
		dc    string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    OutputList
		wantErr bool
	}{
		{
			name:   "Just Plain Variables",
			fields: fields{},
			args: args{
				env: "dev",
				dc:  "us-least-1",
				input: &Variables{
					Vars: EnvVars{
						"FOO": "bar",
					},
					Environments: map[string]Environment{
						"dev": {
							Vars: EnvVars{
								"ENV": "dev",
							},
							Dcs: map[string]DC{
								"us-least-1": {
									Vars: EnvVars{
										"DC": "us-least-1",
									},
								},
							},
						},
						"stage": {
							Vars: EnvVars{
								"env": "stage",
							},
						},
					},
				},
			},
			want: OutputList{
				{
					Comment: "Global Variables",
				},
				{
					Key:   "FOO",
					Value: "bar",
				},
				{
					Comment: "Environment: dev",
				},
				{
					Key:   "ENV",
					Value: "dev",
				},
				{
					Comment: "Datacenter: us-least-1",
				},
				{
					Key:   "DC",
					Value: "us-least-1",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			r := &Reader{
				client: tt.fields.client,
			}
			got, err := r.Read(ctx, tt.args.input, tt.args.env, tt.args.dc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reader.Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reader.Read() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestKVSecretBlock_GetOutputNoDetect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%+v", r)
		var resp []byte
		status := http.StatusOK

		// KV Data
		switch r.URL.Path {
		case "/v1/kv2/data/test":
			resp = []byte(`{"request_id":"bf3b02c0-096e-84d3-dad7-196aa9f112ed","lease_id":"","renewable":false,"lease_duration":0,"data":{"data":{"one":"1","two":"2","three":"3"},"metadata":{"created_time":"2023-12-20T15:32:32.814115685Z","custom_metadata":null,"deletion_time":"","destroyed":false,"version":1}},"wrap_info":null,"warnings":null,"auth":null}`)
		case "/v1/kv/test":
			resp = []byte(`{"request_id":"63c8c31b-f03f-81ac-cfaa-324239789c3f","lease_id":"","renewable":false,"lease_duration":2764800,"data":{"value":"old"},"wrap_info":null,"warnings":null,"auth":null}`)
		default:
			status = http.StatusNotFound
			resp = []byte(`{"errors":[]}`)
		}

		w.WriteHeader(status)
		w.Write(resp)
	}))
	defer server.Close()

	client, _ := vault.New(vault.WithAddress(server.URL))
	reader := &Reader{
		client:          client,
		canDetectMounts: false,
	}

	type fields struct {
		Path string
		Vars KVSecret
	}
	type args struct {
		r *Reader
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    OutputList
		wantErr bool
	}{
		{
			name: "No KV Path",
			args: args{
				r: reader,
			},
			fields: fields{
				Path: "kv2/path",
				Vars: KVSecret{
					"NOT": "here",
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "No KV2 Key",
			args: args{
				r: reader,
			},
			fields: fields{
				Path: "kv2/test",
				Vars: KVSecret{
					"THREE": "nope",
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "With no autodection, KV Read Fails",
			args: args{
				r: reader,
			},
			fields: fields{
				Path: "kv/test",
				Vars: KVSecret{
					"VALUE": "value",
				},
			},
			wantErr: true,
		},
		{
			name: "Test KV2 Read",
			args: args{
				r: reader,
			},
			fields: fields{
				Path: "kv2/test",
				Vars: KVSecret{
					"ONE":   "one",
					"TWO":   "two",
					"THREE": "three",
				},
			},
			want: OutputList{
				{
					Key:     "ONE",
					Value:   "1",
					Comment: "Path: kv2/test, Key: one",
				},
				{
					Key:     "THREE",
					Value:   "3",
					Comment: "Path: kv2/test, Key: three",
				},
				{
					Key:     "TWO",
					Value:   "2",
					Comment: "Path: kv2/test, Key: two",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			s := KVSecretBlock{
				Path: tt.fields.Path,
				Vars: tt.fields.Vars,
			}
			got, err := s.GetOutput(ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("KVSecretBlock.GetOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KVSecretBlock.GetOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKVSecretBlock_GetOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%+v", r)
		var resp []byte
		status := http.StatusOK

		// KV Data
		switch r.URL.Path {
		case "/v1/kv2/data/test":
			resp = []byte(`{"request_id":"bf3b02c0-096e-84d3-dad7-196aa9f112ed","lease_id":"","renewable":false,"lease_duration":0,"data":{"data":{"one":"1","two":"2","three":"3"},"metadata":{"created_time":"2023-12-20T15:32:32.814115685Z","custom_metadata":null,"deletion_time":"","destroyed":false,"version":1}},"wrap_info":null,"warnings":null,"auth":null}`)
		case "/v1/kv/test":
			resp = []byte(`{"request_id":"63c8c31b-f03f-81ac-cfaa-324239789c3f","lease_id":"","renewable":false,"lease_duration":2764800,"data":{"value":"old"},"wrap_info":null,"warnings":null,"auth":null}`)
		default:
			status = http.StatusNotFound
			resp = []byte(`{"errors":[]}`)
		}

		w.WriteHeader(status)
		w.Write(resp)
	}))
	defer server.Close()

	client, _ := vault.New(vault.WithAddress(server.URL))
	reader := &Reader{
		client:          client,
		canDetectMounts: true,
		mounts: Mounts{
			"kv2/": {
				Type:    "kv",
				Version: "2",
			},
			"kv/": {
				Type: "kv",
			},
			"generic/": {
				Type: "generic",
			},
		},
	}

	type fields struct {
		Path string
		Vars KVSecret
	}
	type args struct {
		r *Reader
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    OutputList
		wantErr bool
	}{
		{
			name: "No Mount",
			args: args{
				r: reader,
			},
			fields: fields{
				Path: "secret/test",
				Vars: KVSecret{
					"should": "fail",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "No KV Path",
			args: args{
				r: reader,
			},
			fields: fields{
				Path: "kv2/path",
				Vars: KVSecret{
					"NOT": "here",
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "No KV2 Key",
			args: args{
				r: reader,
			},
			fields: fields{
				Path: "kv2/test",
				Vars: KVSecret{
					"THREE": "nope",
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "Test KV Read",
			args: args{
				r: reader,
			},
			fields: fields{
				Path: "kv/test",
				Vars: KVSecret{
					"VALUE": "value",
				},
			},
			want: OutputList{
				{
					Key:     "VALUE",
					Value:   "old",
					Comment: "Path: kv/test, Key: value",
				},
			},
		},
		{
			name: "Test KV2 Read",
			args: args{
				r: reader,
			},
			fields: fields{
				Path: "kv2/test",
				Vars: KVSecret{
					"ONE":   "one",
					"TWO":   "two",
					"THREE": "three",
				},
			},
			want: OutputList{
				{
					Key:     "ONE",
					Value:   "1",
					Comment: "Path: kv2/test, Key: one",
				},
				{
					Key:     "THREE",
					Value:   "3",
					Comment: "Path: kv2/test, Key: three",
				},
				{
					Key:     "TWO",
					Value:   "2",
					Comment: "Path: kv2/test, Key: two",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			s := KVSecretBlock{
				Path: tt.fields.Path,
				Vars: tt.fields.Vars,
			}
			got, err := s.GetOutput(ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("KVSecretBlock.GetOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KVSecretBlock.GetOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKV1SecretBlock_GetOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%+v", r)
		var resp []byte
		status := http.StatusOK

		// KV Data
		switch r.URL.Path {
		case "/v1/kv/test":
			resp = []byte(`{"request_id":"63c8c31b-f03f-81ac-cfaa-324239789c3f","lease_id":"","renewable":false,"lease_duration":2764800,"data":{"value":"old"},"wrap_info":null,"warnings":null,"auth":null}`)
		default:
			status = http.StatusNotFound
			resp = []byte(`{"errors":[]}`)
		}

		w.WriteHeader(status)
		w.Write(resp)
	}))
	defer server.Close()

	client, _ := vault.New(vault.WithAddress(server.URL))
	reader := &Reader{
		client:          client,
		canDetectMounts: true,
		mounts: Mounts{
			"kv2/": {
				Type:    "kv",
				Version: "2",
			},
			"kv/": {
				Type: "kv",
			},
			"generic/": {
				Type: "generic",
			},
		},
	}

	type fields struct {
		Path string
		Vars KVSecret
	}
	type args struct {
		r *Reader
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    OutputList
		wantErr bool
	}{
		{
			name: "No Mount",
			args: args{
				r: reader,
			},
			fields: fields{
				Path: "secret/test",
				Vars: KVSecret{
					"should": "fail",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "No KV Path",
			args: args{
				r: reader,
			},
			fields: fields{
				Path: "kv/path",
				Vars: KVSecret{
					"NOT": "here",
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "Test KV Read",
			args: args{
				r: reader,
			},
			fields: fields{
				Path: "kv/test",
				Vars: KVSecret{
					"VALUE": "value",
				},
			},
			want: OutputList{
				{
					Key:     "VALUE",
					Value:   "old",
					Comment: "Path: kv/test, Key: value",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			s := KV1SecretBlock{
				Path: tt.fields.Path,
				Vars: tt.fields.Vars,
			}
			got, err := s.GetOutput(ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("KVSecretBlock.GetOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KVSecretBlock.GetOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSkipVault_Reader(t *testing.T) {
	reader, _ := NewReader(WithSkipVault(true))

	type args struct {
		r    *Reader
		i    *Variables
		env  string
		dc   string
		skip bool
	}

	tests := []struct {
		name    string
		args    args
		want    OutputList
		wantErr bool
	}{
		{
			name: "Has Secrets",
			args: args{
				skip: true,
				env:  "dev",
				dc:   "us-least-1",
				r:    reader,
				i: &Variables{
					Vars: EnvVars{
						"XYZ": "yep",
					},
					Secrets: Secrets{
						"Secret1": "it's here",
					},
					KVSecrets: KVSecrets{{
						Path: "path/test",
						Vars: KVSecret{"KVSecret1": "kvsecret1"},
					}},
					KV1Secrets: KV1Secrets{{
						Path: "path2/test",
						Vars: KVSecret{
							"KV1Secret1": "another one",
						},
					}},
				},
			},
			want: OutputList{
				{
					Comment: "Global Variables",
				},
				{Key: "XYZ", Value: "yep", Comment: ""},
				{
					Comment: "Environment: dev",
				},
				{
					Comment: "Datacenter: us-least-1",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := tt.args.r.Read(ctx, tt.args.i, tt.args.env, tt.args.dc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reader.Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reader.Read() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputList_Exec(t *testing.T) {
	key := "BuildEnvTestKey"
	val := "BuildEnvTestVal"
	outputList := OutputList{
		Output{key, val, "acomment"},
	}

	type fields struct {
		Key string
		Val string
		Out OutputList
	}

	type args struct {
		cmd string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr int
	}{
		{
			name: "Good Simple Cmd",
			args: args{
				cmd: "echo -n hi",
			},
			fields: fields{
				Key: key,
				Val: val,
				Out: outputList,
			},
			want:    "hi",
			wantErr: 0,
		},
		{
			name: "Bad Cmd",
			args: args{
				cmd: "./nosuchcommandprobably",
			},
			fields: fields{
				Key: key,
				Val: val,
				Out: outputList,
			},
			want:    "",
			wantErr: 127,
		},
		{
			name: "Has Env Var val",
			args: args{
				cmd: "echo -n ${" + key + "}",
			},
			fields: fields{
				Key: key,
				Val: val,
				Out: outputList,
			},
			want:    val,
			wantErr: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldstdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			got := tt.fields.Out.Exec(tt.args.cmd)

			outC := make(chan string)

			go func() {
				var buf bytes.Buffer
				io.Copy(&buf, r)
				outC <- buf.String()
			}()

			w.Close()

			out := <-outC

			os.Stdout = oldstdout

			if out != tt.want {
				t.Errorf("OutputList.Exec() output = %v, want %v", out, tt.want)
			}
			if got != tt.wantErr {
				t.Errorf("OutputList.Exec() err = %v, wantErr %v", got, tt.wantErr)
			}
		})
	}
}

func TestOutputList_PrintB64Json(t *testing.T) {
	envVars := EnvVars{
		"BuildEnvTestKey1": "BuildEnvTestVal1",
		"BuildEnvTestKey2": "BuildEnvTestVal2",
	}

	type fields struct {
		Out OutputList
	}

	tests := []struct {
		name    string
		fields  fields
		want    EnvVars
		wantErr bool
	}{
		{
			name:   "Basic 2 vars",
			fields: fields{Out: envVars.GetOutput()},
			want:   envVars,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldstdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			outC := make(chan string)

			go func() {
				var buf bytes.Buffer
				io.Copy(&buf, r)
				outC <- buf.String()
			}()

			err := tt.fields.Out.PrintB64Json()
			if err != nil {
				t.Errorf("PrintB64Json() error = %v", err)
			}

			os.Stdout = oldstdout

			w.Close()

			out := <-outC

			decoded := make([]byte, base64.StdEncoding.DecodedLen(len(out)))
			len, err := base64.StdEncoding.Decode(decoded, []byte(out))

			var out_vars EnvVars

			if err != nil {
				t.Errorf("Exception parsing output: %v", err)
				return
			}

			decoded = decoded[:len]

			err = json.Unmarshal([]byte(decoded), &out_vars)
			if err != nil {
				t.Errorf("Exception parsing output: %v", err)
				return
			}

			if !reflect.DeepEqual(out_vars, tt.want) {
				t.Errorf("PrintB64Json() output = %v, want %v", out_vars, tt.want)
			}
		})
	}
}
