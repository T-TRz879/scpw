package scpw

import (
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

var (
	config []*Node
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
	var c []*Node
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}

	config = c

	return config, nil
}

func LoadConfigBytes(names ...string) ([]byte, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	// homedir
	for i := range names {
		sshw, err := os.ReadFile(filepath.Join(u.HomeDir, names[i]))
		if err == nil {
			return sshw, nil
		}
	}
	// relative
	for i := range names {
		sshw, err := os.ReadFile(names[i])
		if err == nil {
			return sshw, nil
		}
	}
	return nil, err
}
