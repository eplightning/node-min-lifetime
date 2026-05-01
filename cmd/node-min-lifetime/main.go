package main

import (
	"log/slog"
	"os"

	"github.com/eplightning/node-min-lifetime/internal/config"
	"github.com/eplightning/node-min-lifetime/internal/controller"
	"github.com/eplightning/node-min-lifetime/internal/supervisor"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	ctrl.SetLogger(logr.FromSlogHandler(logger.Handler()))

	cfg := config.MustLoadConfig()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                        scheme,
		Metrics:                       metricsserver.Options{BindAddress: cfg.MetricsBindAddress},
		HealthProbeBindAddress:        cfg.HealthProbeBindAddress,
		LeaderElection:                cfg.LeaderElect,
		LeaderElectionID:              cfg.LeaderElectID,
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		slog.Error("Unable to start manager", "error", err)
		os.Exit(1)
	}

	nodeReconciler := controller.NewNodeReconciler(mgr.GetClient(), cfg)
	if err = nodeReconciler.SetupWithManager(mgr); err != nil {
		slog.Error("Unable to setup node reconciler", "error", err)
		os.Exit(1)
	}

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		slog.Error("Unable to setup health check", "error", err)
		os.Exit(1)
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		slog.Error("Unable to setup ready check", "error", err)
		os.Exit(1)
	}

	sv := supervisor.New(mgr)

	if err := sv.Start(supervisor.SignalContext()); err != nil {
		slog.Error("Shutting down due to error", "error", err)
		os.Exit(1)
	}
}
