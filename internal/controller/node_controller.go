package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/eplightning/node-min-lifetime/internal/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type NodeReconciler struct {
	client.Client
	cfg *config.Config
}

const (
	ScaleDownDisabledAnnotation = "cluster-autoscaler.kubernetes.io/scale-down-disabled"
	DoNotDisruptAnnotation      = "karpenter.sh/do-not-disrupt"

	CustomScaleDownDisabledAnnotation = "node-min-lifetime.eplight.org/scale-down-disabled"
	CustomScaleDownDisabledLabel      = "node-min-lifetime.eplight.org/scale-down-disabled"
	MinLifetimeAnnotation             = "node-min-lifetime.eplight.org/min-lifetime"
	MinLifetimeLabel                  = "node-min-lifetime.eplight.org/min-lifetime"
)

func NewNodeReconciler(c client.Client, cfg *config.Config) *NodeReconciler {
	return &NodeReconciler{
		Client: c,
		cfg:    cfg,
	}
}

func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Watches(
			&corev1.Node{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
				return r.cfg.NodeSelector.Selector.Matches(labels.Set(object.GetLabels()))
			})),
		).
		Complete(r)
}

func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	var node corev1.Node
	if err := r.Get(ctx, req.NamespacedName, &node); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if node.Annotations == nil {
		node.Annotations = make(map[string]string)
	}

	minLifetime := r.cfg.DefaultMinLifetime
	if node.Annotations[MinLifetimeAnnotation] != "" {
		minLifetime, err = time.ParseDuration(node.Annotations[MinLifetimeAnnotation])
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to parse min lifetime annotation: %w", err)
		}
	} else if node.Labels[MinLifetimeLabel] != "" {
		minLifetime, err = time.ParseDuration(node.Labels[MinLifetimeAnnotation])
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to parse min lifetime label: %w", err)
		}
	}

	lifetime := time.Since(node.CreationTimestamp.Time)
	var requeueAfter time.Duration
	var scaleDownDisabled bool

	if node.Annotations[CustomScaleDownDisabledAnnotation] == "true" || node.Labels[CustomScaleDownDisabledLabel] == "true" {
		scaleDownDisabled = true
	} else if node.Annotations[CustomScaleDownDisabledAnnotation] == "false" || node.Labels[CustomScaleDownDisabledLabel] == "false" {
		scaleDownDisabled = false
	} else if lifetime < minLifetime {
		scaleDownDisabled = true
		requeueAfter = minLifetime - lifetime
	}

	if scaleDownDisabled &&
		(node.Annotations[ScaleDownDisabledAnnotation] != "true" || node.Annotations[DoNotDisruptAnnotation] != "true") {
		node.Annotations[ScaleDownDisabledAnnotation] = "true"
		node.Annotations[DoNotDisruptAnnotation] = "true"
		if err := r.Update(ctx, &node); err != nil {
			return ctrl.Result{}, err
		}
	} else if !scaleDownDisabled &&
		(node.Annotations[ScaleDownDisabledAnnotation] != "" || node.Annotations[DoNotDisruptAnnotation] != "") {
		delete(node.Annotations, ScaleDownDisabledAnnotation)
		delete(node.Annotations, DoNotDisruptAnnotation)
		if err := r.Update(ctx, &node); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{
		RequeueAfter: requeueAfter,
	}, nil
}
