package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	operv1 "github.com/openshift/api/operator/v1"
)

func onMTUSet(configPath string, mtu int, timeout time.Duration, do func() error) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					err = fmt.Errorf("unexpected error waiting for config changes")
					close(done)
					return
				}
				
				actualMTU, err := readMTU(configPath)
				if err != nil {
					close(done)
					return
				}
				
				if (actualMTU == mtu) {
					do()
					close(done)
					return
				}
			case err = <-watcher.Errors:
				err = fmt.Errorf("unexpected error waiting for config changes: %v", err)
				close(done)
				return
			case <-time.After(timeout):
				err = fmt.Errorf("timeout waiting for config changes")
				close(done)
				return
			}
		}
	}()

	log.Printf("Waiting for mtu %d to be set in config %s", mtu, configPath)
	err = watcher.Add(configPath)
	if err != nil {
		return err
	}

	<-done
	return err
}

func readMTU(configPath string) (int, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return 0, err
	}
	
	raw, err := ioutil.ReadAll(file)
	if err != nil {
		return 0, err
	}

	network := operv1.Network{}
	err = json.Unmarshal(raw, &network)
	if err != nil {
		return 0, err
	}

	switch network.Spec.DefaultNetwork.Type {
	case operv1.NetworkTypeOpenShiftSDN:
		if network.Spec.DefaultNetwork.OpenShiftSDNConfig == nil || network.Spec.DefaultNetwork.OpenShiftSDNConfig.MTU == nil {
			return 0, fmt.Errorf("MTU value not available for OpenShiftSDN")
		}
		return int(*network.Spec.DefaultNetwork.OpenShiftSDNConfig.MTU), nil
	case operv1.NetworkTypeOVNKubernetes:
		if network.Spec.DefaultNetwork.OVNKubernetesConfig == nil || network.Spec.DefaultNetwork.OVNKubernetesConfig.MTU == nil {
			return 0, fmt.Errorf("MTU value not available for OVNKubernetes")
		}
		return int(*network.Spec.DefaultNetwork.OVNKubernetesConfig.MTU), nil
	default:
		return 0, fmt.Errorf("network type not supported: %s", string(network.Spec.DefaultNetwork.Type))
	}
}