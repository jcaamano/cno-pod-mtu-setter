package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	podIfname = "eth0"
	maxMTU    = 65000
	minMTU    = 576
)

var (
	mtu                int
	podNetwork         *net.IPNet
	runtimeEndpoint    string
	cnoConfigPath      string
	cnoConfigReadyPath string
	timeout            time.Duration
	dryRun             bool
	checkDev           string
	checkOffset        int
	start              time.Time
)

func main() {
	app := &cli.App{
		Name:  "cno-pod-mtu-setter",
		Usage: "change the mtu of pods",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "runtime-endpoint",
				Usage:       "Endpoint of CRI container runtime service",
				DefaultText: fmt.Sprintf("first available from %v", defaultRuntimeEndpoints),
			},
			&cli.IntFlag{
				Name:        "mtu",
				Usage:       "`MTU` value to set",
				Required:    true,
				DefaultText: "no default, required",
			},
			&cli.StringFlag{
				Name:  "pod-network",
				Usage: "If provided, only perform the change for pods within this pod `network`",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Don't actually change the MTU",
			},
			&cli.StringFlag{
				Name:  "cno-config-path",
				Usage: "If provided, wait until cno config at `path` is updated with the MTU value",
			},
			&cli.StringFlag{
				Name:  "cno-config-ready-path",
				Usage: "If provided, create file at `path` to signal that cno-config-path is being watched",
			},
			&cli.IntFlag{
				Name:        "cno-config-timeout",
				Usage:       "`Duration` to wait for cno config change in seconds",
				DefaultText: "Unlimited",
			},
			&cli.StringFlag{
				Name:        "mtu-check-offset",
				Usage:       "Check that the provided MTU has minimum `offset` with respect default gateway interface MTU",
				DefaultText: "0",
			},
			&cli.StringFlag{
				Name:        "mtu-check-dev",
				Usage:       "Check MTU offset for the provided `interface` MTU",
				DefaultText: "default gateway interface",
			},
		},
		Before: func(c *cli.Context) error {
			start = time.Now()
			runtimeEndpoint = c.String("runtime-endpoint")
			cnoConfigPath = c.String("cno-config-path")
			cnoConfigReadyPath = c.String("cno-config-ready-path")
			checkDev = c.String("mtu-check-dev")
			checkOffset = c.Int("mtu-check-offset")
			dryRun = c.Bool("dry-run")

			mtu = c.Int("mtu")
			if mtu < minMTU || mtu > maxMTU {
				return fmt.Errorf("invalid mtu value %d, not between [%d,%d]", mtu, minMTU, maxMTU)
			}

			_, ipNet, err := net.ParseCIDR(c.String("pod-network"))
			if err != nil {
				return fmt.Errorf("invalid pod network: %v", err)
			}
			podNetwork = ipNet

			timeout = time.Second * time.Duration(c.Int("cno-config-timeout"))
			if timeout.Seconds() == 0 {
				timeout = time.Duration(math.MaxInt64)
			}

			return nil
		},
		Action: func(context *cli.Context) error {
			err := checkMTU(mtu, checkOffset, checkDev)
			if err != nil {
				return err
			}
			if cnoConfigPath != "" {
				return onMTUSet(cnoConfigPath, cnoConfigReadyPath, mtu, timeout, func() error {
					return setPodsMTU(mtu, podNetwork)
				})
			} else {
				return setPodsMTU(mtu, podNetwork)
			}
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

func setPodsMTU(mtu int, podNetwork *net.IPNet) error {
	defer timeTrack(time.Now(), "Setting pods MTU")
	intention := "Changing"
	if dryRun {
		intention = "Would change"
	}
	return forEveryPod(func(pod *podStatus) error {
		if start.Before(time.Unix(0, pod.CreatedAt)) {
			log.Printf("Ignoring pod %s in namespace %s, created after start time of %s",  pod.namespacedName(), pod.networkNamespace(), start.String())
			return nil
		}
		if pod.State != v1alpha2.PodSandboxState_SANDBOX_READY {
			log.Printf("Ignoring pod %s in namespace %s, invalid state %s", pod.namespacedName(), pod.networkNamespace(), v1alpha2.PodSandboxState_name[int32(pod.State)])
			return nil
		}
		if podNetwork != nil && !pod.isOnNetwork(podNetwork) {
			log.Printf("Ignoring pod %s in namespace %s, not in pod network %s", pod.namespacedName(), pod.networkNamespace(), podNetwork.String())
			return nil
		}
		log.Printf("%s MTU to %d for pod %s in namespace %v\n", intention, mtu, pod.namespacedName(), pod.networkNamespace())
		err := setVethMTU(pod.networkNamespace(), podIfname, mtu, maxMTU)
		return errors.Wrapf(err, "failed to set MTU on pod with status %v", *pod)
	})
}

func checkMTU(mtu, offset int, dev string) error {
	var devMTU int
	var err error
	if dev == "" {
		devMTU, dev, err = getDefaultMTU()
	} else {
		devMTU, err = getMTU(dev)
	}
	if err != nil {
		errors.Wrapf(err, "can't get MTU value for provided device")
	}
	if devMTU < mtu+offset {
		return fmt.Errorf("MTU value %d is invalid as is above %s MTU %d with offset %d", mtu, dev, devMTU, offset)
	}
	log.Printf("MTU value %d is within %s MTU %d with offset %d", mtu, dev, devMTU, offset)
	return nil
}
