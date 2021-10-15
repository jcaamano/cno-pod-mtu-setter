package main

import (
	"fmt"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
)

func setVethMTU(nsPath, ifName string, podMTU, hostMTU int) error {
	targetNs, err := ns.GetNS(nsPath)
	if err != nil {
		return err
	}

	var peer int
	err = targetNs.Do(func(hostNs ns.NetNS) error {
		link, err := netlink.LinkByName(ifName)
		if err != nil {
			return err
		}

		veth, ok := link.(*netlink.Veth)
		if !ok {
			return fmt.Errorf("interface %s on namespace %s is not of type veth", ifName, nsPath)
		}

		if veth.MTU != podMTU {
			err = netlink.LinkSetMTU(link, podMTU)
			if err != nil {
				return err
			}
		}

		peer, err = netlink.VethPeerIndex(veth)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	link, err := netlink.LinkByIndex(peer)
	if err != nil {
		return err
	}

	if link.Attrs().MTU != hostMTU {
		err = netlink.LinkSetMTU(link, hostMTU)
		if err != nil {
			return err
		}
	}

	return nil
}