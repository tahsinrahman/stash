package backup

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/appscode/go/log"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
	"stash.appscode.dev/stash/apis"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	stashinformers "stash.appscode.dev/stash/client/informers/externalversions"
	"stash.appscode.dev/stash/client/listers/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/status"
	"stash.appscode.dev/stash/pkg/util"
)

type BackupSessionController struct {
	Config               *rest.Config
	K8sClient            kubernetes.Interface
	StashClient          cs.Interface
	MasterURL            string
	KubeconfigPath       string
	StashInformerFactory stashinformers.SharedInformerFactory
	MaxNumRequeues       int
	NumThreads           int
	ResyncPeriod         time.Duration
	//backupConfiguration
	BackupConfigurationName string
	Namespace               string
	//Backup Session
	bsQueue    *queue.Worker
	bsInformer cache.SharedIndexInformer
	bsLister   v1beta1.BackupSessionLister

	SetupOpt restic.SetupOptions
	Host     string
	Metrics  restic.MetricsOptions
	Recorder record.EventRecorder
}

func (c *BackupSessionController) RunBackup() error {
	stopCh := make(chan struct{})
	defer close(stopCh)

	//get BackupConfiguration
	backupConfiguration, err := c.StashClient.StashV1beta1().BackupConfigurations(c.Namespace).Get(c.BackupConfigurationName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if backupConfiguration.Spec.Target == nil {
		return fmt.Errorf("backupConfiguration target is nil")
	}

	// for Deployment, ReplicaSet and ReplicationController run BackupSession watcher only in leader pod.
	// for others workload i.e. DaemonSet and StatefulSet run BackupSession watcher in all pods.
	switch backupConfiguration.Spec.Target.Ref.Kind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController, apis.KindDeploymentConfig:
		if err := c.electLeaderPod(backupConfiguration, stopCh); err != nil {
			return err
		}
	default:
		if err := c.runBackupSessionController(backupConfiguration, stopCh); err != nil {
			return err
		}
	}
	glog.Info("Stopping Stash backup")
	return nil
}

func (c *BackupSessionController) runBackupSessionController(backupConfiguration *api_v1beta1.BackupConfiguration, stopCh <-chan struct{}) error {
	// start BackupSession watcher
	err := c.initBackupSessionWatcher(backupConfiguration)
	if err != nil {
		return err
	}

	// start BackupSession informer
	c.StashInformerFactory.Start(stopCh)
	for _, v := range c.StashInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return nil
		}
	}
	c.bsQueue.Run(stopCh)

	// BackupSession controller has started successfully. write event to the respective BackupConfiguration.
	err = c.handleBackupSetupSuccess(backupConfiguration)
	if err != nil {
		return err
	}

	// wait until stop signal is sent.
	<-stopCh
	return nil
}

func (c *BackupSessionController) initBackupSessionWatcher(backupConfiguration *api_v1beta1.BackupConfiguration) error {
	// only watch BackupSessions of this BackupConfiguration.
	// respective CronJob creates BackupSession with BackupConfiguration's name as label.
	// so we will watch only those BackupSessions that has this BackupConfiguration name in labels.
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			util.LabelBackupConfiguration: backupConfiguration.Name,
		},
	})
	if err != nil {
		return err
	}

	c.bsInformer = c.StashInformerFactory.Stash().V1beta1().BackupSessions().Informer()
	c.bsQueue = queue.New(api_v1beta1.ResourceKindBackupSession, c.MaxNumRequeues, c.NumThreads, c.processBackupSession)
	c.bsInformer.AddEventHandler(queue.NewFilteredHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if backupsession, ok := obj.(*api_v1beta1.BackupSession); ok {
				queue.Enqueue(c.bsQueue.GetQueue(), backupsession)
			}
		},
	}, selector))
	c.bsLister = c.StashInformerFactory.Stash().V1beta1().BackupSessions().Lister()
	return nil
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *BackupSessionController) processBackupSession(key string) error {
	obj, exists, err := c.bsInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("Backup Session %s does not exist anymore\n", key)

	} else {
		backupSession := obj.(*api_v1beta1.BackupSession)
		glog.Infof("Sync/Add/Update for Backup Session %s", backupSession.GetName())

		backupOutput, err := c.startBackupProcess(backupSession)
		if err != nil {
			return c.handleBackupFailure(backupSession.Name, err)
		}

		if backupOutput != nil {
			return c.handleBackupSuccess(backupSession.Name, backupOutput)
		}
	}
	return nil
}

func (c *BackupSessionController) startBackupProcess(backupSession *api_v1beta1.BackupSession) (*restic.BackupOutput, error) {
	// get respective BackupConfiguration for BackupSession
	backupConfiguration, err := c.StashClient.StashV1beta1().BackupConfigurations(backupSession.Namespace).Get(
		backupSession.Spec.BackupConfiguration.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("can't get BackupConfiguration for BackupSession %s/%s. Reason: %s", backupSession.Namespace, backupSession.Name, err)
	}

	// skip if BackupConfiguration paused
	if backupConfiguration.Spec.Paused {
		log.Infof("Skipping processing BackupSession %s/%s. Reason: Backup Configuration is paused.", backupSession.Namespace, backupSession.Name)
		return nil, nil
	}

	// if BackupSession already has been processed for this host then skip further processing
	if c.isBackupTakenForThisHost(backupSession, c.Host) {
		log.Infof("Skip processing BackupSession %s/%s. Reason: BackupSession has been processed already for host %q\n", backupSession.Namespace, backupSession.Name, c.Host)
		return nil, nil
	}

	// For Deployment, ReplicaSet and ReplicationController only leader pod is running this controller so no problem with restic repo lock.
	// For StatefulSet and DaemonSet all pods are running this controller and all will try to backup simultaneously. But, restic repository can be
	// locked by only one pod. So, we need a leader election to determine who will take backup first. Once backup is complete, the leader pod will
	// step down from leadership so that another replica can acquire leadership and start taking backup.
	switch backupConfiguration.Spec.Target.Ref.Kind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController, apis.KindDeploymentConfig:
		return c.backup(backupSession, backupConfiguration)
	default:
		return nil, c.electBackupLeader(backupSession, backupConfiguration)
	}
}

func (c *BackupSessionController) backup(backupSession *api_v1beta1.BackupSession, backupConfiguration *api_v1beta1.BackupConfiguration) (*restic.BackupOutput, error) {

	// get repository
	repository, err := c.StashClient.StashV1alpha1().Repositories(backupConfiguration.Namespace).Get(backupConfiguration.Spec.Repository.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// configure SourceHost, SecretDirectory, EnableCache and ScratchDirectory
	extraOpt := util.ExtraOptions{
		Host:        c.Host,
		SecretDir:   c.SetupOpt.SecretDir,
		EnableCache: c.SetupOpt.EnableCache,
		ScratchDir:  c.SetupOpt.ScratchDir,
	}

	// configure setupOption
	c.SetupOpt, err = util.SetupOptionsForRepository(*repository, extraOpt)
	if err != nil {
		return nil, fmt.Errorf("setup option for repository fail")
	}

	// apply nice, ionice settings from env
	c.SetupOpt.Nice, err = util.NiceSettingsFromEnv()
	if err != nil {
		return nil, err
	}
	c.SetupOpt.IONice, err = util.IONiceSettingsFromEnv()
	if err != nil {
		return nil, err
	}

	// init restic wrapper
	resticWrapper, err := restic.NewResticWrapper(c.SetupOpt)
	if err != nil {
		return nil, err
	}

	// BackupOptions configuration
	backupOpt := util.BackupOptionsForBackupConfig(*backupConfiguration, extraOpt)
	// Run Backup
	return resticWrapper.RunBackup(backupOpt)
}

func (c *BackupSessionController) electLeaderPod(backupConfiguration *api_v1beta1.BackupConfiguration, stopCh <-chan struct{}) error {
	log.Infoln("Attempting to elect leader pod")

	rlc := resourcelock.ResourceLockConfig{
		Identity:      os.Getenv(util.KeyPodName),
		EventRecorder: eventer.NewEventRecorder(c.K8sClient, BackupEventComponent),
	}
	resLock, err := resourcelock.New(
		resourcelock.ConfigMapsResourceLock,
		backupConfiguration.Namespace,
		util.GetBackupConfigmapLockName(backupConfiguration.Spec.Target.Ref),
		c.K8sClient.CoreV1(),
		c.K8sClient.CoordinationV1(),
		rlc,
	)
	if err != nil {
		return fmt.Errorf("failed to create resource lock. Reason: %s", err)
	}

	// use a Go context so we can tell the leader election code when we
	// want to step down
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start the leader election code loop
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:          resLock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				log.Infoln("Got leadership, starting BackupSession controller")
				// this pod is now leader. run BackupSession controller.
				setupErr := c.runBackupSessionController(backupConfiguration, stopCh)
				if setupErr != nil {
					// write event to the respective BackpConfiguration
					err := c.HandleBackupSetupFailure(err)
					// step down from leadership so that other replicas can try to start BackupSession controller
					cancel()
					// fail the container so that it restart and retry.
					// we should not fail container as it may interrupt user's service.
					// however, we are doing it here because it is happening for the first time when stash has injected
					// a sidecar. user's pod will restart automatically for sidecar injection. so, we can restart the pod
					// to ensure backup has been configured properly at this time as the user will be aware of service interruption.
					if err != nil {
						log.Fatal(err)
					}
				}
			},
			OnStoppedLeading: func() {
				log.Infoln("Lost leadership")
			},
		},
	})
	return nil
}

func (c *BackupSessionController) electBackupLeader(backupSession *api_v1beta1.BackupSession, backupConfiguration *api_v1beta1.BackupConfiguration) error {
	log.Infoln("Attempting to acquire leadership for backup")

	rlc := resourcelock.ResourceLockConfig{
		Identity:      os.Getenv(util.KeyPodName),
		EventRecorder: eventer.NewEventRecorder(c.K8sClient, BackupEventComponent),
	}

	resLock, err := resourcelock.New(
		resourcelock.ConfigMapsResourceLock,
		backupConfiguration.Namespace,
		util.GetBackupConfigmapLockName(backupConfiguration.Spec.Target.Ref),
		c.K8sClient.CoreV1(),
		c.K8sClient.CoordinationV1(),
		rlc,
	)
	if err != nil {
		return fmt.Errorf("error during leader election: %s", err)
	}

	// use a Go context so we can tell the leader election code when we
	// want to step down
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start the leader election code loop
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:          resLock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				log.Infoln("Got leadership, preparing for backup")
				// run backup process
				backupOutput, backupErr := c.backup(backupSession, backupConfiguration)
				if backupErr != nil {
					err := c.handleBackupFailure(backupSession.Name, backupErr)
					if err != nil {
						backupErr = errors.NewAggregate([]error{backupErr, err})
					}
					// step down from leadership so that other replicas can start backup
					cancel()
					// log failure. don't fail the container as it may interrupt user's service
					log.Warningf("failed to complete backup. Reason: %v", backupErr)
				}
				if backupOutput != nil {
					err := c.handleBackupSuccess(backupSession.Name, backupOutput)
					if err != nil {
						// log failure. don't fail the container as it may interrupt user's service
						log.Warningf("failed to complete backup. Reason: %v", backupErr)
					}
				}
				// backup process is complete. now, step down from leadership so that other replicas can start
				cancel()
			},
			OnStoppedLeading: func() {
				log.Infoln("Lost leadership")
			},
		},
	})
	return nil
}

func (c *BackupSessionController) HandleBackupSetupFailure(setupErr error) error {
	// write log
	log.Warningf("failed to start BackupSessionController. Reason: %v", setupErr)

	// write event to BackupConfiguration
	backupConfiguration, err := c.StashClient.StashV1beta1().BackupConfigurations(c.Namespace).Get(c.BackupConfigurationName, metav1.GetOptions{})
	if err != nil {
		return errors.NewAggregate([]error{setupErr, err})
	}
	_, err = eventer.CreateEvent(
		c.K8sClient,
		eventer.EventSourceBackupSidecar,
		backupConfiguration,
		core.EventTypeWarning,
		eventer.EventReasonFailedToStartBackupSessionController,
		fmt.Sprintf("failed to start BackupSession controller in pod %s/%s Reason: %v", meta.Namespace(), os.Getenv(util.KeyPodName), setupErr),
	)
	return errors.NewAggregate([]error{setupErr, err})
}

func (c *BackupSessionController) handleBackupSetupSuccess(backupConfiguration *api_v1beta1.BackupConfiguration) error {
	// write log
	log.Infof("BackupSession controller started successfully.")

	// write event to BackupConfiguration
	_, err := eventer.CreateEvent(
		c.K8sClient,
		eventer.EventSourceBackupSidecar,
		backupConfiguration,
		core.EventTypeNormal,
		eventer.EventReasonBackupSessionControllerStarted,
		fmt.Sprintf("BackupSession controller started successfully in pod %s/%s.", meta.Namespace(), os.Getenv(util.KeyPodName)),
	)
	return err
}

func (c *BackupSessionController) handleBackupSuccess(backupSessionName string, backupOutput *restic.BackupOutput) error {
	// write log
	log.Infof("Backup completed successfully for BackupSession %s", backupSessionName)

	// add/update entry into BackupSession status for this host
	backupConfig, err := c.StashClient.StashV1beta1().BackupConfigurations(c.Namespace).Get(c.BackupConfigurationName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	statusOpt := status.UpdateStatusOptions{
		Config:        c.Config,
		KubeClient:    c.K8sClient,
		StashClient:   c.StashClient,
		Namespace:     c.Namespace,
		Repository:    backupConfig.Spec.Repository.Name,
		BackupSession: backupSessionName,
		Metrics:       c.Metrics,
	}
	return statusOpt.UpdatePostBackupStatus(backupOutput)
}

func (c *BackupSessionController) handleBackupFailure(backupSessionName string, backupErr error) error {
	// write log
	log.Warningf("Failed to take backup for BackupSession %s. Reason: %v", backupSessionName, backupErr)

	// add/update entry into BackupSession status for this host
	backupConfig, err := c.StashClient.StashV1beta1().BackupConfigurations(c.Namespace).Get(c.BackupConfigurationName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	backupOutput := &restic.BackupOutput{
		HostBackupStats: []api_v1beta1.HostBackupStats{
			{
				Hostname: c.Host,
				Phase:    api_v1beta1.HostBackupFailed,
				Error:    fmt.Sprintf("failed to complete backup for host %s. Reaosn: %v", c.Host, backupErr),
			},
		},
	}

	statusOpt := status.UpdateStatusOptions{
		Config:        c.Config,
		KubeClient:    c.K8sClient,
		StashClient:   c.StashClient,
		Namespace:     c.Namespace,
		Repository:    backupConfig.Spec.Repository.Name,
		BackupSession: backupSessionName,
		Metrics:       c.Metrics,
	}
	return statusOpt.UpdatePostBackupStatus(backupOutput)
}

func (c *BackupSessionController) isBackupTakenForThisHost(backupSession *api_v1beta1.BackupSession, host string) bool {

	// if overall backupSession phase is "Succeeded" or "Failed" or "Skipped" then it has been processed already
	if backupSession.Status.Phase == api_v1beta1.BackupSessionSucceeded ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionFailed ||
		backupSession.Status.Phase == api_v1beta1.BackupSessionSkipped {
		return true
	}

	// if backupSession has entry for this host in status field, then it has been already processed for this host
	for _, hostStats := range backupSession.Status.Stats {
		if hostStats.Hostname == host {
			return true
		}
	}
	return false
}
