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
		logrus.Errorf("Could not load envconfig: %v", err)
		return
	}

	if config.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.Debugf("Reading volumes from json file %v", config.Volumes)
	file, err := ioutil.ReadFile(config.Volumes)
	if err != nil {
		logrus.Errorf("Could not load volume file: %v", err)
		return
	}

	var volumes []server.Volume
	err = json.Unmarshal(file, &volumes)
	if err != nil {
		logrus.Errorf("Could not unmarshal volume data: %v", err)
		logrus.Errorln(err)
		return
	}

	volumeAdmission := &server.VolumeAdmission{
		Volumes: volumes,
	}

	s, err := server.GetAdmissionControllerServer(volumeAdmission, config.TLSCert, config.TLSKey, config.ListenOn)
	if err != nil {
		logrus.Errorf("Could not create server instance: %v", err)
		return
	}

	logrus.Infof("Starting web server on %v", config.ListenOn)
	err = s.ListenAndServeTLS("", "")
	if err != nil {
		logrus.Errorf("Could not start web server: %v", err)
		return
	}
}
