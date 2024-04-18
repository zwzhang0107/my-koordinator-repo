package frontend

import (
	"context"

	analysisv1alpha1 "github.com/koordinator-sh/koordinator/apis/analysis/v1alpha1"
	"github.com/koordinator-sh/koordinator/pkg/prediction/manager"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/cri-api/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const Name = "prediction"

// Recommendation reconciles a Recommendation object
type RecommendationReconcile struct {
	client.Client
	Scheme     *runtime.Scheme
	Recorder   record.EventRecorder
	predictMgr *manager.PredictionMgrImpl
}

// +kubebuilder:rbac:groups=analysis.koordinator.sh,resources=metricpredictions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=analysis.koordinator.sh,resources=metricpredictions/status,verbs=get;update;patch
func (r *RecommendationReconcile) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var recommendation analysisv1alpha1.Recommendation

	_ = log.FromContext(ctx, "recommendation-reconcile", req.NamespacedName)

	if err := r.Client.Get(ctx, req.NamespacedName, &recommendation); err != nil {
		if errors.IsNotFound(err) {
			r.predictMgr.Unregister(req.NamespacedName)
		} else {
			return ctrl.Result{Requeue: true}, err
		}
	}
	r.predictMgr.Register(recommendation.Spec, req.NamespacedName)
	return ctrl.Result{}, nil
}

/*
func (r *RecommendationReconcile) UpdateStatus() {
	// List All crd keys or not?
	for id, key := range r.predictMgr.RecommendationList {
		status, err := r.predictMgr.GetResult()
		if err != nil {
			klog.Error("Error while Get Result:", err)
			return
		}
		r.Client.Status().Update()
	}
}
*/

func Add(mgr ctrl.Manager, predictImpl *manager.PredictionMgrImpl) error {
	reconciler := &RecommendationReconcile{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Recorder:   mgr.GetEventRecorderFor("metric-prediction"),
		predictMgr: predictImpl,
	}
	return reconciler.SetupWithManager(mgr)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RecommendationReconcile) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&analysisv1alpha1.Recommendation{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
