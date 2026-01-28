package config

import (
	"fmt"
	ini "gopkg.in/ini.v1"
)

func GetCurrentConfig(configString string) (name, addr string, err error) {
	cfg, openerr := ini.Load(configString)
	if openerr != nil {
		fmt.Println("fialed open server.ini ,err", openerr)
	}
	name = cfg.Section("currentNode").Key("name").String()
	addr = cfg.Section("currentNode").Key("addr").String()
	return name, addr, openerr
}

func GetClusterConfig(configString string) (name, addr []string, err error) {
	cfg, openerr := ini.Load(configString)
	if openerr != nil {
		fmt.Println("fialed open server.ini ,err", openerr)
	}
	name = cfg.Section("cluster").Key("name").Strings(",")
	addr = cfg.Section("cluster").Key("addr").Strings(",")
	return name, addr, openerr
}
