package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/fsnotify/fsnotify"
	operv1 "github.com/openshift/api/operator/v1"
	"github.com/pkg/errors"
)

// onMTUSet waits until the provided MTU is set at config path
// to invoke the provided callback.
func onMTUSet(configPath string, readyPath string, mtu int, timeout time.Duration, do func() error) error {
	configPath = path.Clean(configPath)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	done := make(chan bool)
	var watchErr error
	go func() {
		for {
			select {
			case <-watcher.Events:
				var actualMTU int
				actualMTU, watchErr = readMTU(configPath)
				if watchErr != nil {
					close(done)
					return
				}

				if actualMTU == mtu {
					log.Printf("Read actual MTU %d matches expected MTU %d, proceeding", actualMTU, mtu)
					watchErr = do()
					close(done)
					return
				}
				log.Printf("Read actual MTU %d does not match expected MTU %d, waiting", actualMTU, mtu)
				

			case err = <-watcher.Errors:
				watchErr = errors.Wrapf(err, "unexpected error waiting for config changes")
				close(done)
				return

			case <-time.After(timeout):
				watchErr = fmt.Errorf("timeout waiting for config changes")
				close(done)
				return
			}

			// watch again to work around:
			// https://github.com/fsnotify/fsnotify/issues/363
			watcher.Remove(configPath)
			err = watcher.Add(configPath)
			if err != nil {
				watchErr = errors.Wrapf(err, "could not watch %s", configPath)
			}
		}
	}()

	watcher.Remove(configPath)
	err = watcher.Add(configPath)
	if err != nil {
		watchErr = errors.Wrapf(err, "could not watch %s", configPath)
	}

	if cnoConfigReadyPath != "" {
		err = createReadyPath(readyPath)
		if err != nil {
			return errors.Wrapf(err, "could not create ready path %s", readyPath)
		}
	}

	log.Printf("Waiting for mtu %d to be set in config %s", mtu, configPath)
	<-done
	return watchErr
}

// readMTU reads the MTU of an OpenshiftSDN or OVNKubernetes
// configuraiton at the provided path
func readMTU(configPath string) (int, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	raw, err := ioutil.ReadAll(file)
	if err != nil {
		return 0, err
	}

	network := operv1.NetworkSpec{}
	err = json.Unmarshal(raw, &network)
	if err != nil {
		return 0, err
	}

	switch network.DefaultNetwork.Type {
	case operv1.NetworkTypeOpenShiftSDN:
		if network.DefaultNetwork.OpenShiftSDNConfig == nil || network.DefaultNetwork.OpenShiftSDNConfig.MTU == nil {
			return 0, fmt.Errorf("MTU value not available for OpenShiftSDN")
		}
		return int(*network.DefaultNetwork.OpenShiftSDNConfig.MTU), nil
	case operv1.NetworkTypeOVNKubernetes:
		if network.DefaultNetwork.OVNKubernetesConfig == nil || network.DefaultNetwork.OVNKubernetesConfig.MTU == nil {
			return 0, fmt.Errorf("MTU value not available for OVNKubernetes")
		}
		return int(*network.DefaultNetwork.OVNKubernetesConfig.MTU), nil
	default:
		return 0, fmt.Errorf("network type not supported: %s", string(network.DefaultNetwork.Type))
	}
}

func createReadyPath(readyPath string) error {
	_, err := os.Stat(readyPath)
	if err == nil {
		return fmt.Errorf("cannot signal readiness with path that already exists: %s", readyPath)
	}
	var file *os.File
	if os.IsNotExist(err) {
		file, err = os.Create(readyPath)
	}
	if err != nil {
		return fmt.Errorf("cannot signal readiness with path %s: %v", readyPath, err)
	}
	file.Close()
	return nil
}
