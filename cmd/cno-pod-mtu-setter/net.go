package main

import (
	"fmt"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

// setVethMTU sets the MTU of the provided veth interface at
// the provided namespace path, as well as the peer MTU.
func setVethMTU(nsPath, name string, nsMTU, peerMTU int) error {
	targetNs, err := ns.GetNS(nsPath)
	if err != nil {
		return errors.Wrapf(err, "could not get namespace %s", nsPath)
	}

	var peer int
	err = targetNs.Do(func(hostNs ns.NetNS) error {
		link, err := netlink.LinkByName(name)
		if err != nil {
			return errors.Wrapf(err, "could not get link with name %s", name)
		}

		veth, ok := link.(*netlink.Veth)
		if !ok {
			return fmt.Errorf("interface %s on namespace %s is not of type veth", name, nsPath)
		}

		if veth.MTU != nsMTU && !dryRun {
			err = netlink.LinkSetMTU(link, nsMTU)
			if err != nil {
				return errors.Wrapf(err, "could not set mtu %d on %s", nsMTU, name)
			}
		}

		peer, err = netlink.VethPeerIndex(veth)
		if err != nil {
			return errors.Wrapf(err, "could not get veth peer for %s", name)
		}

		return nil
	})
	if err != nil {
		return err
	}

	link, err := netlink.LinkByIndex(peer)
	if err != nil {
		return errors.Wrapf(err, "could not get link with index %d", peer)
	}

	if link.Attrs().MTU != peerMTU && !dryRun {
		err = netlink.LinkSetMTU(link, peerMTU)
		if err != nil {
			return errors.Wrapf(err, "could not set mtu %d on %s", peerMTU, link.Attrs().Name)
		}
	}

	return nil
}

// getDefaultMTU gets the mtu of the default route.
func getDefaultMTU() (int, string, error) {
	// Get the interface with the default route
	// TODO(cdc) handle v6-only nodes
	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return 0, "", errors.Wrapf(err, "could not list routes")
	}
	if len(routes) == 0 {
		return 0, "", errors.Errorf("got no routes")
	}

	const maxMTU = 65536
	mtu := maxMTU + 1
	var name string
	for _, route := range routes {
		// Skip non-default routes
		if route.Dst != nil {
			continue
		}
		link, err := netlink.LinkByIndex(route.LinkIndex)
		if err != nil {
			return 0, "", errors.Wrapf(err, "could not retrieve link id %d", route.LinkIndex)
		}

		newmtu := link.Attrs().MTU
		if newmtu > 0 && newmtu < mtu {
			mtu = newmtu
			name = link.Attrs().Name
		}
	}
	if mtu > maxMTU {
		return 0, "", errors.Errorf("unable to determine MTU")
	}

	return mtu, name, nil
}

// getMTU gets the mtu of an interface.
func getMTU(dev string) (int, error) {
	link, err := netlink.LinkByName(dev)
	if err != nil {
		return 0, errors.Wrapf(err, "could not retrieve link %s", dev)
	}
	return link.Attrs().MTU, nil
}
