package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	DefaultConfPath       = filepath.Base("config.yaml")
	DefaultStaticRootPath = filepath.Base("static")
	DefaultIndexPage      = filepath.Join(DefaultStaticRootPath, "index.html")
	DefaultError404Page   = filepath.Join(DefaultStaticRootPath, "404.html")
	DefaultError500Page   = filepath.Join(DefaultStaticRootPath, "500.html")
)
var SlbConfig *Config

type Config struct {
	Server           serverConf          `yaml:"server" validate:"required"`
	ShardCount       int                 `yaml:"shard_count" validate:"required,gte=1"`
	Locations        map[string]location `yaml:"locations" validate:"required,locationsMap"`
	BufferSize       int                 `yaml:"buffer_size" validate:"required,gte=4096"`
	LoadBalancingAlg string              `yaml:"load_balancing_alg" validate:"required,oneof=round_robin least_conn"`
}

type serverConf struct {
	Addr         string        `yaml:"addr" validate:"required"`
	ReadTimeout  time.Duration `yaml:"read_timeout,omitempty"`
	WriteTimeout time.Duration `yaml:"write_timeout,omitempty"`
	IdleTimeout  time.Duration `yaml:"idle_timeout,omitempty"`
}

type Upstream struct {
	Name string   `yaml:"name" validate:"required"`
	Addr []string `yaml:"addr" validate:"required,min=1,dive,hostname|ipv4"`
}

type location struct {
	Upstream *Upstream `yaml:"upstream,omitempty" validate:"omitempty"`
}

func init() {
	_, err := os.Stat(DefaultConfPath)
	if errors.Is(err, os.ErrNotExist) {
		writeBaseConfig()
	} else {
		readConfigFile()
		validateConfig()
	}
}

func writeBaseConfig() {
	SlbConfig = &Config{
		Server: serverConf{
			Addr:         ":8080",
			ReadTimeout:  5 * time.Second,
			IdleTimeout:  500 * time.Millisecond,
			WriteTimeout: 5 * time.Second,
		},
		ShardCount:       8,
		BufferSize:       8192,
		LoadBalancingAlg: "round_robin",
		Locations: map[string]location{
			"/": {},
			"/api": {
				Upstream: &Upstream{
					Name: "backend-1",
					Addr: []string{"127.0.0.1:8000", "127.0.0.1:8001"},
				}},
		},
	}

	f, err := os.OpenFile(DefaultConfPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	checkErr(err)
	defer f.Close()

	enc := yaml.NewEncoder(f)
	err = enc.Encode(SlbConfig)
	checkErr(err)
}

func readConfigFile() {
	file, err := os.OpenFile(DefaultConfPath, os.O_RDONLY, 0600)
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
