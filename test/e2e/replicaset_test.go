package e2e_test

import (
	"fmt"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apps_util "kmodules.xyz/client-go/apps/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"stash.appscode.dev/stash/apis"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"
)

var _ = XDescribe("ReplicaSet", func() {
	var (
		err          error
		f            *framework.Invocation
		restic       api.Restic
		secondRestic api.Restic
		cred         core.Secret
		rs           apps.ReplicaSet
		recovery     api.Recovery
		localRef     api.LocalTypedReference
	)

	BeforeEach(func() {
		f = root.Invoke()
	})
	AfterEach(func() {
		err := framework.WaitUntilReplicaSetDeleted(f.KubeClient, rs.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, rs.Namespace, f.AppLabel())
		Expect(err).NotTo(HaveOccurred())

		err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		f.DeleteRepositories(f.ReplicaSetRepos(&rs))

		err = framework.WaitUntilRepositoriesDeleted(f.StashClient, f.ReplicaSetRepos(&rs))
		Expect(err).NotTo(HaveOccurred())

	})
	JustBeforeEach(func() {
		if missing, _ := BeZero().Match(cred); missing {
			Skip("Missing repository credential")
		}
		restic.Spec.Backend.StorageSecretName = cred.Name
		secondRestic.Spec.Backend.StorageSecretName = cred.Name
		pvc := f.GetPersistentVolumeClaim()
		err := f.CreatePersistentVolumeClaim(pvc)
		Expect(err).NotTo(HaveOccurred())
		rs = f.ReplicaSet(pvc.Name)
		localRef = api.LocalTypedReference{
			Kind: apis.KindReplicaSet,
			Name: rs.Name,
		}
	})

	var (
		shouldBackupNewReplicaSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicaSet " + rs.Name)
			_, err = f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos := f.ReplicaSetRepos(&rs)
			Expect(repos).NotTo(BeEmpty())
			f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldBackupExistingReplicaSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicaSet " + rs.Name)
			_, err = f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos := f.ReplicaSetRepos(&rs)
			Expect(repos).NotTo(BeEmpty())
			f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldStopBackup = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicaSet " + rs.Name)
			_, err = f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			f.EventuallyReplicaSet(rs.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldStopBackupIfLabelChanged = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicaSet " + rs.Name)
			_, err = f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Removing labels of ReplicaSet " + rs.Name)
			_, _, err = apps_util.PatchReplicaSet(f.KubeClient, &rs, func(in *apps.ReplicaSet) *apps.ReplicaSet {
				in.Labels = map[string]string{
					"app": "unmatched",
				}
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			f.EventuallyReplicaSet(rs.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldStopBackupIfSelectorChanged = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicaSet " + rs.Name)
			_, err = f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Change selector of Restic " + restic.Name)
			err = f.UpdateRestic(restic.ObjectMeta, func(in *api.Restic) *api.Restic {
				in.Spec.Selector = metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "unmatched",
					},
				}
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting to remove sidecar")
			f.EventuallyReplicaSet(rs.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldElectLeaderAndBackupRS = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			rs.Spec.Replicas = types.Int32P(2) // two replicas
			By("Creating ReplicaSet " + rs.Name)
			_, err = f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for leader election")
			f.CheckLeaderElection(rs.ObjectMeta, apis.KindReplicaSet, api.ResourceKindRestic)

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos := f.ReplicaSetRepos(&rs)
			Expect(repos).NotTo(BeEmpty())
			f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldMutateAndBackupNewReplicaSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicaSet " + rs.Name)
			obj, err := f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())

			// sidecar should be added as soon as rs created, we don't need to wait for it
			By("Checking sidecar created")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos := f.ReplicaSetRepos(&rs)
			Expect(repos).NotTo(BeEmpty())
			f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldNotMutateNewReplicaSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicaSet " + rs.Name)
			obj, err := f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar not added")
			Expect(obj).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldRejectToCreateNewReplicaSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating first restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating second restic " + secondRestic.Name)
			err = f.CreateRestic(secondRestic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicaSet " + rs.Name)
			_, err := f.CreateReplicaSet(rs)
			Expect(err).To(HaveOccurred())
		}

		shouldRemoveSidecarInstantly = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicaSet " + rs.Name)
			obj, err := f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar added")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Removing labels of ReplicaSet " + rs.Name)
			obj, _, err = apps_util.PatchReplicaSet(f.KubeClient, &rs, func(in *apps.ReplicaSet) *apps.ReplicaSet {
				in.Labels = map[string]string{
					"app": "unmatched",
				}
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar has removed")
			Expect(obj).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldAddSidecarInstantly = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicaSet " + rs.Name)
			previousLabel := rs.Labels
			rs.Labels = map[string]string{
				"app": "unmatched",
			}
			obj, err := f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar not added")
			Expect(obj).ShouldNot(HaveSidecar(util.StashContainer))

			By("Adding label to match restic" + rs.Name)
			obj, _, err = apps_util.PatchReplicaSet(f.KubeClient, &rs, func(in *apps.ReplicaSet) *apps.ReplicaSet {
				in.Labels = previousLabel
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar added")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
		}
	)

	Describe("Creating restic for", func() {
		AfterEach(func() {
			f.DeleteReplicaSet(rs.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should backup new ReplicaSet`, shouldBackupNewReplicaSet)
			It(`should backup existing ReplicaSet`, shouldBackupExistingReplicaSet)
		})

		Context(`"S3" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForS3Backend()
				restic = f.ResticForS3Backend()
			})
			It(`should backup new ReplicaSet`, shouldBackupNewReplicaSet)
			It(`should backup existing ReplicaSet`, shouldBackupExistingReplicaSet)
		})

		Context(`"DO" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForDOBackend()
				restic = f.ResticForDOBackend()
			})
			It(`should backup new ReplicaSet`, shouldBackupNewReplicaSet)
			It(`should backup existing ReplicaSet`, shouldBackupExistingReplicaSet)
		})

		Context(`"GCS" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForGCSBackend()
				restic = f.ResticForGCSBackend()
			})
			It(`should backup new ReplicaSet`, shouldBackupNewReplicaSet)
			It(`should backup existing ReplicaSet`, shouldBackupExistingReplicaSet)
		})

		Context(`"Azure" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForAzureBackend()
				restic = f.ResticForAzureBackend()
			})
			It(`should backup new ReplicaSet`, shouldBackupNewReplicaSet)
			It(`should backup existing ReplicaSet`, shouldBackupExistingReplicaSet)
		})

		Context(`"Swift" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForSwiftBackend()
				restic = f.ResticForSwiftBackend()
			})
			It(`should backup new ReplicaSet`, shouldBackupNewReplicaSet)
			It(`should backup existing ReplicaSet`, shouldBackupExistingReplicaSet)
		})

		Context(`"B2" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForB2Backend()
				restic = f.ResticForB2Backend()
			})
			It(`should backup new ReplicaSet`, shouldBackupNewReplicaSet)
			It(`should backup existing ReplicaSet`, shouldBackupExistingReplicaSet)
		})
	})

	Describe("Changing ReplicaSet labels", func() {
		AfterEach(func() {
			f.DeleteReplicaSet(rs.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})
		BeforeEach(func() {
			cred = f.SecretForLocalBackend()
			restic = f.ResticForLocalBackend()
		})
		It(`should stop backup`, shouldStopBackupIfLabelChanged)
	})

	Describe("Changing Restic selector", func() {
		AfterEach(func() {
			f.DeleteReplicaSet(rs.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})
		BeforeEach(func() {
			cred = f.SecretForLocalBackend()
			restic = f.ResticForLocalBackend()
		})
		It(`should stop backup`, shouldStopBackupIfSelectorChanged)
	})

	Describe("Deleting restic for", func() {
		AfterEach(func() {
			f.DeleteReplicaSet(rs.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"S3" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForS3Backend()
				restic = f.ResticForS3Backend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"DO" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForDOBackend()
				restic = f.ResticForDOBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"GCS" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForGCSBackend()
				restic = f.ResticForGCSBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"Azure" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForAzureBackend()
				restic = f.ResticForAzureBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"Swift" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForSwiftBackend()
				restic = f.ResticForSwiftBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"B2" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForB2Backend()
				restic = f.ResticForB2Backend()
			})
			It(`should stop backup`, shouldStopBackup)
		})
	})

	Describe("Leader election for", func() {
		AfterEach(func() {
			f.DeleteReplicaSet(rs.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should elect leader and backup new RS`, shouldElectLeaderAndBackupRS)
		})
	})

	Describe("Stash Webhook for", func() {
		AfterEach(func() {
			f.DeleteReplicaSet(rs.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteRestic(secondRestic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)

			err := framework.WaitUntilResticDeleted(f.StashClient, secondRestic.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
				secondRestic = restic
				secondRestic.Name = "second-restic"
			})
			It("should mutate and backup new ReplicaSet", shouldMutateAndBackupNewReplicaSet)
			It("should not mutate new ReplicaSet if no restic select it", shouldNotMutateNewReplicaSet)
			It("should reject to create new ReplicaSet if multiple restic select it", shouldRejectToCreateNewReplicaSet)
			It("should remove sidecar instantly if label change to match no restic", shouldRemoveSidecarInstantly)
			It("should add sidecar instantly if label change to match single restic", shouldAddSidecarInstantly)
		})
	})

	Describe("Offline backup for", func() {
		AfterEach(func() {
			f.DeleteReplicaSet(rs.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`Single Replica`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.Type = api.BackupOffline
				restic.Spec.Schedule = "@every 3m"
			})
			It(`should backup new ReplicaSet`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating ReplicaSet " + rs.Name)
				_, err = f.CreateReplicaSet(rs)
				Expect(err).NotTo(HaveOccurred())

				cronJobName := util.ScaledownCronPrefix + restic.Name
				By("Checking cron job created: " + cronJobName)
				Eventually(func() error {
					_, err := f.KubeClient.BatchV1beta1().CronJobs(restic.Namespace).Get(cronJobName, metav1.GetOptions{})
					return err
				}).Should(BeNil())

				start := time.Now()
				for i := 1; i <= 3; i++ {
					fmt.Printf("=============== Waiting for backup no: %d =============\n", i)
					By("Waiting for scale down rs to 0 replica")
					f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveReplica(0))

					By("Waiting for scale up rs to 1 replica")
					f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveReplica(1))

					By("Waiting for init-container")
					f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

					By("Waiting for Repository CRD")
					f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

					By("Waiting for backup to complete")
					f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically("==", i)))
				}
				elapsedTime := time.Since(start).Minutes()

				// backup is scheduled for every 3 minutes.
				// so 3 backup by cronJob should not take more than 9 minutes + some overhead.(let 1 minute overhead for each backup)
				Expect(elapsedTime).Should(BeNumerically("<=", 9+3))

				By("Waiting for backup event")
				repos := f.ReplicaSetRepos(&rs)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Waiting for scale up rs to original replica")
				f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveReplica(int(*rs.Spec.Replicas)))
			})
		})

		Context("Multiple Replica", func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.Type = api.BackupOffline
				restic.Spec.Schedule = "@every 3m"
			})
			It(`should backup new ReplicaSet`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating ReplicaSet " + rs.Name)
				rs.Spec.Replicas = types.Int32P(3)
				_, err = f.CreateReplicaSet(rs)
				Expect(err).NotTo(HaveOccurred())

				cronJobName := util.ScaledownCronPrefix + restic.Name
				By("Checking cron job created: " + cronJobName)
				Eventually(func() error {
					_, err := f.KubeClient.BatchV1beta1().CronJobs(restic.Namespace).Get(cronJobName, metav1.GetOptions{})
					return err
				}).Should(BeNil())

				start := time.Now()
				for i := 1; i <= 3; i++ {
					fmt.Printf("=============== Waiting for backup no: %d =============\n", i)
					By("Waiting for scale down rs to 0 replica")
					f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveReplica(0))

					By("Waiting for scale up rs to 1 replica")
					f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveReplica(1))

					By("Waiting for init-container")
					f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

					By("Waiting for Repository CRD")
					f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

					By("Waiting for backup to complete")
					f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically("==", i)))
				}
				elapsedTime := time.Since(start).Minutes()

				// backup is scheduled for every 3 minutes.
				// so 3 backup by cronJob should not take more than 9 minutes + some overhead.(let 1 minute overhead for each backup)
				Expect(elapsedTime).Should(BeNumerically("<=", 9+3))

				By("Waiting for backup event")
				repos := f.ReplicaSetRepos(&rs)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Waiting for scale up rs to original replica")
				f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveReplica(int(*rs.Spec.Replicas)))
			})
		})
	})

	Describe("Pause Restic to stop backup", func() {
		Context(`"Local" backend`, func() {
			AfterEach(func() {
				f.DeleteReplicaSet(rs.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
			})
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should be able to Pause and Resume backup`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating ReplicaSet " + rs.Name)
				_, err = f.CreateReplicaSet(rs)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.ReplicaSetRepos(&rs)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By(`Patching Restic with "paused: true"`)
				err = f.CreateOrPatchRestic(restic.ObjectMeta, func(in *api.Restic) *api.Restic {
					in.Spec.Paused = true
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				// wait some time for ongoing backup
				time.Sleep(time.Second * 30)
				repos = f.ReplicaSetRepos(&rs)
				Expect(repos).NotTo(BeEmpty())

				previousBackupCount := repos[0].Status.BackupCount

				By("Waiting 2 minutes")
				time.Sleep(2 * time.Minute)

				By("Checking that Backup count has not changed")
				repos = f.ReplicaSetRepos(&rs)
				Expect(repos).NotTo(BeEmpty())
				Expect(repos[0].Status.BackupCount).Should(BeNumerically("==", previousBackupCount))

				By(`Patching Restic with "paused: false"`)
				err = f.CreateOrPatchRestic(restic.ObjectMeta, func(in *api.Restic) *api.Restic {
					in.Spec.Paused = false
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">", previousBackupCount)))

				By("Waiting for backup event")
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">", previousBackupCount)))

			})

		})
	})

	Describe("Create Repository CRD", func() {
		Context(`"Local" backend`, func() {
			AfterEach(func() {
				f.DeleteReplicaSet(rs.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
			})
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should create Repository CRD`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating ReplicaSet " + rs.Name)
				_, err = f.CreateReplicaSet(rs)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.ReplicaSetRepos(&rs)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
			})

		})
	})

	Describe("Complete Recovery", func() {
		Context(`"Local" backend, single fileGroup`, func() {
			AfterEach(func() {
				f.CleanupRecoveredVolume(rs.ObjectMeta)
				f.DeleteReplicaSet(rs.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
				f.DeleteRecovery(recovery.ObjectMeta)

				err := framework.WaitUntilRecoveryDeleted(f.StashClient, recovery.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
			})
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				recovery = f.RecoveryForRestic(restic)
			})
			It(`recovered volume should have same data`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating ReplicaSet " + rs.Name)
				_, err = f.CreateReplicaSet(rs)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.ReplicaSetRepos(&rs)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(rs.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic))
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting rs")
				f.DeleteReplicaSet(rs.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// wait until rs terminated
				err = framework.WaitUntilReplicaSetDeleted(f.KubeClient, rs.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, rs.Namespace, f.AppLabel())
				Expect(err).NotTo(HaveOccurred())

				recovery.Spec.Repository.Name = localRef.GetRepositoryCRDName("", "")
				recovery.Spec.Repository.Namespace = f.Namespace()

				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying rs with recovered volume")
				rs.Spec.Template.Spec.Volumes = f.RecoveredVolume()
				_, err = f.CreateReplicaSet(rs)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(rs.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic)).Should(BeEquivalentTo(previousData))
			})

		})

		Context(`"Local" backend, multiple fileGroup`, func() {
			AfterEach(func() {
				f.CleanupRecoveredVolume(rs.ObjectMeta)
				f.DeleteReplicaSet(rs.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
				f.DeleteRecovery(recovery.ObjectMeta)
			})
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.FileGroups = framework.FileGroupsForHostPathVolumeWithMultipleDirectory()
				recovery = f.RecoveryForRestic(restic)
			})
			It(`recovered volume should have same data`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				rs.Spec.Template.Spec.Volumes = f.HostPathVolumeWithMultipleDirectory()
				By("Creating ReplicaSet " + rs.Name)
				_, err = f.CreateReplicaSet(rs)
				Expect(err).NotTo(HaveOccurred())
				apps_util.WaitUntilReplicaSetReady(f.KubeClient, rs.ObjectMeta)

				By("Creating demo data in hostPath")
				err = f.CreateDemoData(rs.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.ReplicaSetRepos(&rs)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(rs.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic))
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting rs")
				f.DeleteReplicaSet(rs.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// wait until rs terminated
				err = framework.WaitUntilReplicaSetDeleted(f.KubeClient, rs.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, rs.Namespace, f.AppLabel())
				Expect(err).NotTo(HaveOccurred())

				recovery.Spec.Repository.Name = localRef.GetRepositoryCRDName("", "")
				recovery.Spec.Repository.Namespace = f.Namespace()

				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying rs with recovered volume")
				rs.Spec.Template.Spec.Volumes = f.RecoveredVolume()
				_, err = f.CreateReplicaSet(rs)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(rs.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic)).Should(BeEquivalentTo(previousData))
			})

		})
	})

	Describe("Recover from different namespace", func() {
		var (
			recoveryNamespace *core.Namespace
		)
		Context(`"Local" backend, single fileGroup`, func() {
			AfterEach(func() {
				f.CleanupRecoveredVolume(rs.ObjectMeta)
				f.DeleteReplicaSet(rs.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
				f.DeleteRecovery(recovery.ObjectMeta)
				f.DeleteNamespace(recoveryNamespace.Name)

				err = framework.WaitUntilNamespaceDeleted(f.KubeClient, recoveryNamespace.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
			})
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				recovery = f.RecoveryForRestic(restic)
				recoveryNamespace = f.NewNamespace(rand.WithUniqSuffix("recovery"))
			})
			It(`recovered volume should have same data`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating ReplicaSet " + rs.Name)
				_, err = f.CreateReplicaSet(rs)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.ReplicaSetRepos(&rs)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(rs.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic))
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting rs")
				f.DeleteReplicaSet(rs.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// wait until rs terminated
				err = framework.WaitUntilReplicaSetDeleted(f.KubeClient, rs.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, rs.Namespace, f.AppLabel())
				Expect(err).NotTo(HaveOccurred())

				recovery.Spec.Repository.Name = localRef.GetRepositoryCRDName("", "")
				recovery.Spec.Repository.Namespace = f.Namespace()

				By("Creating new namespace: " + recoveryNamespace.Name)
				err = f.CreateNamespace(recoveryNamespace)
				Expect(err).NotTo(HaveOccurred())

				By("Creating recovery " + recovery.Name)
				recovery.Namespace = recoveryNamespace.Name
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying rs with recovered volume")
				rs.Name = recoveryNamespace.Name
				rs.Spec.Template.Spec.Volumes = f.RecoveredVolume()
				_, err = f.CreateReplicaSet(rs)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(rs.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic)).Should(BeEquivalentTo(previousData))
			})

		})
	})
})
