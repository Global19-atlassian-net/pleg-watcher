package containerinfo

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rhdedgar/pleg-watcher/containerscan"
	"github.com/rhdedgar/pleg-watcher/dial"
	"github.com/rhdedgar/pleg-watcher/docker"
	"github.com/rhdedgar/pleg-watcher/models"
	"github.com/rhdedgar/pleg-watcher/sender"
)

var (
	// Path is the path to the container runtime interface utility
	Path = "/usr/bin/crictl"
	// UseDocker if crictl not found
	UseDocker = false
)

// ProcessContainer takes a containerID string and retrieves
// info about it from crictl. Then sends the information to
// pod-logger if found.
func ProcessContainer(containerID string) {
	var dCon docker.DockerContainer
	var cCon models.Status

	fmt.Println("ProcessContainer: ", containerID)

	jbyte := dial.CallInfoSrv(containerID, "GetContainerInfo")

	if len(jbyte) > 0 {
		if UseDocker {
			fmt.Println("docker enabled, dCon is:", dCon)
			if err := json.Unmarshal(jbyte, &dCon); err != nil {
				fmt.Println("Error unmarshalling docker output json:", err)
				return
			}
			if strings.HasPrefix(dCon[0].Config.Labels.IoKubernetesPodNamespace, "openshift-") {
				fmt.Println("Container is in openshift-* namespace, skipping")
				return
			} else if dCon[0].State.Status == "running" {
				fmt.Println("container state is running")
				go containerscan.PrepDockerScan(dCon)
				sender.SendDockerData(dCon)
			}
		} else {
			if err := json.Unmarshal(jbyte, &cCon); err != nil {
				fmt.Println("Error unmarshalling crictl output json:", err)
				fmt.Println("bytes look like: ", string(jbyte))
				return
			}
			if strings.HasPrefix(cCon.Status.Labels.IoKubernetesPodNamespace, "openshift-") {
				fmt.Println("Container is in openshift-* namespace, skipping")
				return
			} else if cCon.Status.State == "CONTAINER_RUNNING" {
				go containerscan.PrepCrioScan(cCon)
				sender.SendCrioData(cCon)
			}
		}
	} else {
		fmt.Println("Bytes returned empty")
	}
}

func init() {
	if _, err := os.Stat("/host/usr/bin/crictl"); os.IsNotExist(err) {
		fmt.Println("Cannot find /host/usr/bin/crictl, using /host/usr/bin/docker")
		Path = "/usr/bin/docker"
		UseDocker = true
	}
}
