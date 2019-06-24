package containerscan

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/rhdedgar/pleg-watcher/channels"
	clscmd "github.com/rhdedgar/pleg-watcher/cmd"
	"github.com/rhdedgar/pleg-watcher/models"
	"github.com/rhdedgar/pleg-watcher/runcspec"
	mainscan "github.com/rhdedgar/pleg-watcher/scanner"
)

var (
	scanResultsDir = os.Getenv("SCANRESULTSDIR")
	postResultURL  = os.Getenv("POSTRESULTURL")
	outFile        = os.Getenv("OUTFILE")
)

// custSplit takes 3 parameters and returns a string.
// s is the string to split.
// d is the delimiter by which to split s.
// i is the slice index of the string to return, if applicable. Usually 1 or 0.
// If the string was not split, the original string is returned idempotently.
func custSplit(s, d string, i int) string {
	tempS := s
	splits := strings.Split(s, d)

	if len(splits) >= i+1 {
		tempS = splits[i]
	}

	return tempS
}

// custReg takes 2 arguments and returns a string slice.
//
// scanOut is the string output from the crio /proc/$PID/mountinfo file.
//
// regString is the `raw string` containing the regex match pattern to use.
func custReg(scanOut, regString string) []string {
	var newLayers []string

	reg := regexp.MustCompile(regString)
	matched := reg.FindAllString(scanOut, -1)

	if matched != nil {
		for _, layer := range matched {
			newLayers = append(newLayers, layer)
		}
	}

	return newLayers
}

func getCrioLayers(containerID string) []string {
	var layers []string
	var crioLayers []string
	var runcState runcspec.RuncState

	fmt.Println("inspecting: ", containerID)

	go channels.SetStringChan(models.ChrootChan, containerID)

	select {
	case jbyte := <-models.ChrootOut:

		if err := json.Unmarshal(jbyte, &runcState); err != nil {
			fmt.Println("Error unmarshalling crictl output json:", err)
			return crioLayers
		}

		pid := runcState.Pid
		//rootPath := runcState.RootFS
		//dirPath := filepath.Dir(rootPath)
		//IDPath := filepath.Base(rootPath)

		mountPath := "/proc/" + string(pid) + "/mountinfo"
		//mountOutput := ""

		f, err := os.Open(mountPath)
		if err != nil {
			fmt.Println("error opening file:", mountPath, err)
		}

		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Scan()
		scanOut := scanner.Text()

		if err := scanner.Err(); err != nil {
			fmt.Println("Error reading layer", err)
		}

		layers = append(layers, custReg(scanOut, `lowerdir=(.*),upperdir`)...)
		layers = append(layers, custReg(scanOut, `upperdir=(.*),workdir`)...)

		for _, l := range layers {
			items := strings.Split(l, ":")
			for _, i := range items {
				j := custSplit(i, ",", 0)
				j = custSplit(j, "=", 1)

				crioLayers = append(crioLayers, j)
			}
		}
		return crioLayers

	default:
		return crioLayers
	}
}

// PrepCrioScan gets a slice of container filesystem layers from getCrioLayers
// and then initiates a scan for each of the returned layers.
func PrepCrioScan(cCon models.Status) {
	scannerOptions := clscmd.NewDefaultContainerLayerScannerOptions()
	cID := cCon.Status.ID

	cLayers := getCrioLayers(cID)

	if len(cLayers) == 0 {
		fmt.Println("layers returned empty")
		return
	}

	scannerOptions.ScanResultsDir = scanResultsDir //"./"
	scannerOptions.PostResultURL = postResultURL   // "127.0.0.1"
	scannerOptions.OutFile = outFile               // "clamav_scan_results.log"

	fmt.Println("Scanning layers", cLayers)

	for _, l := range cLayers {
		scannerOptions.ScanDir = l

		if err := scannerOptions.Validate(); err != nil {
			fmt.Println("error validating scanner options", err)
		}

		scanner := mainscan.NewDefaultContainerLayerScanner(*scannerOptions)
		if err := scanner.ClamScanner(); err != nil {
			fmt.Println("error creating new scanner", err)
		}
	}
}