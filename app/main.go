package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	daemonsetv1beta1 "github.com/darkowlzz/daemonset-job/pkg/apis/daemonset/v1beta1"
	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

func main() {
	cfg, err := restclient.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := kubernetes.NewForConfigOrDie(cfg)

	watchNamespace := os.Getenv("NAMESPACE")
	if watchNamespace == "" {
		watchNamespace = "default"
	}

	labelSelector := os.Getenv("POD_LABEL_SELECTOR")
	if labelSelector == "" {
		labelSelector = "daemonset-job=true"
	}

	terminationWord := os.Getenv("TERMINATION_WORD")
	if terminationWord == "" {
		labelSelector = "done"
	}

	defaultDuration := time.Duration(10)
	var duration time.Duration

	tickerDuration := os.Getenv("TICKER_DURATION")
	if tickerDuration != "" {
		i, err := strconv.Atoi(tickerDuration)
		if err != nil {
			log.Printf("failed to convert %s to int: %v", tickerDuration, err)
			duration = defaultDuration
		} else {
			duration = time.Duration(i)
		}
	}

	aTimer := time.NewTicker(duration * time.Second)

	for {
		select {
		case _ = <-aTimer.C:
			go checkPods(client, watchNamespace, labelSelector, terminationWord)
		}
	}
}

// getPlainLogs reads the logs from a request and returns the log text as string.
func getPlainLogs(req *restclient.Request) (string, error) {
	var buf bytes.Buffer
	readCloser, err := req.Stream()
	if err != nil {
		return "", err
	}

	defer readCloser.Close()
	_, err = io.Copy(&buf, readCloser)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// checkPods fetches a podList with daemonset-job label and checks log of each
// of the pods for the termination word. If found, the pod is counted as
// completed. Once all the pods have completed their task, the associated
// DaemonSet Job is deleted.
func checkPods(client *kubernetes.Clientset, namespace, labelSelector, terminationWord string) {
	podListOpts := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	pods, err := client.CoreV1().Pods(namespace).List(podListOpts)
	if err != nil {
		log.Println("failed to get podList:", err)
		return
	}

	totalPods := len(pods.Items)
	completedPods := 0

	// Skip if there are no daemonset-job pods.
	if totalPods == 0 {
		return
	}

	opts := &corev1.PodLogOptions{}
	var jobName string

	for _, p := range pods.Items {
		req := client.CoreV1().Pods(namespace).GetLogs(p.GetName(), opts)
		logText, err := getPlainLogs(req)
		if err != nil {
			log.Printf("failed to logs from pod %s: %v", p.GetName(), err)
			// Continue checking other pods.
			continue
		}
		if strings.Contains(logText, terminationWord) {
			completedPods++
		}
		jobName = p.GetLabels()["job"]
	}

	// TODO: Add support for multiple jobs. Job names on all the pods should be
	// considered before deciding job completion.

	if totalPods == completedPods {
		job := &daemonsetv1beta1.Job{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "daemonset.darkowlzz.space/v1beta1",
				Kind:       "Job",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: namespace,
			},
		}
		// Using operator-SDK to make it easy to fetch and delete a custom
		// resource. This dependency should be dropped if the same can be done
		// easily via controller-runtime client. DaemonSet Job's API definition
		// adds dependency on controller-runtime. Can't drop controller-runtime.
		if err := sdk.Get(job); err != nil {
			log.Println("failed to get Job:", err)
			return
		}
		log.Println("Deleting Job", job.GetName())
		if err := sdk.Delete(job); err != nil {
			log.Println("failed to delete Job:", err)
		}
	}
}
