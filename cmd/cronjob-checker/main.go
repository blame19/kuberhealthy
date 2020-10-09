package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	kh "github.com/Comcast/kuberhealthy/v2/pkg/checks/external/checkclient"
	"github.com/Comcast/kuberhealthy/v2/pkg/checks/external/nodeCheck"
	"github.com/Comcast/kuberhealthy/v2/pkg/kubeClient"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// kubeConfigFile is a variable containing file path of Kubernetes config files
var kubeConfigFile = filepath.Join(os.Getenv("HOME"), ".kube", "config")

// cronJob name
var cronJob = os.Getenv("CRONJOB")

// namespace is a variable to allow code to target all namespaces or a single namespace
var namespace = os.Getenv("NAMESPACE")

// reason is a varaible to search for event types
var reason = os.Getenv("REASON")

func init() {
	// set debug mode for nodeCheck pkg
	nodeCheck.EnableDebugOutput()
}

func main() {

	// create context
	checkTimeLimit := time.Minute * 1
	ctx, _ := context.WithTimeout(context.Background(), checkTimeLimit)

	// create kubernetes client
	kubernetesClient, err := kubeClient.Create("")
	if err != nil {
		log.Errorln("Error creating kubeClient with error" + err.Error())
	}

	// hits kuberhealthy endpoint to see if node is ready
	err = nodeCheck.WaitForKuberhealthy(ctx)
	if err != nil {
		log.Errorln("Error waiting for kuberhealthy endpoint to be contactable by checker pod with error:" + err.Error())
	}

	// fetches kube proxy to see if it is ready
	err = nodeCheck.WaitForKubeProxy(ctx, kubernetesClient, "kuberhealthy", "kube-system")
	if err != nil {
		log.Errorln("Error waiting for kube proxy to be ready and running on the node with error:" + err.Error())
	}

	// //create events client
	client := kubernetesClient.EventsV1beta1()

	//retrive events from namespace
	log.Infoln("Begining to retrieve events from cronjob " + cronJob)
	e := client.Events(namespace)

	listOpts := v1.ListOptions{}
	eventList, err := e.List(context.TODO(), listOpts)
	if err != nil {
		log.Fatalln("Error listing events with error:", err)
	}

	//range over eventList for cronjob events that match provided reason
	for _, e := range eventList.Items {
		// log.Debugln(e.Reason)
		if reason == e.Reason {
			log.Infoln("There was an event with reason:" + e.Reason + " for cronjob" + cronJob)
			reportErr := fmt.Errorf("cronjob: " + cronJob + " has an event with reason: " + reason)
			ReportFailureAndExit(reportErr)
		}
	}

	log.Infoln("Success! There were no events with reason " + reason + " for cronjob " + cronJob)
	err = kh.ReportSuccess()
	if err != nil {
		log.Fatalln("error when reporting to kuberhealthy:", err.Error())
	}
	log.Infoln("Successfully reported to Kuberhealthy")
}

// ReportFailureAndExit logs and reports an error to kuberhealthy and then exits the program.
// If a error occurs when reporting to kuberhealthy, the program fatals.
func ReportFailureAndExit(err error) {
	// log.Errorln(err)
	err2 := kh.ReportFailure([]string{err.Error()})
	if err2 != nil {
		log.Fatalln("error when reporting to kuberhealthy:", err.Error())
	}
	log.Infoln("Succesfully reported error to kuberhealthy")
	os.Exit(0)
}
