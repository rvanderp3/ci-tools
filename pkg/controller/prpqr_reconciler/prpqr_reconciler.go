package prpqr_reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	prowv1 "k8s.io/test-infra/prow/apis/prowjobs/v1"
	prowconfig "k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/pjutil"
	utilpointer "k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1 "github.com/openshift/ci-tools/pkg/api/pullrequestpayloadqualification/v1"
	"github.com/openshift/ci-tools/pkg/controller/prpqr_reconciler/pjstatussyncer"
	controllerutil "github.com/openshift/ci-tools/pkg/controller/util"
	"github.com/openshift/ci-tools/pkg/jobconfig"
)

const (
	controllerName = "prpqr_reconciler"
)

func AddToManager(mgr manager.Manager, ns string) error {
	if err := pjstatussyncer.AddToManager(mgr, ns); err != nil {
		return fmt.Errorf("failed to construct pjstatussyncer: %w", err)
	}

	c, err := controller.New(controllerName, mgr, controller.Options{
		MaxConcurrentReconciles: 1,
		Reconciler: &reconciler{
			logger: logrus.WithField("controller", controllerName),
			client: mgr.GetClient(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	// Watch only on Create events
	predicateFuncs := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Object.GetNamespace() == ns
		},
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		UpdateFunc:  func(event.UpdateEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
	if err := c.Watch(source.NewKindWithCache(&v1.PullRequestPayloadQualificationRun{}, mgr.GetCache()), prpqrHandler(), predicateFuncs); err != nil {
		return fmt.Errorf("failed to create watch for PullRequestPayloadQualificationRun: %w", err)
	}

	return nil
}

func prpqrHandler() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(o ctrlruntimeclient.Object) []reconcile.Request {
		prpqr, ok := o.(*v1.PullRequestPayloadQualificationRun)
		if !ok {
			logrus.WithField("type", fmt.Sprintf("%T", o)).Error("Got object that was not a PullRequestPayloadQualificationRun")
			return nil
		}

		return []reconcile.Request{
			{NamespacedName: types.NamespacedName{Namespace: prpqr.Namespace, Name: prpqr.Name}},
		}
	})
}

type reconciler struct {
	logger *logrus.Entry
	client ctrlruntimeclient.Client
}

func (r *reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := r.logger.WithField("request", req.String())
	err := r.reconcile(ctx, req, logger)
	if err != nil {
		logger.WithError(err).Error("Reconciliation failed")
	} else {
		logger.Info("Finished reconciliation")
	}
	return reconcile.Result{}, controllerutil.SwallowIfTerminal(err)
}

func (r *reconciler) reconcile(ctx context.Context, req reconcile.Request, logger *logrus.Entry) error {
	logger = logger.WithField("namespace", req.Namespace).WithField("prpqr_name", req.Name)
	logger.Info("Starting reconciliation")

	prpqr := &v1.PullRequestPayloadQualificationRun{}
	if err := r.client.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: req.Namespace, Name: req.Name}, prpqr); err != nil {
		return fmt.Errorf("failed to get the PullRequestPayloadQualificationRun: %s in namespace %s: %w", req.Name, req.Namespace, err)
	}

	prowjobs := generateProwjobs(prpqr.Spec.PullRequest.Org, prpqr.Spec.PullRequest.Repo, prpqr.Spec.PullRequest.BaseRef, req.Name, req.Namespace, prpqr.Spec.Jobs.Jobs)
	for releaseJobName, pj := range prowjobs {
		logger = logger.WithFields(logrus.Fields{"name": pj.Name, "namespace": req.Namespace})

		err := r.client.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: pj.Namespace, Name: pj.Name}, &prowv1.ProwJob{})
		if err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to get the Prowjob: %s in namespace %s: %w", req.Name, req.Namespace, err)
		}

		if !kerrors.IsNotFound(err) {
			continue
		}

		logger.Info("Creating prowjob...")
		if err := r.client.Create(ctx, &pj); err != nil {
			return fmt.Errorf("failed to create prowjob: %w", err)
		}

		// There is some delay until it gets back to our cache, so block until we can retrieve
		// it successfully.
		key := ctrlruntimeclient.ObjectKey{Namespace: pj.Namespace, Name: pj.Name}
		if err := wait.Poll(100*time.Millisecond, 5*time.Second, func() (bool, error) {
			if err := r.client.Get(ctx, key, &prowv1.ProwJob{}); err != nil {
				if kerrors.IsNotFound(err) {
					return false, nil
				}
				return false, fmt.Errorf("getting prowJob failed: %w", err)
			}
			return true, nil
		}); err != nil {
			return fmt.Errorf("failed to wait for created ProwJob to appear in cache: %w", err)
		}

		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			prpqr := &v1.PullRequestPayloadQualificationRun{}
			if err := r.client.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: req.Namespace, Name: req.Name}, prpqr); err != nil {
				return fmt.Errorf("failed to get the PullRequestPayloadQualificationRun: %w", err)
			}

			prpqr.Status.Jobs = append(prpqr.Status.Jobs, v1.PullRequestPayloadJobStatus{
				ReleaseJobName: releaseJobName,
				ProwJob:        pj.Name,
				Status:         pj.Status,
			})

			logger.Info("Updating PullRequestPayloadQualificationRun...")
			if err := r.client.Update(ctx, prpqr); err != nil {
				return fmt.Errorf("failed to update PullRequestPayloadQualificationRun %s: %w", prpqr.Name, err)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// TODO: Currently we create a single dummy prowjob just for testing. The actual implementation
// will be introduced in https://issues.redhat.com/browse/DPTP-2577
func generateProwjobs(org, repo, branch, prpqrName, prpqrNamespace string, releaseJobSpec []v1.ReleaseJobSpec) map[string]prowv1.ProwJob {
	ret := make(map[string]prowv1.ProwJob)
	for _, spec := range releaseJobSpec {
		labels := map[string]string{
			v1.PullRequestPayloadQualificationRunLabel: prpqrName,
		}

		base := prowconfig.JobBase{
			Agent: string(prowv1.KubernetesAgent),
			Spec: &corev1.PodSpec{
				Containers: []corev1.Container{{Image: "centos:8", Command: []string{"sleep"}, Args: []string{"100"}}},
			},

			UtilityConfig: prowconfig.UtilityConfig{
				Decorate: utilpointer.BoolPtr(true),
			},
		}

		extraRefs := prowv1.Refs{
			Org:     org,
			Repo:    repo,
			BaseRef: branch,
		}
		base.ExtraRefs = []prowv1.Refs{extraRefs}

		periodicJob := prowconfig.Periodic{
			JobBase: base,
			Cron:    "@yearly",
		}

		pj := pjutil.NewProwJob(pjutil.PeriodicSpec(periodicJob), labels, nil)
		pj.Namespace = prpqrNamespace

		ret[spec.CIOperatorConfig.JobName(jobconfig.PeriodicPrefix, spec.Test)] = pj
	}
	return ret
}
