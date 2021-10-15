module github.com/jcaamano/pod-mtu-setter

go 1.16

require (
	github.com/containernetworking/plugins v1.0.1
	github.com/fsnotify/fsnotify v1.4.9
	github.com/opencontainers/runtime-spec v1.0.3-0.20200929063507-e6143ca7d51d
	github.com/openshift/api v0.0.0-20211014164657-4436dc8be01e
	github.com/pkg/errors v0.9.1
	github.com/urfave/cli/v2 v2.3.0
	github.com/vishvananda/netlink v1.1.1-0.20210330154013-f5de75959ad5
	google.golang.org/grpc v1.41.0
	k8s.io/cri-api v0.22.2
)
