# Render example:
# > go get github.com/noqcks/gucci@1.5.0
# openshift:
# > cat <<EOF > vars.yaml
# MTU: "1160"
# MTU_CHECK_OFFSET: "100"
# POD_NETWORK: "10.128.0.0/14"
# CNO_CONFIG_PATH: "/etc/cno-config"
# NODENAMES:
#   - master-0.c.openshift-gce-devel.internal
#   - master-1.c.openshift-gce-devel.internal
#   - master-2.c.openshift-gce-devel.internal
#   - worker-a-w7vl7
#   - worker-b-stpv6
#   - worker-c-thx84
# EOF
# > ~/go/bin/gucci -f vars.yaml dist/templates/cno-mtu-setter-pod.yaml.tpl
#
# kind:
# > cat <<EOF > vars.yaml
# MTU: "1160"
# MTU_CHECK_OFFSET: "100"
# POD_NETWORK: "10.244.1.0/16"
# NODENAMES:
#   - control-plane
#   - worker1
#   - worker2
# EOF
# > ~/go/bin/gucci -f vars.yaml dist/templates/cno-mtu-setter-pod.yaml.tpl
#
{{- range $node := .NODENAMES }}
---
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: pod-mtu-setter
  name: pod-mtu-setter-{{ randBytes 5 | sha1sum | trunc 5}}
{{- if $.CNO_CONFIG_PATH }}
  namespace: openshift-network-operator
{{- end }}
spec:
  hostNetwork: true
  initContainers:
  - name: cno-pod-mtu-check
    image: quay.io/jcaamano/cno-pod-mtu-setter
    args:
    - --runtime-endpoint
    - $(RUNTIME_ENDPOINT)
    - --pod-network
    - $(POD_NETWORK)
    - --mtu
    - $(MTU)
    - --mtu-check-offset
    - $(MTU_CHECK_OFFSET)
    - --dry-run
    - "true"
    securityContext:
      capabilities:
        add: ["NET_ADMIN", "SYS_ADMIN"]
    env:
    - name: RUNTIME_ENDPOINT
      value: {{ $.RUNTIME_ENDPOINT }}
    - name: POD_NETWORK
      value: {{ $.POD_NETWORK }}
    - name: MTU
      value: "{{ $.MTU }}"
    - name: MTU_CHECK_OFFSET
      value: "{{ $.MTU_CHECK_OFFSET }}"
    volumeMounts:
    - name: runtime-endpoint
      mountPath: {{ $.RUNTIME_ENDPOINT }}
    - name: netns
      mountPath: /var/run/netns
  containers:
  - name: cno-pod-mtu-set
    image: quay.io/jcaamano/cno-pod-mtu-setter
    args:
    - --runtime-endpoint
    - $(RUNTIME_ENDPOINT)
    - --pod-network
    - $(POD_NETWORK)
    - --mtu
    - $(MTU)
    - --mtu-check-offset
    - $(MTU_CHECK_OFFSET)
{{- if $.CNO_CONFIG_PATH }}
    - --cno-config-path
    - $(CNO_CONFIG_PATH)
    - --cno-config-ready-path
    - /tmp/ready
{{- end }}
    securityContext:
      capabilities:
        add: ["NET_ADMIN", "SYS_ADMIN"]
    env:
    - name: RUNTIME_ENDPOINT
      value: {{ $.RUNTIME_ENDPOINT }}
    - name: POD_NETWORK
      value: {{ $.POD_NETWORK }}
    - name: MTU
      value: "{{ $.MTU }}"
    - name: MTU_CHECK_OFFSET
      value: "{{ $.MTU_CHECK_OFFSET }}"
{{- if $.CNO_CONFIG_PATH }}
    - name: CNO_CONFIG_PATH
      value: /run/cno-pod-mtu-setter/{{ $.CNO_CONFIG_PATH }}
{{- end }}
    volumeMounts:
    - name: runtime-endpoint
      mountPath: {{ $.RUNTIME_ENDPOINT }}
    - name: netns
      mountPath: /var/run/netns
{{- if $.CNO_CONFIG_PATH }}
    - name: config-volume
      mountPath: /run/cno-pod-mtu-setter/{{ dir $.CNO_CONFIG_PATH }}
{{- end }}
{{- if $.CNO_CONFIG_PATH }}
    readinessProbe:
      exec:
        command:
          - cat
          - /tmp/ready
      periodSeconds: 1
      failureThreshold: 1
{{- end }}
  volumes:
  - name: runtime-endpoint
    hostPath:
      path: {{ $.RUNTIME_ENDPOINT }}
  - name: netns
    hostPath:
      path: /var/run/netns
{{- if $.CNO_CONFIG_PATH }}
  - name: config-volume
    configMap:
      name: applied-cluster
      items:
      - key: applied
        path: {{ base $.CNO_CONFIG_PATH }}
{{- end }}
  restartPolicy: Never
  nodeName: {{ $node }}
{{- end }}
