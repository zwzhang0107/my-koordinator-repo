package main

import (
	"flag"
	"os"
	"sync"
	"time"

	analysisv1alpha1 "github.com/koordinator-sh/koordinator/apis/analysis/v1alpha1"
	analysis_clientset "github.com/koordinator-sh/koordinator/pkg/client/clientset/versioned"
	"github.com/koordinator-sh/koordinator/pkg/prediction/frontend"
	"github.com/koordinator-sh/koordinator/pkg/prediction/manager"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/apis/batch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("predict-setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = batch.AddToScheme(scheme)
	_ = analysisv1alpha1.AddToScheme(scheme)
}

var predictImpl *manager.PredictionMgrImpl

func main() {
	flag.Parse()

	cfg, err := restclient.InClusterConfig()
	if err != nil {
		setupLog.Error(err, "problem get rest client")
	}

	go wait.Forever(klog.Flush, 5*time.Second)
	defer klog.Flush()

	stopCtx := signals.SetupSignalHandler()
	wg := &sync.WaitGroup{}

	//init manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "unable to start predict manager")
		os.Exit(1)
	}

	//start mgr
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running predict manager")
		os.Exit(1)
	}

	predictImpl = manager.InitPredictMgr()
	//register crd reconcile to manager
	frontend.Add(mgr, predictImpl)

	client := analysis_clientset.NewForConfigOrDie(cfg).AnalysisV1alpha1()
	fetcher := frontend.InitStatusFetcher(client, stopCtx, predictImpl)
	setupLog.Info("starting status fetcher")
	//status fetcher start
	wg.Add(1)
	go func() {
		fetcher.Run()
		wg.Done()
	}()

	//predictionImpl Start
	setupLog.Info("starting predict manager")
	wg.Add(1)
	go func() {
		predictImpl.Run()
		wg.Done()
	}()

	<-stopCtx.Done()
	wg.Wait()
	// +kubebuilder:scaffold:builder
}
