package main

import (
	"fmt"
	"os"

	"github.com/realmicro/realmicro/config"
	"github.com/realmicro/realmicro/config/encoder/yaml"
	"github.com/realmicro/realmicro/config/reader"
	"github.com/realmicro/realmicro/config/reader/json"
	"github.com/realmicro/realmicro/config/source/env"
)

func main() {
	os.Setenv("REALMICRO_CFG_HOSTS_DATABASE_ADDRESS", `10.0.0.1,10.0.0.2`)
	os.Setenv("REALMICRO_CFG_HOSTS_DATABASE_PORT", `3306`)

	// new config
	c, _ := config.NewConfig(
		config.WithReader(
			json.NewReader(
				reader.WithEncoder(yaml.NewEncoder()),
			),
		),
	)

	if err := c.Load(env.NewSource(
		env.WithStrippedPrefix("REALMICRO_CFG"),
	)); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("data", c.Map())

	type Cfg struct {
		Hosts struct {
			Database struct {
				Address []string `json:"address"`
				Port    int      `json:"port"`
			} `json:"database"`
			Cache struct {
				Address string `json:"address"`
				Port    int    `json:"port"`
			} `json:"cache"`
		} `json:"hosts"`
		Env string `json:"env"`
	}

	var cfg Cfg

	// read a database host
	if err := c.Scan(&cfg); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(cfg)
}
