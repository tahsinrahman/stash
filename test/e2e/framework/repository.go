package framework

import (
	"fmt"
	"math"
	"strconv"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gomodules.xyz/stow"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	store "kmodules.xyz/objectstore-api/api/v1"
	"kmodules.xyz/objectstore-api/osm"
	"stash.appscode.dev/stash/apis"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/pkg/util"
)

type KindMetaReplicas struct {
	Kind     string
	Meta     metav1.ObjectMeta
	Replicas int
}

func (f *Framework) EventuallyRepository(workload interface{}) GomegaAsyncAssertion {
	return Eventually(func() []*api.Repository {
		switch workload.(type) {
		case *apps.DaemonSet:
			return f.DaemonSetRepos(workload.(*apps.DaemonSet))
		case *apps.Deployment:
			return f.DeploymentRepos(workload.(*apps.Deployment))
		case *core.ReplicationController:
			return f.ReplicationControllerRepos(workload.(*core.ReplicationController))
		case *apps.ReplicaSet:
			return f.ReplicaSetRepos(workload.(*apps.ReplicaSet))
		case *apps.StatefulSet:
			return f.StatefulSetRepos(workload.(*apps.StatefulSet))
		default:
			return nil
		}
	})
}

func (f *Framework) GetRepositories(kmr KindMetaReplicas) []*api.Repository {
	repoNames := make([]string, 0)
	nodeName := f.GetNodeName(kmr.Meta)
	workload := api.LocalTypedReference{Name: kmr.Meta.Name, Kind: kmr.Kind}
	switch kmr.Kind {
	case apis.KindDeployment, apis.KindReplicationController, apis.KindReplicaSet, apis.KindDaemonSet:
		repoNames = append(repoNames, workload.GetRepositoryCRDName("", nodeName))
	case apis.KindStatefulSet:
		for i := 0; i < kmr.Replicas; i++ {
			repoNames = append(repoNames, workload.GetRepositoryCRDName(kmr.Meta.Name+"-"+strconv.Itoa(i), nodeName))
		}
	}
	repositories := make([]*api.Repository, 0)
	for _, repoName := range repoNames {
		obj, err := f.StashClient.StashV1alpha1().Repositories(kmr.Meta.Namespace).Get(repoName, metav1.GetOptions{})
		if err == nil {
			repositories = append(repositories, obj)
		}
	}
	return repositories
}

func (f *Framework) DeleteRepositories(repositories []*api.Repository) {
	for _, repo := range repositories {
		err := f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Delete(repo.Name, deleteInBackground())
		Expect(err).NotTo(HaveOccurred())
	}
}

func (f *Framework) DeleteRepository(repository *api.Repository) error {
	err := f.StashClient.StashV1alpha1().Repositories(repository.Namespace).Delete(repository.Name, deleteInBackground())
	return err
}
func (f *Framework) BrowseResticRepository(repository *api.Repository) ([]stow.Item, error) {
	cfg, err := osm.NewOSMContext(f.KubeClient, repository.Spec.Backend, repository.Namespace)
	if err != nil {
		return nil, err
	}

	loc, err := stow.Dial(cfg.Provider, cfg.Config)
	if err != nil {
		return nil, err
	}

	bucket, prefix, err := util.GetBucketAndPrefix(&repository.Spec.Backend)
	if err != nil {
		return nil, err
	}
	prefix = prefix + "/"

	container, err := loc.Container(bucket)
	if err != nil {
		return nil, err
	}

	cursor := stow.CursorStart
	items, _, err := container.Items(prefix, cursor, 50)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (f *Framework) BackupCountInRepositoriesStatus(repos []*api.Repository) int64 {
	var backupCount int64 = math.MaxInt64

	// use minimum backupCount among all repos
	for _, repo := range repos {
		if int64(repo.Status.BackupCount) < backupCount {
			backupCount = int64(repo.Status.BackupCount)
		}
	}
	return backupCount
}

func (f *Framework) CreateRepository(repo *api.Repository) error {
	_, err := f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(repo)

	return err

}

func (f *Invocation) NewLocalRepository(secretName string, pvcName string) *api.Repository {
	return &api.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(f.app + "-local"),
			Namespace: f.namespace,
		},
		Spec: api.RepositorySpec{
			Backend: store.Backend{
				Local: &store.LocalSpec{
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
					MountPath: TestSafeDataMountPath,
				},
				StorageSecretName: secretName,
			},
		},
	}
}

func (f *Invocation) NewGCSRepository(secretName string) *api.Repository {
	return &api.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("gcs-%s", f.app)),
			Namespace: f.namespace,
		},
		Spec: api.RepositorySpec{
			Backend: store.Backend{
				GCS: &store.GCSSpec{
					Bucket: "appscode-qa",
					Prefix: fmt.Sprintf("stash-e2e/%s/%s", f.namespace, f.app),
				},
				StorageSecretName: secretName,
			},
			WipeOut: true,
		},
	}
}

func (f *Invocation) SetupLocalRepository() (*api.Repository, error) {
	// Create Storage Secret
	By("Creating Storage Secret")
	cred := f.SecretForLocalBackend()
	err := f.CreateSecret(cred)
	Expect(err).NotTo(HaveOccurred())
	f.AppendToCleanupList(&cred)

	// We are going to use an PVC as backend
	By("Creating Backend PVC")
	backendPVC := f.GetPersistentVolumeClaim()
	err = f.CreatePersistentVolumeClaim(backendPVC)
	Expect(err).NotTo(HaveOccurred())
	f.AppendToCleanupList(backendPVC)

	// Generate Repository Definition
	repo := f.NewLocalRepository(cred.Name, backendPVC.Name)

	// Create Repository
	By("Creating Repository")
	return f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(repo)
}

func (f *Invocation) SetupGCSRepository() (*api.Repository, error) {
	// Create Storage Secret
	By("Creating Storage Secret")
	cred := f.SecretForGCSBackend()

	if missing, _ := BeZero().Match(cred); missing {
		Skip("Missing GCS credential")
	}
	err := f.CreateSecret(cred)
	Expect(err).NotTo(HaveOccurred())
	f.AppendToCleanupList(&cred)

	// Generate Repository Definition
	repo := f.NewGCSRepository(cred.Name)

	// Create Repository
	By("Creating Repository")
	return f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(repo)
}
