package controllers

import (
	"reflect"
	"time"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/caicloud/cyclone/pkg/k8s/clientset"
	"github.com/caicloud/cyclone/pkg/k8s/informers"
	"github.com/caicloud/cyclone/pkg/workflow/controller"
	"github.com/caicloud/cyclone/pkg/workflow/controller/handlers/executioncluster"
)

// NewExecutionClusterController ...
func NewExecutionClusterController(client clientset.Interface) *Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	factory := informers.NewSharedInformerFactoryWithOptions(
		client,
		controller.Config.ResyncPeriodSeconds*time.Second,
	)

	informer := factory.Cyclone().V1alpha1().ExecutionClusters().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				return
			}
			queue.Add(Event{
				Key:       key,
				EventType: CREATE,
				Object:    obj,
			})
		},
		UpdateFunc: func(old, new interface{}) {
			if reflect.DeepEqual(old, new) {
				return
			}
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err != nil {
				return
			}
			queue.Add(Event{
				Key:       key,
				EventType: UPDATE,
				Object:    new,
				OldObject: old,
			})
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				return
			}
			queue.Add(Event{
				Key:       key,
				EventType: DELETE,
				Object:    obj,
			})
		},
	})

	return &Controller{
		name:      "Execution Cluster Controller",
		clientSet: client,
		informer:  informer,
		queue:     queue,
		eventHandler: &executioncluster.Handler{
			Client: client,
		},
	}
}
