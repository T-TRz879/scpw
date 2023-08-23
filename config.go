package scpw

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"os/user"
	"path/filepath"
)

type SCPWType = string

const (
	PUT SCPWType = "PUT"
	GET SCPWType = "GET"
)

type Node struct {
	Name     string   `yaml:"name"`
	Host     string   `yaml:"host"`
	User     string   `yaml:"user"`
	Port     string   `yaml:"port"`
	KeyPath  string   `yaml:"keypath"`
	Password string   `yaml:"password"`
	Children []*Node  `yaml:"children"`
	LRMap    []LRMap  `yaml:"lr-map"`
	Typ      SCPWType `yaml:"type"`
}

type LRMap struct {
	Local  string `yaml:"local"`
	Remote string `yaml:"remote"`
}

func LoadConfig() ([]*Node, error) {
	b, err := LoadConfigBytes(".scpw", ".scpw.yml", ".scpw.yaml")
	if err != nil {
		return nil, err
	}
	var config []*Node
	err = yaml.Unmarshal(b, &config)
	return config, err
}

func LoadConfigBytes(names ...string) ([]byte, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	// homedir
	for i := range names {
		if sshw, e := os.ReadFile(filepath.Join(u.HomeDir, names[i])); e == nil {
			return sshw, nil
		}
	}
	// relative
	for i := range names {
		if sshw, e := os.ReadFile(names[i]); e == nil {
			return sshw, nil
		}
	}
	return nil, fmt.Errorf("cannot find config from %s", u.HomeDir)
}
