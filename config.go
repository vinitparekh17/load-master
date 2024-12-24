package main

import (
	"errors"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const DEFAULT_CONF_PATH = "./config.yaml"

var SlbConfig SlbConf

type SlbConf struct {
	Server           serverConf  `yaml:"server" validate:"required"`
	ShardCount       int16       `yaml:"shard_count" validate:"required,gte=1"`
	Upstreams        *[]upstream `yaml:"upstreams,omitempty" validate:"dive, omitempty"`
	Locations        []location  `yaml:"locations" validate:"required,dive"`
	BufferSize       int32       `yaml:"buffer_size" validate:"required,gte=4096"`
	AccessLog        string      `yaml:"access_log" validate:"required,filepath"`
	ErrorLog         string      `yaml:"error_log" validate:"required,filepath"`
	LoadBalancingAlg string      `yaml:"load_balancing_alg" validate:"required,oneof=round_robin least_conn"`
}

type serverConf struct {
	Addr         string        `yaml:"addr" validate:"required"`
	ReadTimeout  time.Duration `yaml:"read_timeout,omitempty"`
	WriteTimeout time.Duration `yaml:"write_timeout,omitempty"`
	IdleTimeout  time.Duration `yaml:"idle_timeout,omitempty"`
}

type upstream struct {
	Name string   `yaml:"name" validate:"required"`
	Addr []string `yaml:"addr" validate:"required,min=1,dive,hostname|ipv4"`
}

type location struct {
	Path      string  `yaml:"path" validate:"required"`
	Upstream  *string `yaml:"upstream,omitempty" validate:"omitempty"`
	Root      string  `yaml:"root,omitempty" validate:"omitempty,dirpath"`
	IndexFile string  `yaml:"index_file"`
	ErrorFile string  `yaml:"error_file"`
}

func init() {
	_, err := os.Stat(DEFAULT_CONF_PATH)
	if errors.Is(err, os.ErrNotExist) {
		writeBaseConfig()
	} else {
		readConfigFile()
	}
}

func writeBaseConfig() {
	SlbConfig = SlbConf{
		Server: serverConf{
			Addr:         ":8080",
			ReadTimeout:  5 * time.Second,
			IdleTimeout:  500 * time.Millisecond,
			WriteTimeout: 5 * time.Second,
		},
		ShardCount:       8,
		BufferSize:       8192,
		AccessLog:        "/var/log/slb/access.log",
		ErrorLog:         "/var/log/slb/error.log",
		LoadBalancingAlg: "round_robin",
		Locations: []location{
			{
				Path:      "/",
				Root:      "/build",
				IndexFile: "index.html",
				ErrorFile: "error.html",
			},
		},
	}

	f, err := os.OpenFile(DEFAULT_CONF_PATH, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	checkErr(err)
	defer f.Close()

	enc := yaml.NewEncoder(f)
	err = enc.Encode(SlbConfig)
	checkErr(err)
}

func readConfigFile() {
	file, err := os.OpenFile(DEFAULT_CONF_PATH, os.O_RDONLY, 0600)
	checkErr(err)

	dec := yaml.NewDecoder(file)
	err = dec.Decode(&SlbConfig)
	checkErr(err)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func strPtr(s string) *string {
	return &s
}
