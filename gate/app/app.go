package app

import (
	"encoding/xml"
	"flag"
	"github.com/nothollyhigh/kiss/log"
	"io"
	"io/ioutil"
	"os"
	"time"
)

var (
	appVersion = ""
	bornTime   = time.Now() /* 进程启动时间 */
	confpath   = flag.String("config", "./conf/gate.xml", "config file path, default is ./conf/gate.xml")

	logout = io.Writer(nil)
)

func initConfig() {
	flag.Parse()

	filename := *confpath

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("Error when Open xml config file: %s: %v\n", filename, err)
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal("Error when Read xml config file: %s: %v\n", filename, err)
	}

	err = xml.Unmarshal(data, xmlconfig)
	if err != nil {
		log.Fatal("Error when xml.Unmarshal from xml config file: %s: %v\n    data: %s\n", filename, err, string(data))
	}

	log.Info("config: %v\n%v", filename, string(data))
}

func initLog() {
	var (
		fileWriter = &log.FileWriter{
			RootDir:     xmlconfig.Options.LogDir + time.Now().Format("20060102150405/"),
			DirFormat:   "",
			FileFormat:  "20060102.log",
			MaxFileSize: 1024 * 1024 * 32,
			EnableBufio: false,
		}
	)
	if xmlconfig.Options.Debug {
		logout = io.MultiWriter(os.Stdout, fileWriter)
	} else {
		logout = fileWriter
	}

	log.SetOutput(logout)
}

func Run(version string) {
	appVersion = version

	initConfig()

	initLog()

	log.Info("kissgate start, app version: '%v'", version)

	proxyMgr.InitPorxy()

	connMgr.StartDataFlowRecord(time.Second * 60)
}

func Stop() {
	log.Info("kissgate stop")
}
