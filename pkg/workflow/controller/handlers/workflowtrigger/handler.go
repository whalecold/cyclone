package workflowtrigger

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/caicloud/cyclone/pkg/apis/cyclone/v1alpha1"
	"github.com/caicloud/cyclone/pkg/k8s/clientset"
	"github.com/caicloud/cyclone/pkg/workflow/controller"
	"github.com/caicloud/cyclone/pkg/workflow/controller/handlers"
)

// Handler ...
type Handler struct {
	client      clientset.Interface
	cronManager *CronTriggerManager
}

var (
	// Check whether *Handler has implemented handlers.Interface interface.
	_ handlers.Interface = (*Handler)(nil)
)

const (
	// finalizerWorkflowTrigger is the cyclone related finalizer key for workflow trigger.
	finalizerWorkflowTrigger string = "workflowtrigger.cyclone.dev/finalizer"
)

// NewHandler ...
func NewHandler(client clientset.Interface) *Handler {
	return &Handler{
		client:      client,
		cronManager: NewTriggerManager(client),
	}
}

// Reconcile compares the actual state with the desired, and attempts to
// converge the two.
func (h *Handler) Reconcile(obj interface{}) (res controller.Result, err error) {
	wft, err := ToWorkflowTrigger(obj)
	if err != nil {
		log.Warn("Convert to WorkflowTrigger error: ", err)
		return res, err
	}
	if wft.Spec.Type == v1alpha1.TriggerTypeCron {
		h.cronManager.UpdateCron(wft)
	}
	return res, nil
}

// finalize ...
func (h *Handler) finalize(wft *v1alpha1.WorkflowTrigger) error {
	if wft.Spec.Type == v1alpha1.TriggerTypeCron {
		h.cronManager.DeleteCron(wft)
	}
	return nil
}

// AddFinalizer adds a finalizer to the object and update the object to the Kubernetes.
func (h *Handler) AddFinalizer(obj interface{}) error {
	originWft, ok := obj.(*v1alpha1.WorkflowTrigger)
	if !ok {
		log.WithField("obj", obj).Warning("Expect WorkflowTrigger, got unknown type resource")
		return fmt.Errorf("unknown resource type")
	}

	if sets.NewString(originWft.Finalizers...).Has(finalizerWorkflowTrigger) {
		return nil
	}

	log.WithField("name", originWft.Name).Debug("Start to add finalizer for workflowTrigger")

	wft := originWft.DeepCopy()
	wft.ObjectMeta.Finalizers = append(wft.ObjectMeta.Finalizers, finalizerWorkflowTrigger)
	_, err := h.client.CycloneV1alpha1().WorkflowTriggers(wft.Namespace).Update(context.TODO(), wft, metav1.UpdateOptions{})
	return err
}

// HandleFinalizer does the finalizer key representing things.
func (h *Handler) HandleFinalizer(obj interface{}) error {
	originWft, ok := obj.(*v1alpha1.WorkflowTrigger)
	if !ok {
		log.WithField("obj", obj).Warning("Expect WorkflowTrigger, got unknown type resource")
		return fmt.Errorf("unknown resource type")
	}

	if !sets.NewString(originWft.Finalizers...).Has(finalizerWorkflowTrigger) {
		return nil
	}

	log.WithField("name", originWft.Name).Debug("Start to process finalizer for workflowTrigger")

	// Handler finalizer
	wft := originWft.DeepCopy()
	if err := h.finalize(wft); err != nil {
		return nil
	}

	wft.ObjectMeta.Finalizers = sets.NewString(wft.ObjectMeta.Finalizers...).Delete(finalizerWorkflowTrigger).UnsortedList()
	_, err := h.client.CycloneV1alpha1().WorkflowTriggers(wft.Namespace).Update(context.TODO(), wft, metav1.UpdateOptions{})
	return err
}
