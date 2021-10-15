# Render example:
# > go get github.com/noqcks/gucci@1.5.0
# openshift:
# > ~/go/bin/gucci -s "RUNTIME_ENDPOINT=unix:///run/crio/crio.sock" \
#                  -s "MTU=9000" \
#                  -s "POD_NETWORK=10.128.0.0/14" \
#                  -s "CNO_CONFIG_PATH=/etc/cno-config" \
#                  -s "NODENAME=worker-a" \
#                  dist/templates/cno-mtu-setter-pod.yaml.tpl
# kind:
# > ~/go/bin/gucci -s "RUNTIME_ENDPOINT=unix:///run/containerd/containerd.sock" \
#                  -s "MTU=9000" \
#                  -s "POD_NETWORK=10.244.1.0/16" \
#                  -s "NODENAME=ovn-worker-2" \
#                  dist/templates/cno-mtu-setter-pod.yaml.tpl
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: cno-pod-mtu-setter
  name: cno-pod-mtu-setter
{{- if index . "CNO_CONFIG_PATH" }}
  namespace: openshift-network-operator
{{- end }}
spec:
  hostNetwork: true
  containers:
  - name: cno-pod-mtu-setter
    image: quay.io/jcaamano/cno-pod-mtu-setter
    args:
    - --runtime-endpoint
    - $(RUNTIME_ENDPOINT)
    - --pod-network
    - $(POD_NETWORK)
    - --mtu
    - $(MTU)
{{- if index . "CNO_CONFIG_PATH" }}
    - --cno-config-path
    - $(CNO_CONFIG_PATH)
{{- end }}
    securityContext:
      capabilities:
        add: ["NET_ADMIN", "SYS_ADMIN"]
    env:
    - name: RUNTIME_ENDPOINT
      value: {{ .RUNTIME_ENDPOINT }}
    - name: POD_NETWORK
      value: {{ .POD_NETWORK }}
    - name: MTU
      value: "{{ .MTU }}"
{{- if index . "CNO_CONFIG_PATH" }}
    - name: CNO_CONFIG_PATH
      value: /run/cno-pod-mtu-setter/{{ .CNO_CONFIG_PATH }}
{{- end }}
    volumeMounts:
    - name: runtime-endpoint
      mountPath: {{ .RUNTIME_ENDPOINT }}
    - name: netns
      mountPath: /var/run/netns
{{- if index . "CNO_CONFIG_PATH" }}
    - name: config-volume
      mountPath: /run/cno-pod-mtu-setter/{{ dir .CNO_CONFIG_PATH }}
{{- end }}
  volumes:
  - name: runtime-endpoint
    hostPath:
      path: {{ .RUNTIME_ENDPOINT }}
  - name: netns
    hostPath:
      path: /var/run/netns
{{- if index . "CNO_CONFIG_PATH" }}
  - name: config-volume
    configMap:
      name: applied-cluster
      items:
      - key: applied
        path: {{ base .CNO_CONFIG_PATH }}
{{- end }}
  restartPolicy: Never
  nodeName: {{ .NODENAME }}