package workloads

//
//import (
//	"github.com/appscode/go/types"
//	. "github.com/onsi/ginkgo"
//	. "github.com/onsi/gomega"
//	apps "k8s.io/api/apps/v1"
//	core "k8s.io/api/core/v1"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"stash.appscode.dev/stash/apis"
//	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
//	"stash.appscode.dev/stash/apis/stash/v1beta1"
//	"stash.appscode.dev/stash/pkg/util"
//	"stash.appscode.dev/stash/test/e2e/framework"
//	"stash.appscode.dev/stash/test/e2e/matcher"
//)
//
//var _ = XDescribe("ReplicaSet", func() {
//	var (
//		f              *framework.Invocation
//		cred           core.Secret
//		repo           *api.Repository
//		backupCfg      v1beta1.BackupConfiguration
//		restoreSession v1beta1.RestoreSession
//		pvc            *core.PersistentVolumeClaim
//		targetref      v1beta1.TargetRef
//		rules          []v1beta1.Rule
//		rs             apps.ReplicaSet
//		recoveredRS    apps.ReplicaSet
//		sampleData     []string
//		restoredData   []string
//	)
//
//	BeforeEach(func() {
//		f = framework.NewInvocation()
//	})
//
//	JustBeforeEach(func() {
//		cred = f.SecretForLocalBackend()
//		if missing, _ := BeZero().Match(cred); missing {
//			Skip("Missing repository credential")
//		}
//		pvc = f.GetPersistentVolumeClaim()
//		err := f.CreatePersistentVolumeClaim(pvc)
//		Expect(err).NotTo(HaveOccurred())
//		repo = f.GetLocalRepository(cred.Name, pvc.Name)
//
//		backupCfg = f.BackupConfiguration(repo.Name, targetref)
//		rules = []v1beta1.Rule{
//			{
//				Paths: []string{
//					framework.TestSourceDataMountPath,
//				},
//			},
//		}
//		restoreSession = f.RestoreSession(repo.Name, targetref, rules)
//	})
//
//	AfterEach(func() {
//		err := f.DeleteSecret(cred.ObjectMeta)
//		Expect(err).NotTo(HaveOccurred())
//		err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
//		Expect(err).NotTo(HaveOccurred())
//	})
//
//	var (
//		testRSBackup = func() {
//			By("Creating ReplicaSet " + rs.Name)
//			_, err := f.CreateReplicaSet(rs)
//			Expect(err).NotTo(HaveOccurred())
//			err = util.WaitUntilReplicaSetReady(f.KubeClient, rs.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Creating sample data inside workload")
//			err = f.CreateSampleDataInsideWorkload(rs.ObjectMeta, apis.KindReplicaSet)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Reading sample data from /source/data mountPath inside workload")
//			sampleData, err = f.ReadSampleDataFromFromWorkload(rs.ObjectMeta, apis.KindReplicaSet)
//			Expect(err).NotTo(HaveOccurred())
//			Expect(sampleData).NotTo(BeEmpty())
//
//			By("Creating storage Secret " + cred.Name)
//			err = f.CreateSecret(cred)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Creating new repository")
//			err = f.CreateRepository(repo)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Creating BackupConfiguration" + backupCfg.Name)
//			err = f.CreateBackupConfiguration(backupCfg)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for sidecar")
//			f.EventuallyReplicaSet(rs.ObjectMeta).Should(matcher.HaveSidecar(util.StashContainer))
//
//			By("Waiting for BackupSession")
//			f.EventuallyBackupSessionCreated(backupCfg.ObjectMeta).Should(BeTrue())
//			bs, err := f.GetBackupSession(backupCfg.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Check for succeeded BackupSession")
//			f.EventuallyBackupSessionPhase(bs.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))
//
//			By("Check for repository status updated")
//			f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
//
//			By("Delete BackupConfiguration")
//			err = f.DeleteBackupConfiguration(backupCfg)
//			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting to remove sidecar")
//			f.EventuallyReplicaSet(rs.ObjectMeta).ShouldNot(matcher.HaveSidecar(util.StashContainer))
//			err = util.WaitUntilReplicaSetReady(f.KubeClient, rs.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Remove sample data from workload")
//			err = f.CleanupSampleDataFromWorkload(rs.ObjectMeta, apis.KindReplicaSet)
//			Expect(err).NotTo(HaveOccurred())
//
//		}
//	)
//
//	Context("General Backup and Restore New ReplicaSet", func() {
//		BeforeEach(func() {
//			pvc = f.GetPersistentVolumeClaim()
//			err := f.CreatePersistentVolumeClaim(pvc)
//			Expect(err).NotTo(HaveOccurred())
//			rs = f.ReplicaSet(pvc.Name)
//
//			targetref = v1beta1.TargetRef{
//				APIVersion: "apps/v1",
//				Kind:       apis.KindReplicaSet,
//				Name:       rs.Name,
//			}
//		})
//
//		AfterEach(func() {
//			err := f.DeleteReplicaSet(rs.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilReplicaSetDeleted(f.KubeClient, rs.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			err = f.DeleteRepository(repo)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilRepositoryDeleted(f.StashClient, repo)
//			Expect(err).NotTo(HaveOccurred())
//
//			err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilRestoreSessionDeleted(f.StashClient, restoreSession.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//		})
//
//		It("General Backup new ReplicaSet", func() {
//			By("Creating New ReplicaSet Backup")
//			testRSBackup()
//
//			By("Creating Restore Session")
//			err := f.CreateRestoreSession(restoreSession)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for initContainer")
//			f.EventuallyReplicaSet(rs.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))
//
//			By("Waiting for restore to succeed")
//			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))
//			err = util.WaitUntilReplicaSetReady(f.KubeClient, rs.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("checking the workload data has been restored")
//			restoredData, err = f.ReadSampleDataFromMountedDirectory(rs.ObjectMeta, framework.GetPathsFromRestoreSession(&restoreSession), apis.KindReplicaSet)
//			Expect(err).NotTo(HaveOccurred())
//			Expect(restoredData).To(Not(BeEmpty()))
//
//			By("Verifying restored data is same as original data")
//			Expect(restoredData).To(BeEquivalentTo(sampleData))
//
//		})
//	})
//
//	Context("Leader election and backup && restore for ReplicaSet", func() {
//		BeforeEach(func() {
//			pvc = f.GetPersistentVolumeClaim()
//			err := f.CreatePersistentVolumeClaim(pvc)
//			Expect(err).NotTo(HaveOccurred())
//			rs = f.ReplicaSet(pvc.Name)
//
//			targetref = v1beta1.TargetRef{
//				APIVersion: "apps/v1",
//				Kind:       apis.KindReplicaSet,
//				Name:       rs.Name,
//			}
//		})
//
//		AfterEach(func() {
//			err := f.DeleteReplicaSet(rs.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilReplicaSetDeleted(f.KubeClient, rs.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			err = f.DeleteRepository(repo)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilRepositoryDeleted(f.StashClient, repo)
//			Expect(err).NotTo(HaveOccurred())
//
//			err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilRestoreSessionDeleted(f.StashClient, restoreSession.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//		})
//
//		It("Should leader elect and Backup new ReplicaSet", func() {
//			rs.Spec.Replicas = types.Int32P(2) // two replicas
//			By("Creating ReplicationController " + rs.Name)
//			_, err := f.CreateReplicaSet(rs)
//			Expect(err).NotTo(HaveOccurred())
//			err = util.WaitUntilReplicaSetReady(f.KubeClient, rs.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Creating sample data inside workload")
//			err = f.CreateSampleDataInsideWorkload(rs.ObjectMeta, apis.KindReplicaSet)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Creating storage Secret " + cred.Name)
//			err = f.CreateSecret(cred)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Creating new repository")
//			err = f.CreateRepository(repo)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Creating BackupConfiguration" + backupCfg.Name)
//			err = f.CreateBackupConfiguration(backupCfg)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for sidecar")
//			f.EventuallyReplicaSet(rs.ObjectMeta).Should(matcher.HaveSidecar(util.StashContainer))
//
//			By("Waiting for leader election")
//			f.CheckLeaderElection(rs.ObjectMeta, apis.KindReplicaSet, v1beta1.ResourceKindBackupConfiguration)
//
//			By("Waiting for BackupSession")
//			f.EventuallyBackupSessionCreated(backupCfg.ObjectMeta).Should(BeTrue())
//			bs, err := f.GetBackupSession(backupCfg.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Check for succeeded BackupSession")
//			f.EventuallyBackupSessionPhase(bs.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))
//
//			By("Check for repository status updated")
//			f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
//
//			By("Delete BackupConfiguration")
//			err = f.DeleteBackupConfiguration(backupCfg)
//			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for sidecar to be removed")
//			f.EventuallyReplicaSet(rs.ObjectMeta).ShouldNot(matcher.HaveSidecar(util.StashContainer))
//			err = util.WaitUntilReplicaSetReady(f.KubeClient, rs.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Delete sample data from workload")
//			err = f.CleanupSampleDataFromWorkload(rs.ObjectMeta, apis.KindReplicaSet)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Creating Restore Session")
//			err = f.CreateRestoreSession(restoreSession)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for initContainer")
//			f.EventuallyReplicaSet(rs.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))
//
//			By("Waiting for restore to succeed")
//			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))
//
//		})
//	})
//
//	Context("Restore data on different ReplicaSet", func() {
//		BeforeEach(func() {
//			pvc = f.GetPersistentVolumeClaim()
//			err := f.CreatePersistentVolumeClaim(pvc)
//			Expect(err).NotTo(HaveOccurred())
//			rs = f.ReplicaSet(pvc.Name)
//
//			pvc = f.GetPersistentVolumeClaim()
//			err = f.CreatePersistentVolumeClaim(pvc)
//			Expect(err).NotTo(HaveOccurred())
//			recoveredRS = f.ReplicaSet(pvc.Name)
//			recoveredRS.Spec.Selector = &metav1.LabelSelector{
//				MatchLabels: map[string]string{
//					"replicaset": "recovered",
//				},
//			}
//			recoveredRS.Spec.Template.Labels = map[string]string{
//				"replicaset": "recovered",
//			}
//
//			targetref = v1beta1.TargetRef{
//				APIVersion: "apps/v1",
//				Kind:       apis.KindReplicaSet,
//				Name:       rs.Name,
//			}
//		})
//
//		AfterEach(func() {
//			err := f.DeleteReplicaSet(rs.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilReplicaSetDeleted(f.KubeClient, rs.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			err = f.DeleteReplicaSet(recoveredRS.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilReplicaSetDeleted(f.KubeClient, recoveredRS.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			err = f.DeleteRepository(repo)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilRepositoryDeleted(f.StashClient, repo)
//			Expect(err).NotTo(HaveOccurred())
//
//			err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilRestoreSessionDeleted(f.StashClient, restoreSession.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//		})
//
//		It("Restore data on different ReplicaSet", func() {
//			By("Creating New ReplicaSet Backup")
//			testRSBackup()
//
//			By("Creating another ReplicaSet " + recoveredRS.Name)
//			_, err := f.CreateReplicaSet(recoveredRS)
//			Expect(err).NotTo(HaveOccurred())
//			err = util.WaitUntilReplicaSetReady(f.KubeClient, recoveredRS.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			restoreSession.Spec.Target.Ref.Name = recoveredRS.Name
//
//			By("Creating Restore Session")
//			err = f.CreateRestoreSession(restoreSession)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for initContainer")
//			f.EventuallyReplicaSet(recoveredRS.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))
//
//			By("Waiting for restore to succeed")
//			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))
//			err = util.WaitUntilReplicaSetReady(f.KubeClient, recoveredRS.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("checking the workload data has been restored")
//			restoredData, err = f.ReadSampleDataFromMountedDirectory(recoveredRS.ObjectMeta, framework.GetPathsFromRestoreSession(&restoreSession), apis.KindReplicaSet)
//			Expect(err).NotTo(HaveOccurred())
//			Expect(restoredData).To(Not(BeEmpty()))
//
//			By("Compare between restore data and sample data")
//			Expect(restoredData).To(BeEquivalentTo(sampleData))
//		})
//	})
//})
