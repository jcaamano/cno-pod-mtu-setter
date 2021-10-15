REPO?=quay.io/jcaamano/cno-pod-mtu-setter

build: Dockerfile
	@docker build -t cno-pod-mtu-setter .

push: build
	@docker tag cno-pod-mtu-setter ${REPO}
	@docker push ${REPO}

