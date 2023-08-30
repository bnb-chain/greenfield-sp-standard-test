package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/bnb-chain/greenfield-sp-standard-test/core/log"
)

type Config struct {
	GreenfieldChainId  string `yaml:"GreenfieldChainId"`
	GreenfieldEndpoint string `yaml:"GreenfieldEndpoint"`
	SPAddr             string `yaml:"SPOperatorAddr"`
	RootAcc            string `yaml:"RootAcc"`
	TestAcc            string `yaml:"TestAcc"`

	// for performance testing
	TPS      int    `yaml:"TPS"`
	FileSize uint64 `yaml:"FileSize"`
	Duration int64  `yaml:"Duration"`
	NumUsers int    `yaml:"NumUsers"`
	Upload   bool   `yaml:"Upload"`
	Download bool   `yaml:"Download"`

	HttpHeaders map[string]string `yaml:"HttpHeaders"`
}

var CfgEnv Config

func init() {
	InitConfig()
}

func InitConfig() {
	pwd, _ := os.Getwd()
	cfgPath := filepath.Join(pwd, "config", "config.yml")
	LoadConfig(cfgPath, &CfgEnv)
}

func LoadConfig(cfgPath string, cfgEnv *Config) {
	log.Infof("===config file: %s ===", cfgPath)
	fileContent, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Fatal("ReadFile err: ", err)
	}
	err = yaml.Unmarshal(fileContent, cfgEnv)
	if err != nil {
		log.Fatal("Unmarshal err: ", err)
	}
}
