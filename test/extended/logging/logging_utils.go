package logging

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"

	//"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	//apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/utils"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 400
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

// OperatorObjects objects for creating operators via OLM
type OperatorObjects struct {
	Namepsace     string
	Operatorgroup string
	Csc           string
	Sub           string
	//	Rbac          string
}

// CreateLoggingResources creating the resources
func createLoggingResources(oo OperatorObjects) error {
	var oc = exutil.NewCLIWithoutNamespace("logging")
	filenames := reflect.ValueOf(oo)
	t := reflect.TypeOf(oo)
	num := filenames.NumField()
	for i := 0; i < num; i++ {
		filename := filenames.Field(i).Interface()
		name := fmt.Sprint(filename)
		fmt.Printf("Creating %s ...\n", t.Field(i).Name)
		err := oc.AsAdmin().Run("create").Args("-f", name).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		if err != nil {
			return err
		}
	}
	return nil
}

func waitForOperatorToBeReady(namespace string, name string, retryInterval, timeout time.Duration) error {
	var oc = exutil.NewCLIWithoutNamespace("logging")
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		deployment, err := oc.AdminKubeClient().AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				e2e.Logf("Waiting for availability of %s deployment\n", name)
				return false, nil
			}
			return false, err
		}

		if int(deployment.Status.AvailableReplicas) == int(deployment.Status.Replicas) {
			replicas := int(deployment.Status.Replicas)
			e2e.Logf("Deployment %s available (%d/%d)\n", name, replicas, replicas)
			return true, nil
		}
		e2e.Logf("Waiting for full availability of %s deployment (%d/%d)\n", name, deployment.Status.AvailableReplicas, deployment.Status.Replicas)
		return false, nil
	})
	if err != nil {
		return err
	}
	return nil
	/*
		var oc = exutil.NewCLIWithoutNamespace("logging")
		err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
				output, err := oc.AsAdmin().Run("get").Args("-n", namespace, "deploy", name).Output()
				if err != nil {
					errstring := fmt.Sprintf("%v", output)
					if strings.Contains(errstring, "NotFound") {
						e2e.Logf("Waiting for availability of %s deploy ...\n", name)
						return false, nil
					}
					return false, err
				}
				ars, _ := oc.AsAdmin().Run("get").Args("-n", namespace, "deploy", name, "--output=jsonpath=\"{.status.availableReplicas}\"").Output()
				rs, _ := oc.AsAdmin().Run("get").Args("-n", namespace, "deploy", name, "--output=jsonpath=\"{.status.replicas}\"").Output()

				if len(ars) != 0 && ars == rs {
					return true, nil
				}
				e2e.Logf("Waiting for full availability of %s \n", name)
				return false, nil
			})
			if err != nil {
				return err
			}
			e2e.Logf("Operator %s is available \n", name)
			return nil
	*/
}

// ClearResources to delete objects in the cluster
func clearResources(resourcetype string, name string, ns string) error {
	var oc = exutil.NewCLIWithoutNamespace("logging")
	msg, err := oc.AsAdmin().Run("delete").Args("-n", ns, resourcetype, name).Output()
	if err != nil {
		errstring := fmt.Sprintf("%v", msg)
		if strings.Contains(errstring, "NotFound") {
			return nil
		}
		return err
	}
	return nil
}

// DeleteNamespace delete specific namespace
func deleteNamespace(ns string) error {
	var oc = exutil.NewCLIWithoutNamespace("logging")
	/*
		msg, err := oc.AsAdmin().Run("delete").Args("project", ns).Output()
		if err != nil {
			errstring := fmt.Sprintf("%v", msg)
			if strings.Contains(errstring, "NotFound") {
				return nil
			}
			return err
		}
		return nil
	*/
	// err := oc.AdminKubeClient().CoreV1().Namespaces().Delete(ns, nil)
	err := oc.AdminKubeClient().CoreV1().Namespaces().Delete(ns, &metav1.DeleteOptions{})
	//err := oc.AdminKubeClient().CoreV1().Namespaces().Delete(ns, metav1.NewDeleteOptions(0))
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// CheckLoggingPodsRunning check the EFK pods running or not, for pods have >1 containers, need to find a new way to check
func checkLoggingPodsRunning(ns string, label string) error {
	Label := exutil.ParseLabelsOrDie(label)
	var oc = exutil.NewCLIWithoutNamespace("logging")
	err := wait.Poll(retryInterval, timeout, func() (bool, error) {
		err := utils.WaitForPodsWithLabelRunning(oc.AdminKubeClient(), ns, Label)
		if err != nil {
			e2e.Logf("Failed getting pods: %v", err)
			return false, nil // Ignore this error (nil) and try again in "Poll" time
		}
		return true, nil
	})
	return err
}

// crd
func checkResourcesCreatedByOperators(ns string, resourcetype string, name string) error {
	var oc = exutil.NewCLIWithoutNamespace("")
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		output, err := oc.AsAdmin().Run("get").Args("-n", ns, resourcetype, name).Output()
		if err != nil {
			msg := fmt.Sprintf(output)
			if strings.Contains(msg, "NotFound") {
				return false, nil
			}
			return false, err
		}
		e2e.Logf("Find %s %s", resourcetype, name)
		return true, nil
	})
	return err
}

func checkCronJob(name string, ns string) error {
	var oc = exutil.NewCLIWithoutNamespace("")
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		_, err = oc.AdminKubeClient().BatchV1beta1().CronJobs(ns).Get(name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	return err
}

func checkDaemonsetPods(name string, ns string) error {
	var oc = exutil.NewCLIWithoutNamespace("")
	nodes, err := oc.AdminKubeClient().CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	nodeCount := len(nodes.Items)
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		daemonset, err := oc.AdminKubeClient().AppsV1().DaemonSets(ns).Get(name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				e2e.Logf("Waiting for availability of %s daemonset\n", name)
				return false, nil
			}
			return false, err
		}
		if int(daemonset.Status.NumberReady) == nodeCount {
			return true, nil
		}
		e2e.Logf("Waiting for full availability of %s daemonset (%d/%d)\n", name, int(daemonset.Status.NumberReady), nodeCount)
		return false, nil
	})
	if err != nil {
		return err
	}
	e2e.Logf("Daemonset %s available (%d/%d)\n", name, nodeCount, nodeCount)
	return nil
}
