package scpw

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	u, err := user.Current()
	os.Remove(filepath.Join(u.HomeDir, ".scpw"))
	os.Remove(filepath.Join(u.HomeDir, ".scpw.yml"))
	os.Remove(filepath.Join(u.HomeDir, ".scpw.yaml"))

	_, err = LoadConfig()
	assert.NotNil(t, err)

	var config []*Node
	config = append(config, &Node{Name: "local", Host: "127.0.0.1", User: "root", Port: "22", Password: "123", LRMap: []LRMap{{Local: "/tmp/a", Remote: "/tmp/b"}}, Typ: GET})
	b, err := yaml.Marshal(config)
	assert.Nil(t, err)
	_, err = os.Create(filepath.Join(u.HomeDir, ".scpw"))
	assert.Nil(t, err)
	assert.Nil(t, os.WriteFile(filepath.Join(u.HomeDir, ".scpw"), b, os.FileMode(0777)))

	_, err = LoadConfig()
	assert.Nil(t, err)
}
