package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"

	v1 "k8s.io/api/networking/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiWatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

var (
	k8sHostname   string
	versionUrl    = "https://github.com/getditto/k8s-ingress-hosts"
	version       = "dev"
	hostFile      = flag.String("host-file", "/etc/hosts", "host file location")
	writeHostFile = flag.Bool("write", false, "rewrite host file?")
	showVersion   = flag.Bool("version", false, "show version and exit")
	kubeconfig    = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	watch         = flag.Bool("watch", false, "watch for changes")
)

const (
	sectionStart = "# generated using k8s-ingress-hosts start #"
	sectionEnd   = "# generated using k8s-ingress-hosts end #\n"
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}

	return os.Getenv("USERPROFILE")
}

type Rule struct {
	Domain  string
	Service string
}

func (r *Rule) String() string { return fmt.Sprintf("%s %s\t# %s", k8sHostname, r.Domain, r.Service) }

type HostsList []Rule

func (h HostsList) Len() int      { return len(h) }
func (h HostsList) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h HostsList) Less(i, j int) bool {
	return strings.ToLower(h[i].Domain) < strings.ToLower(h[j].Domain)
}

func k8sHost(config *rest.Config) string {
	u, err := url.Parse(config.Host)
	if err != nil {
		log.Fatalln(err.Error())
	}

	return u.Hostname()
}

func tryWriteToHostFile(hostEntries string) error {

	block := []byte(fmt.Sprintf("%s\n%s\n%s", sectionStart, hostEntries, sectionEnd))
	fileContent, err := ioutil.ReadFile(*hostFile)
	if err != nil {
		return err
	}

	re := regexp.MustCompile(fmt.Sprintf("(?ms)%s(.*)%s", sectionStart, sectionEnd))
	if re.Match(fileContent) {
		fileContent = re.ReplaceAll(fileContent, block)
	} else {
		fileContent = append(fileContent, block...)
	}

	if err := ioutil.WriteFile(*hostFile, fileContent, 0644); err != nil {
		return err
	}

	fmt.Println(hostEntries)
	return nil
}

func sortAndWrite(entries HostsList) {
	sort.Sort(HostsList(entries))

	var hostEntries string
	for _, item := range entries {
		hostEntries = hostEntries + fmt.Sprintf("%s\n", item.String())
	}

	wBuffer := new(bytes.Buffer)
	writer := tabwriter.NewWriter(wBuffer, 0, 0, 2, ' ', 0)
	fmt.Fprint(writer, hostEntries)
	writer.Flush()

	if !*writeHostFile {
		fmt.Println(wBuffer.String())
		return
	}

	if err := tryWriteToHostFile(wBuffer.String()); err != nil {
		log.Fatalln(err)
	}
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("Glasses\n url: %s\n version: %s", versionUrl, version)
		os.Exit(2)
	}

	fmt.Println("# reading k8s ingress resource...")
	if *kubeconfig == "" {
		*kubeconfig = fmt.Sprintf("%s/.kube/config", homeDir())
	}
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatalln(err.Error())
	}

	k8sHostname = k8sHost(config)
	if addr := net.ParseIP(k8sHostname); addr == nil {
		lookupResults, err := net.LookupHost(k8sHostname)
		if err != nil {
			log.Fatalf("k8s hostname %s not found", k8sHostname)
		}
		k8sHostname = lookupResults[0]
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err.Error())
	}

	ingresses, err := client.NetworkingV1().Ingresses("").List(context.Background(), metaV1.ListOptions{})
	if err != nil {
		log.Fatalln(err.Error())
	}

	var entries HostsList
	for _, elem := range ingresses.Items {
		for _, rule := range elem.Spec.Rules {
			entries = append(entries, Rule{
				Domain:  rule.Host,
				Service: elem.Name,
			})
		}
	}

	sortAndWrite(entries)

	if *watch {
		fmt.Println("# watching k8s ingress resource...")
		watcher, err := client.NetworkingV1().Ingresses("").Watch(context.Background(), metaV1.ListOptions{})
		if err != nil {
			log.Fatalln(err)
		}

		for {
			select {
			case event := <-watcher.ResultChan():
				switch event.Type {
				case apiWatch.Added:
					newIngress := event.Object.(*v1.Ingress)
					log.Printf("ingress added: %s", newIngress.Name)
					for _, rule := range newIngress.Spec.Rules {
						entries = append(entries, Rule{
							Domain:  rule.Host,
							Service: newIngress.Name,
						})
					}
				case apiWatch.Modified:
					updatedIngress := event.Object.(*v1.Ingress)
					log.Printf("ingress modified: %s", updatedIngress.Name)
					for i, entry := range entries {
						if entry.Service == updatedIngress.Name {
							entries[i].Domain = updatedIngress.Spec.Rules[0].Host
						}
					}
				case apiWatch.Deleted:
					deletedIngress := event.Object.(*v1.Ingress)
					log.Printf("ingress deleted: %s", deletedIngress.Name)
					for _, rule := range deletedIngress.Spec.Rules {
						for i, entry := range entries {
							if entry.Service == deletedIngress.Name && entry.Domain == rule.Host {
								entries = append(entries[:i], entries[i+1:]...)
								break
							}
						}
					}
				}
				sortAndWrite(entries)
			}
		}
	}

}
