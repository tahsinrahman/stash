package workloads
//
//import (
//	"github.com/appscode/go/types"
//	. "github.com/onsi/ginkgo"
//	. "github.com/onsi/gomega"
//	apps "k8s.io/api/apps/v1"
//	core "k8s.io/api/core/v1"
//	"stash.appscode.dev/stash/apis"
//	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
//	"stash.appscode.dev/stash/apis/stash/v1beta1"
//	"stash.appscode.dev/stash/pkg/util"
//	"stash.appscode.dev/stash/test/e2e/framework"
//	"stash.appscode.dev/stash/test/e2e/matcher"
//)
//
//var _ = XDescribe("StatefulSet", func() {
//
//	var (
//		f              *framework.Invocation
//		cred           core.Secret
//		ss             apps.StatefulSet
//		recoveredss    apps.StatefulSet
//		repo           *api.Repository
//		backupCfg      v1beta1.BackupConfiguration
//		restoreSession v1beta1.RestoreSession
//		pvc            *core.PersistentVolumeClaim
//		targetref      v1beta1.TargetRef
//		rules          []v1beta1.Rule
//		svc            core.Service
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
//
//		By("Creating service " + svc.Name)
//		err := f.CreateOrPatchService(svc)
//		Expect(err).NotTo(HaveOccurred())
//
//		pvc = f.GetPersistentVolumeClaim()
//		err = f.CreatePersistentVolumeClaim(pvc)
//		Expect(err).NotTo(HaveOccurred())
//		repo = f.GetLocalRepository(cred.Name, pvc.Name)
//
//		backupCfg = f.BackupConfiguration(repo.Name, targetref)
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
//		testStatefulsetBackup = func() {
//			ss.Spec.Replicas = types.Int32P(3)
//			By("Create Statefulset with multiple replica" + ss.Name)
//			_, err := f.CreateStatefulSet(ss)
//			Expect(err).NotTo(HaveOccurred())
//			err = util.WaitUntilStatefulSetReady(f.KubeClient, ss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Creating Sample data in inside pod")
//			err = f.CreateSampleDataInsideWorkload(ss.ObjectMeta, apis.KindStatefulSet)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Reading sample data from /source/data mountPath inside workload")
//			sampleData, err = f.ReadSampleDataFromFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
//			Expect(err).NotTo(HaveOccurred())
//			Expect(sampleData).ShouldNot(BeEmpty())
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
//			f.EventuallyStatefulSet(ss.ObjectMeta).Should(matcher.HaveSidecar(util.StashContainer))
//
//			By("Waiting for BackupSession")
//			f.EventuallyBackupSessionCreated(backupCfg.ObjectMeta).Should(BeTrue())
//			bs, err := f.GetBackupSession(backupCfg.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Check for repository status updated")
//			f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
//
//			By("Check for succeeded BackupSession")
//			f.EventuallyBackupSessionPhase(bs.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))
//
//			By("Delete BackupConfiguration")
//			err = f.DeleteBackupConfiguration(backupCfg)
//			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
//			Expect(err).ShouldNot(HaveOccurred())
//
//			By("Deleting sample data from pod")
//			err = f.CleanupSampleDataFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
//			Expect(err).NotTo(HaveOccurred())
//
//		}
//	)
//
//	Context("General Backup new StatefulSet", func() {
//		BeforeEach(func() {
//			svc = f.HeadlessService()
//			ss = f.StatefulSetForV1beta1API()
//			targetref = v1beta1.TargetRef{
//				Name:       ss.Name,
//				Kind:       apis.KindStatefulSet,
//				APIVersion: "apps/v1",
//			}
//			rules = []v1beta1.Rule{
//				{
//					Paths: []string{
//						framework.TestSourceDataMountPath,
//					},
//				},
//			}
//		})
//
//		AfterEach(func() {
//			err := f.DeleteStatefulSet(ss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, ss.ObjectMeta)
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
//		It("General Backup new Statefulset", func() {
//			By("Creating Statefulset Backup")
//			testStatefulsetBackup()
//
//			By("Creating Restore Session")
//			err := f.CreateRestoreSession(restoreSession)
//			Expect(err).NotTo(HaveOccurred())
//			err = util.WaitUntilStatefulSetReady(f.KubeClient, ss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for initContainer")
//			f.EventuallyStatefulSet(ss.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for restore to succeed")
//			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))
//			err = util.WaitUntilStatefulSetReady(f.KubeClient, ss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("checking the workload data has been restored")
//			restoredData, err = f.ReadSampleDataFromMountedDirectory(ss.ObjectMeta, framework.GetPathsFromRestoreSession(&restoreSession), apis.KindStatefulSet)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Compare between restore data and sample data")
//			Expect(restoredData).To(BeEquivalentTo(sampleData))
//
//		})
//	})
//
//	Context("Restore data on different StatefulSet", func() {
//		BeforeEach(func() {
//			svc = f.HeadlessService()
//			ss = f.StatefulSetForV1beta1API()
//			recoveredss = f.StatefulSetForV1beta1API()
//			targetref = v1beta1.TargetRef{
//				Name:       ss.Name,
//				Kind:       apis.KindStatefulSet,
//				APIVersion: "apps/v1",
//			}
//			rules = []v1beta1.Rule{
//				{
//					Paths: []string{
//						framework.TestSourceDataMountPath,
//					},
//				},
//			}
//		})
//
//		AfterEach(func() {
//			err := f.DeleteStatefulSet(ss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, ss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = f.DeleteStatefulSet(recoveredss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, recoveredss.ObjectMeta)
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
//		It("General Backup new StatefulSet", func() {
//			By("Creating StatefulSet Backup")
//			testStatefulsetBackup()
//
//			By("Creating another StatefulSet " + recoveredss.Name)
//			_, err := f.CreateStatefulSet(recoveredss)
//			Expect(err).NotTo(HaveOccurred())
//
//			restoreSession.Spec.Target.Ref.Name = recoveredss.Name
//
//			By("Creating Restore Session")
//			err = f.CreateRestoreSession(restoreSession)
//			Expect(err).NotTo(HaveOccurred())
//			err = util.WaitUntilStatefulSetReady(f.KubeClient, recoveredss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for initContainer")
//			f.EventuallyStatefulSet(recoveredss.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for restore to succeed")
//			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))
//			err = util.WaitUntilStatefulSetReady(f.KubeClient, recoveredss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("checking the workload data has been restored")
//			restoredData, err = f.ReadSampleDataFromMountedDirectory(recoveredss.ObjectMeta, framework.GetPathsFromRestoreSession(&restoreSession), apis.KindStatefulSet)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Compare between restore data and sample data")
//			Expect(restoredData).To(BeEquivalentTo(sampleData))
//
//		})
//	})
//
//	Context("Restore data on Scaled Up StatefulSet", func() {
//		BeforeEach(func() {
//			svc = f.HeadlessService()
//			ss = f.StatefulSetForV1beta1API()
//			recoveredss = f.StatefulSetForV1beta1API()
//			targetref = v1beta1.TargetRef{
//				Name:       ss.Name,
//				Kind:       apis.KindStatefulSet,
//				APIVersion: "apps/v1",
//			}
//			rules = []v1beta1.Rule{
//				{
//					TargetHosts: []string{
//						"host-3",
//						"host-4",
//					},
//					SourceHost: "host-1",
//					Paths: []string{
//						framework.TestSourceDataMountPath,
//					},
//				},
//				{
//					TargetHosts: []string{},
//					SourceHost:  "",
//					Paths: []string{
//						framework.TestSourceDataMountPath,
//					},
//				},
//			}
//		})
//
//		AfterEach(func() {
//			err := f.DeleteStatefulSet(ss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, ss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = f.DeleteStatefulSet(recoveredss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//			err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, recoveredss.ObjectMeta)
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
//		It("General Backup new StatefulSet", func() {
//			By("Creating StatefulSet Backup")
//			testStatefulsetBackup()
//
//			By("Creating another StatefulSet " + recoveredss.Name)
//			recoveredss.Spec.Replicas = types.Int32P(5)
//			_, err := f.CreateStatefulSet(recoveredss)
//			Expect(err).NotTo(HaveOccurred())
//			err = util.WaitUntilStatefulSetReady(f.KubeClient, recoveredss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			restoreSession.Spec.Target.Ref.Name = recoveredss.Name
//
//			By("Creating Restore Session")
//			err = f.CreateRestoreSession(restoreSession)
//			Expect(err).NotTo(HaveOccurred())
//			err = util.WaitUntilStatefulSetReady(f.KubeClient, recoveredss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for initContainer")
//			f.EventuallyStatefulSet(recoveredss.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Waiting for restore to succeed")
//			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))
//			err = util.WaitUntilStatefulSetReady(f.KubeClient, recoveredss.ObjectMeta)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("checking the workload data has been restored")
//			restoredData, err = f.ReadSampleDataFromMountedDirectory(recoveredss.ObjectMeta, framework.GetPathsFromRestoreSession(&restoreSession), apis.KindStatefulSet)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Comparing between first and second StatefulSet sample data")
//			Expect(sampleData).Should(BeEquivalentTo(restoredData[0 : (len(restoredData)-len(sampleData))+1]))
//			data := make([]string, 0)
//			data = append(data, sampleData[1])
//			data = append(data, sampleData[1])
//			Expect(data).Should(BeEquivalentTo(restoredData[(len(restoredData) - len(sampleData) + 1):]))
//		})
//	})
//
//})
