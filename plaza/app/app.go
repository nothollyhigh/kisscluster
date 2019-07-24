package app

import (
	"flag"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/nothollyhigh/kiss/log"
	"github.com/nothollyhigh/kiss/util"
	"io"
	"io/ioutil"
	"os"
	"time"
)

var (
	appVersion = ""
	config     = &Config{}
	json       = jsoniter.ConfigCompatibleWithStandardLibrary
	confpath   = flag.String("config", "./conf/plaza.json", "config file path, default is conf/plaza.json")

	logout = io.Writer(nil)
)

type Config struct {
	Debug  bool   `json:"Debug"`
	LogDir string `json:"LogDir"`

	SvrID     string `json:"SvrID"`
	SvrPasswd string `json:"SvrPasswd"`

	CenterAddr string `json:"CenterAddr"`

	SvrAddr    string `json:"SvrAddr"`
	StaticAddr string `json:"StaticAddr"`
}

func initConfig() {
	flag.Parse()

	filename := *confpath
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Panic("initConfig ReadFile Failed: %v", err)
	}

	data = util.TrimComment(data)
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Panic("initConfig json.Unmarshal Failed: %v", err)
	}

}

func initLog() {
	var (
		fileWriter = &log.FileWriter{
			RootDir:     config.LogDir + time.Now().Format("20060102150405/"),
			DirFormat:   "",
			FileFormat:  "20060102.log",
			MaxFileSize: 1024 * 1024 * 32,
			EnableBufio: false,
		}
	)
	if config.Debug {
		logout = io.MultiWriter(os.Stdout, fileWriter)
		gin.SetMode(gin.DebugMode)
	} else {
		logout = fileWriter
		gin.SetMode(gin.ReleaseMode)
	}

	log.SetOutput(logout)

	gin.DefaultWriter = logout

	configData, _ := json.MarshalIndent(config, "", "    ")
	log.Info("config: %v\n%v", *confpath, string(configData))
}

func Run(version string) {
	appVersion = version

	initConfig()

	initLog()

	log.Info("app version: '%v'", version)

	startCenterSession()

	// startUpdateServerListTask()

	startTcpServer()
}

func Stop() {
	stopCenterSession()
	stopTcpServer()

}
