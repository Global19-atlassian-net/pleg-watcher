package sender

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/rhdedgar/pleg-watcher/docker"
	"github.com/rhdedgar/pleg-watcher/models"
)

var (
	dockerURL = os.Getenv("DOCKER_LOG_URL")
	crioURL   = os.Getenv("CRIO_LOG_URL")
)

// SendDockerData Marshals and POSTs json data to the pod-logger service.
func SendDockerData(dCon docker.DockerContainer) {
	jsonStr, err := json.Marshal(dCon)
	if err != nil {
		fmt.Println("Error marshalling docker json to send to pod-logger: ", err)
	}
	sendLog(jsonStr, dockerURL)
}

// SendCrioData Marshals and POSTs json data to the pod-logger service.
func SendCrioData(mStat models.Status) {
	jsonStr, err := json.Marshal(mStat)
	if err != nil {
		fmt.Println("Error marshalling crio json to send to pod-logger: ", err)
	}
	sendLog(jsonStr, crioURL)
}

func sendLog(jsonStr []byte, url string) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending to pod-logger: ", err)
	}
	defer resp.Body.Close()

	// TODO Prometheus to check header response
	fmt.Println("response Status:", resp.Status)
}
