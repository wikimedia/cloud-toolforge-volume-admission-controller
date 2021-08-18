package main

import (
	"gerrit.wikimedia.org/cloud/toolforge/volume-admission-controller/server"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
)

type Config struct {
	ListenOn string   `default:"0.0.0.0:8080"`
	TLSCert  string   `default:"/etc/webhook/certs/cert.pem"`
	TLSKey   string   `default:"/etc/webhook/certs/key.pem"`
	Volumes  string   `default:"/etc/volumes.json"`
	Debug    bool     `default:"true"`
}

func main() {
	config := &Config{}
	err := envconfig.Process("", config)
	if err != nil {
		logrus.Errorln(err)
		return
	}

	if config.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.Debugln("Reading volumes from json file")
	file, err := ioutil.ReadFile(config.Volumes)
	if err != nil {
		logrus.Errorln(err)
		return
	}

	var volumes []server.Volume
	err = json.Unmarshal(file, &volumes)
	if err != nil {
		logrus.Errorln(err)
		return
	}

	volumeAdmission := &server.VolumeAdmission{
		Volumes: volumes,
	}

	s, err := server.GetAdmissionControllerServer(volumeAdmission, config.TLSCert, config.TLSKey, config.ListenOn)
	if err != nil {
		logrus.Errorln(err)
		return
	}

	err = s.ListenAndServeTLS("", "")
	if err != nil {
		logrus.Errorln(err)
		return
	}
}
