package frontend

import (
	"context"
	"encoding/json"
	"time"

	analysisv1alpha1 "github.com/koordinator-sh/koordinator/apis/analysis/v1alpha1"
	"github.com/koordinator-sh/koordinator/pkg/client/clientset/versioned/typed/analysis/v1alpha1"
	"github.com/koordinator-sh/koordinator/pkg/prediction/manager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

var _ StatusFetcher = &statusFetcher{}

type statusFetcher struct {
	client     v1alpha1.RecommendationsGetter
	predictMgr manager.PredictionManager
	shareState ShareState
	ctx        context.Context
}

type StatusFetcher interface {
	Run()
	Started() bool
	UpdateStatus()
}

func InitStatusFetcher(client v1alpha1.RecommendationsGetter, ctx context.Context, predictMgr manager.PredictionManager, shareState ShareState) *statusFetcher {
	return &statusFetcher{
		client:     client,
		ctx:        ctx,
		predictMgr: predictMgr,
		shareState: shareState,
	}
}

func getNextUpdate(now time.Time) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
	if next.Before(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func (s *statusFetcher) Run() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(time.Until(getNextUpdate(time.Now()))):
			s.UpdateStatus()
		}
	}
}

func (s *statusFetcher) Started() bool {
	return false
}

func (s *statusFetcher) UpdateStatus() {
	var result apis.ProfileResult
	for key, enabled := range s.shareState.ProfilerList {
		if enabled {
			klog.Infof("Updater is updating crd %+v", key)
			err := s.predictMgr.GetResult(key, &result, nil)
			if err != nil {
				klog.Error("Error while get result of Profilekey ", key)
			}
			status := result.AsStatus()
			_, err = s.patchRecommendation(key.Name(), key.Namespace(), &status)
			if err != nil {
				klog.Errorf("Error while patch Recommend status: %+v", err)
				return
			}
		} else {
			continue
		}
	}
}

type patchRecord struct {
	Op    string      `json:"op,inline"`
	Path  string      `json:"path,inline"`
	Value interface{} `json:"value"`
}

func (s *statusFetcher) patchRecommendation(name string, namespace string, status *analysisv1alpha1.RecommendedPodStatus) (result *analysisv1alpha1.Recommendation, err error) {
	patches := []patchRecord{{
		Op:    "add",
		Path:  "/status",
		Value: status,
	}}
	bytes, err := json.Marshal(patches)
	if err != nil {
		klog.Errorf("Can not marshal Recommend status patches %+v, Reason: %+v", patches, err)
		return nil, err
	}
	client := s.client.Recommendations(namespace)
	return client.Patch(context.TODO(), name, types.JSONPatchType, bytes, metav1.PatchOptions{})
}
