# k8s-ingress-hosts

Get all the [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) resources from [k8s](https://kubernetes.io/) and update local `/etc/hosts` file. 


## Installation

src
```
go get -u github.com/getditto/k8s-ingress-hosts
```

## Usage

help mode:
```
$ k8s-ingress-hosts -h
Usage of k8s-ingress-hosts:
  -host-file string
    	host file location (default "/etc/hosts")
  -kubeconfig string
    	absolute path to the kubeconfig file
  -version
    	show version and exit
  -watch
    	watch for changes
  -write
    	rewrite host file?
```

dry-run mode:
```
$ k8s-ingress-hosts
# reading k8s ingress resource...
192.168.99.100 grafana.local     # sad-chicken-grafana
192.168.99.100 prometheus.local  # your-turkey-prometheus
```

populate your `/etc/hosts` file:
```
$ sudo -E k8s-ingress-hosts -write
# reading k8s ingress resource...
192.168.99.100 grafana.local     # sad-chicken-grafana
192.168.99.100 prometheus.local  # your-turkey-prometheus

$ cat /etc/hosts
# generated using k8s-ingress-hosts start #
192.168.99.100 grafana.local     # sad-chicken-grafana
192.168.99.100 prometheus.local  # your-turkey-prometheus

# generated using k8s-ingress-hosts end #
```

watch mode:
```
$ k8s-ingress-hosts -watch
# reading k8s ingress resource...
# watching k8s ingress resource...
2021/11/16 12:58:40 ingress added: sad-chicken-grafana
192.168.99.100 grafana.local     # sad-chicken-grafana

2021/11/16 12:59:10 ingress modified: sad-chicken-grafana
192.168.99.100 grafana-sad.local     # sad-chicken-grafana

2021/11/16 12:59:40 ingress deleted: sad-chicken-grafana
```