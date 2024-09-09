package apis

import (
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/types"
)

var _ ProfileKey = recommendKey{}

type recommendKey struct {
	namespace string
	name      string
}

func MakeProfileKey(key types.NamespacedName) ProfileKey {
	return recommendKey{
		namespace: key.Namespace,
		name:      key.Name,
	}
}

func (r recommendKey) Key() string {
	return r.namespace + r.name
}

func (r recommendKey) Namespace() string {
	return r.namespace
}

func (r recommendKey) Name() string {
	return r.name
}

func (r recommendKey) NamePattern() string {
	return fmt.Sprintf("^%s.*$", regexp.QuoteMeta(r.name))
}
