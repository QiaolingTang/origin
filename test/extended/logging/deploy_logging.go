package logging

import (
	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
)

var (
	//CLO resource files to create subscription for cluster-logging-operator
	CLO = OperatorObjects{
		"test/extended/testdata/clusterlogging/deployment/01-clo-project.yaml",
		"test/extended/testdata/clusterlogging/deployment/02-clo-og.yaml",
		"test/extended/testdata/clusterlogging/deployment/03-clo-csc.yaml",
		"test/extended/testdata/clusterlogging/deployment/04-clo-sub.yaml"}
	//EO resource files to create subscription for elasticsearch-operator
	EO = OperatorObjects{
		"test/extended/testdata/clusterlogging/deployment/01_eo-project.yaml",
		"test/extended/testdata/clusterlogging/deployment/02_eo-og.yaml",
		"test/extended/testdata/clusterlogging/deployment/03_eo-csc.yaml",
		"test/extended/testdata/clusterlogging/deployment/05_eo-sub.yaml"}
	rbac = "test/extended/testdata/clusterlogging/deployment/04_eo-rbac.yaml"
)

var _ = g.Describe("[Feature:Logging]Logging", func() {
	defer g.GinkgoRecover()
	var oc = exutil.NewCLIWithoutNamespace("logging")
	g.BeforeEach(func() {
		err := oc.AsAdmin().Run("get").Args("-n", "openshift-marketplace", "packagemanifests", "cluster-logging").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		err = oc.AsAdmin().Run("get").Args("-n", "openshift-marketplace", "packagemanifests", "elasticsearch-operator").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
	})
	/*
		g.AfterEach(func() {

		})
	*/

	g.Describe("OCP-21311 - Deploy Logging Via Community Operators", func() {

		g.It("should deploy logging successfully", func() {
			instance := "https://raw.githubusercontent.com/openshift-qe/v3-testfiles/master/logging/clusterlogging/example.yaml"
			ns := "openshift-logging"
			g.By("Creating subscription for cluster-logging-operator")
			err := createLoggingResources(CLO)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Creating subscription for elasticsearch-operator")
			err = createLoggingResources(EO)
			o.Expect(err).NotTo(o.HaveOccurred())
			err = oc.AsAdmin().Run("create").Args("-f", rbac).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Check if the cluster-logging-operator is ready or not")
			err = waitForOperatorToBeReady(ns, "cluster-logging-operator", retryInterval, timeout)
			o.Expect(err).NotTo(o.HaveOccurred())
			//clolabel := exutil.ParseLabelsOrDie("name=cluster-logging-operator")
			//err = utils.WaitForPodsWithLabelRunning(oc.AdminKubeClient(), "openshift-logging", clolabel)
			//o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Check if the elasticsearch-operator is ready or not")
			err = waitForOperatorToBeReady("openshift-operators-redhat", "elasticsearch-operator", retryInterval, timeout)
			o.Expect(err).NotTo(o.HaveOccurred())
			//eolabel := exutil.ParseLabelsOrDie("name=elasticsearch-operator")
			//err = utils.WaitForPodsWithLabelRunning(oc.AdminKubeClient(), "openshift-operators-redhat", eolabel)
			//o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Creating clusterlogging instance")
			err = oc.AsAdmin().Run("create").Args("-f", instance).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("waiting for the fluentd pods to be ready...")
			err = checkDaemonsetPods("fluentd", ns)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("waiting for the kibana pod to be ready...")
			err = checkLoggingPodsRunning(ns, "component=kibana")
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("waiting for the elasticsearch pod to be ready...")
			err = checkLoggingPodsRunning(ns, "cluster-name=elasticsearch")
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("check cronjob")
			err = checkCronJob("curator", ns)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("check CRDs")
			err = checkResourcesCreatedByOperators(ns, "clusterlogging", "instance")
			o.Expect(err).NotTo(o.HaveOccurred())
			err = checkResourcesCreatedByOperators(ns, "elasticsearch", "elasticsearch")
			o.Expect(err).NotTo(o.HaveOccurred())
			err = checkResourcesCreatedByOperators(ns, "servicemonitor", "fluentd")
			o.Expect(err).NotTo(o.HaveOccurred())
			err = checkResourcesCreatedByOperators(ns, "servicemonitor", "monitor-elasticsearch-cluster")
			o.Expect(err).NotTo(o.HaveOccurred())
			err = checkResourcesCreatedByOperators(ns, "prometheusrule", "elasticsearch-prometheus-rules")
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Deleting logging resources")
			err = clearResources("sub", "--all", ns)
			o.Expect(err).NotTo(o.HaveOccurred())
			err = clearResources("sub", "--all", "openshift-operators-redhat")
			o.Expect(err).NotTo(o.HaveOccurred())
			err = clearResources("clusterlogging", "instance", ns)
			o.Expect(err).NotTo(o.HaveOccurred())
			err = clearResources("csc", "cluster-logging-operator", "openshift-marketplace")
			o.Expect(err).NotTo(o.HaveOccurred())
			err = clearResources("csc", "elasticsearch", "openshift-marketplace")
			o.Expect(err).NotTo(o.HaveOccurred())
			err = deleteNamespace(ns)
			o.Expect(err).NotTo(o.HaveOccurred())
			err = deleteNamespace("openshift-operators-redhat")
			o.Expect(err).NotTo(o.HaveOccurred())
		})
	})

})
