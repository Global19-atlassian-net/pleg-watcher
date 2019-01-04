package watcher

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
)

type PLEGEvent struct {
	ID   string `json:"ID"`
	Type string `json:"Type"`
	Data string `json:"Data"`
}

type PLEGBuffer struct {
	bLine bytes.Buffer
}

//func (p *PLEGBuffer) Write(w io.Writer) (int, error) {
//b, _ := json.Marshal(*p)
//w.Write(b)
//}

func quoteVar(s string, r string) string {
	return strings.Replace(s, r, "\""+r+"\"", 1)
}

func ProcessContainer(inputStr string) {
	fmt.Println(inputStr)
}

func CheckOutput(inputStr string) string {
	var plegEvent PLEGEvent

	if strings.Contains(inputStr, "ContainerStarted") {

		// Gather only the unquoted json of the PLEG Event
		out := strings.SplitAfter(inputStr, "&pleg.PodLifecycleEvent")[1]

		// Quote the json so it can be Unmarshaled into a struct
		for _, item := range []string{"ID", "Type", "Data"} {
			out = quoteVar(out, item)
		}

		if err := json.Unmarshal([]byte(out), &plegEvent); err != nil {
			fmt.Println("error unmarshaling json: ", err)
		}

		fmt.Println("Data key:\n", plegEvent.Data)

		return plegEvent.Data
	}
	fmt.Println("Not a creation event, skipping")
	return ""
}

func PLEGWatch() {
	//var buffer PLEGBuffer //bytes.Buffer

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	reader := bufio.NewReader(&b)
	path := "/var/log/journal"

	jcfg := sdjournal.JournalReaderConfig{
		NumFromTail: 10,
		Path:        path,
		Matches: []sdjournal.Match{
			{
				Field: sdjournal.SD_JOURNAL_FIELD_SYSLOG_IDENTIFIER,
				Value: "atomic-openshift-node",
			},
		},
	}

	jr, err := sdjournal.NewJournalReader(jcfg)
	if err != nil {
		log.Printf("[ERROR] journal: %v", err)
		return
	}
	defer jr.Close()

	//log.Printf("[INFO] journal: show last %v entries for %v", h.cfg.JournalNumEntries, name)
	fmt.Println("=== begin journal ===")
	//uTime := time.Date(2100, 11, 17, 0, 0, 0, 0, time.UTC)

	until := make(chan time.Time)

	jr.Follow(until, writer) //buffer)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
		event := CheckOutput(scanner.Text())
		if event != "" {
			go ProcessContainer(event)
		}
	}
	// shouldn't reach this block
	fmt.Println("=== end journal ===")

	//	if err := scanner.Err(); err != nil {
	//		log.Printf("[ERROR] journal: %v", err)
	//		return
	//	}
}
