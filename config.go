package main

import "github.com/koding/multiconfig"

type Config struct {
	CacheExpire      int    `default:24`
	SaveFileInterval int    `default:120`
	GetInterval      int    `default:5`
	CleanupInterval  int    `default:120`
	CacheFileName    string `default:"cache.dat"`
	ListenAddress    string `default:":18181"`
	DataFormatStyle  string `default:"15:04:05"`
}

var (
	config = new(Config)
)

//读取配置文件
func readConfig() {
	m := multiconfig.NewWithPath("config.toml") // supports TOML and JSON
	// Populated the serverConf struct
	m.MustLoad(config) // Check for error
}
