package api

import "github.com/loadmesh/loadmesh/model"

type ResourceReconciler interface {
	Reconcile(resource *model.Resource) error
}
