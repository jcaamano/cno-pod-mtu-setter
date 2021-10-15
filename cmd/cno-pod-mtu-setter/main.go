package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

const (
	podIfname = "eth0"
	maxMTU    = 65000
)

var (
	mtu             int
	podNetwork      *net.IPNet
	runtimeEndpoint string
	cnoConfigPath   string
	timeout         time.Duration
)

func main() {
	app := &cli.App{
		Name:  "pod-mtu-setter",
		Usage: "change the mtu of pods",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "runtime-endpoint",
				Usage:       "Endpoint of CRI container runtime service",
				DefaultText: fmt.Sprintf("first available from %v", defaultRuntimeEndpoints),
			},
			&cli.StringFlag{
				Name:     "pod-network",
				Usage:    "For pods within this pod network",
				Required: true,
			},
			&cli.IntFlag{
				Name:  "mtu",
				Usage: "`MTU` value to set",
			},
			&cli.StringFlag{
				Name:  "cno-config-path",
				Usage: "If provided, wait until cno config mtu is changed to `MTU` to change pods MTU",
			},
			&cli.IntFlag{
				Name:  "cno-config-timeout",
				Usage: "If provided, amount of time to wait for cno config change in seconds",
			},
		},
		Before: func(c *cli.Context) error {
			runtimeEndpoint = c.String("runtime-endpoint")
			mtu = c.Int("mtu")
			cnoConfigPath = c.String("cno-config-path")

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
			defer timeTrack(time.Now(), "pod-mtu-setter")
			if cnoConfigPath != "" {
				return onMTUSet(cnoConfigPath, mtu, timeout, func() error {
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
	return forEveryPod(func(pod *podStatus) error {
		if pod.isOnNetwork(podNetwork) {
			log.Printf("Changing to MTU %d for pod %s in namespace %v\n", mtu, pod.namespacedName(), pod.networkNamespace())
			return setVethMTU(pod.networkNamespace(), podIfname, mtu, maxMTU)
		}
		return nil
	})
}
