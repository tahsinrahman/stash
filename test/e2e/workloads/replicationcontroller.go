package workloads
//
//import (
//	"github.com/appscode/go/types"
//	. "github.com/onsi/ginkgo"
//	. "github.com/onsi/gomega"
//	core "k8s.io/api/core/v1"
//	"stash.appscode.dev/stash/apis"
//	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
//	"stash.appscode.dev/stash/apis/stash/v1beta1"
//	"stash.appscode.dev/stash/pkg/util"
//	"stash.appscode.dev/stash/test/e2e/framework"
//	"stash.appscode.dev/stash/test/e2e/matcher"
//)
//
//var _ = XDescribe("ReplicationController", func() {
//
//	var (
//		f              *framework.Invocation
//		cred           core.Secret
//		repo           *api.Repository
//		backupCfg      v1beta1.BackupConfiguration
//		restoreSession v1beta1.RestoreSession
//		pvc            *core.PersistentVolumeClaim
//		targetref      v1beta1.TargetRef
//		rules          []v1beta1.Rule
//		rc             core.ReplicationController
//		recoveredRC    core.ReplicationController
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
//		testRCBackup = func() {
//			By("Creating ReplicationController " + rc.Name)
//			_, err := f.CreateReplicationController(rc)
//			Expect(err).NotTo(HaveOccurred())
//			err = util.WaitUntilRCReady(f.KubeClient, rc.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Creating sample data inside workload")
//			err = f.CreateSampleDataInsideWorkload(rc.ObjectMeta, apis.KindReplicationController)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Reading sample data from /source/data mountPath inside workload")
//			sampleData, err = f.ReadSampleDataFromFromWorkload(rc.ObjectMeta, apis.KindReplicationController)
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
//			f.EventuallyReplicationController(rc.ObjectMeta).Should(matcher.HaveSidecar(util.StashContainer))
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
//			f.EventuallyRepository(&rc).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
//
//			By("Delete BackupConfiguration")
//			err = f.DeleteBackupConfiguration(backupCfg)
//			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting to remove sidecar")
//			f.EventuallyReplicationController(rc.ObjectMeta).ShouldNot(matcher.HaveSidecar(util.StashContainer))
//			err = util.WaitUntilRCReady(f.KubeClient, rc.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//		}
//	)
//
//	Context("Backup && Restore for ReplicationController", func() {
//		BeforeEach(func() {
//			pvc = f.GetPersistentVolumeClaim()
//			err := f.CreatePersistentVolumeClaim(pvc)
//			Expect(err).NotTo(HaveOccurred())
//			rc = f.ReplicationController(pvc.Name)
//
//			targetref = v1beta1.TargetRef{
//				APIVersion: "v1",
//				Kind:       apis.KindReplicationController,
//				Name:       rc.Name,
//			}
//		})
//
//		AfterEach(func() {
//			err := f.DeleteReplicationController(rc.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilReplicationControllerDeleted(f.KubeClient, rc.ObjectMeta)
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
//		It("General Backup new ReplicationController", func() {
//			By("Creating New ReplicationController Backup")
//			testRCBackup()
//
//			By("Remove sample data from workload")
//			err := f.CleanupSampleDataFromWorkload(rc.ObjectMeta, apis.KindReplicationController)
//			Expect(err).NotTo(HaveOccurred())
//			By("Creating Restore Session")
//			err = f.CreateRestoreSession(restoreSession)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for initContainer")
//			f.EventuallyReplicationController(rc.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))
//
//			By("Waiting for restore to succeed")
//			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))
//			err = util.WaitUntilRCReady(f.KubeClient, rc.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("checking the workload data has been restored")
//			restoredData, err = f.ReadSampleDataFromMountedDirectory(rc.ObjectMeta, framework.GetPathsFromRestoreSession(&restoreSession), apis.KindReplicationController)
//			Expect(err).NotTo(HaveOccurred())
//			Expect(restoredData).NotTo(BeEmpty())
//
//			By("Verifying restored data is same as original data")
//			Expect(restoredData).To(BeEquivalentTo(sampleData))
//		})
//	})
//
//	Context("Leader election and backup && restore for ReplicationController", func() {
//		BeforeEach(func() {
//			pvc = f.GetPersistentVolumeClaim()
//			err := f.CreatePersistentVolumeClaim(pvc)
//			Expect(err).NotTo(HaveOccurred())
//			rc = f.ReplicationController(pvc.Name)
//
//			targetref = v1beta1.TargetRef{
//				APIVersion: "v1",
//				Kind:       apis.KindReplicationController,
//				Name:       rc.Name,
//			}
//		})
//
//		AfterEach(func() {
//			err := f.DeleteReplicationController(rc.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilReplicationControllerDeleted(f.KubeClient, rc.ObjectMeta)
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
//		It("Should leader elect and Backup new ReplicationController", func() {
//			rc.Spec.Replicas = types.Int32P(2) // two replicas
//			By("Creating ReplicationController " + rc.Name)
//			_, err := f.CreateReplicationController(rc)
//			Expect(err).NotTo(HaveOccurred())
//			err = util.WaitUntilRCReady(f.KubeClient, rc.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Creating sample data inside workload")
//			err = f.CreateSampleDataInsideWorkload(rc.ObjectMeta, apis.KindReplicationController)
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
//			f.EventuallyReplicationController(rc.ObjectMeta).Should(matcher.HaveSidecar(util.StashContainer))
//
//			By("Waiting for leader election")
//			f.CheckLeaderElection(rc.ObjectMeta, apis.KindReplicationController, v1beta1.ResourceKindBackupConfiguration)
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
//			f.EventuallyRepository(&rc).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
//
//			By("Delete BackupConfiguration")
//			err = f.DeleteBackupConfiguration(backupCfg)
//			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for sidecar to be removed")
//			f.EventuallyReplicationController(rc.ObjectMeta).ShouldNot(matcher.HaveSidecar(util.StashContainer))
//			err = util.WaitUntilRCReady(f.KubeClient, rc.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Delete sample data from workload")
//			err = f.CleanupSampleDataFromWorkload(rc.ObjectMeta, apis.KindReplicationController)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Creating Restore Session")
//			err = f.CreateRestoreSession(restoreSession)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for initContainer")
//			f.EventuallyReplicationController(rc.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))
//
//			By("Waiting for restore to succeed")
//			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))
//		})
//	})
//
//	Context("Restore data on different ReplicationController", func() {
//		BeforeEach(func() {
//			pvc = f.GetPersistentVolumeClaim()
//			err := f.CreatePersistentVolumeClaim(pvc)
//			Expect(err).NotTo(HaveOccurred())
//			rc = f.ReplicationController(pvc.Name)
//
//			pvc = f.GetPersistentVolumeClaim()
//			err = f.CreatePersistentVolumeClaim(pvc)
//			Expect(err).NotTo(HaveOccurred())
//			recoveredRC = f.ReplicationController(pvc.Name)
//			recoveredRC.Spec.Selector = map[string]string{
//				"rc": "recovered",
//			}
//			recoveredRC.Spec.Template.Labels = map[string]string{
//				"rc": "recovered",
//			}
//			targetref = v1beta1.TargetRef{
//				APIVersion: "v1",
//				Kind:       apis.KindReplicationController,
//				Name:       rc.Name,
//			}
//
//		})
//
//		AfterEach(func() {
//			err := f.DeleteReplicationController(rc.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilReplicationControllerDeleted(f.KubeClient, rc.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			err = f.DeleteReplicationController(recoveredRC.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilReplicationControllerDeleted(f.KubeClient, recoveredRC.ObjectMeta)
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
//		It("Restore data on different ReplicationController", func() {
//			By("Creating New ReplicationController Backup")
//			testRCBackup()
//
//			By("Creating another ReplicationController " + recoveredRC.Name)
//			_, err := f.CreateReplicationController(recoveredRC)
//			Expect(err).NotTo(HaveOccurred())
//			err = util.WaitUntilRCReady(f.KubeClient, recoveredRC.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			restoreSession.Spec.Target.Ref.Name = recoveredRC.Name
//
//			By("Creating Restore Session")
//			err = f.CreateRestoreSession(restoreSession)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for initContainer")
//			f.EventuallyReplicationController(recoveredRC.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))
//
//			By("Waiting for restore to succeed")
//			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))
//			err = util.WaitUntilRCReady(f.KubeClient, recoveredRC.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("checking the workload data has been restored")
//			restoredData, err = f.ReadSampleDataFromMountedDirectory(recoveredRC.ObjectMeta, framework.GetPathsFromRestoreSession(&restoreSession), apis.KindReplicationController)
//			Expect(err).NotTo(HaveOccurred())
//			Expect(restoredData).NotTo(BeEmpty())
//
//			By("Compare between restore data and sample data")
//			Expect(restoredData).To(BeEquivalentTo(sampleData))
//		})
//	})
//})
