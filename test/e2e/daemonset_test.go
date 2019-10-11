package e2e_test

import (
	"fmt"
	"time"

	"github.com/appscode/go/crypto/rand"
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

var _ = XDescribe("DaemonSet", func() {
	var (
		err          error
		f            *framework.Invocation
		restic       api.Restic
		secondRestic api.Restic
		cred         core.Secret
		daemon       apps.DaemonSet
		recovery     api.Recovery
		localRef     api.LocalTypedReference
	)

	BeforeEach(func() {
		f = root.Invoke()
	})
	AfterEach(func() {
		err := framework.WaitUntilDaemonSetDeleted(f.KubeClient, daemon.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, daemon.Namespace, f.AppLabel())
		Expect(err).NotTo(HaveOccurred())

		err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		f.DeleteRepositories(f.DaemonSetRepos(&daemon))

		err = framework.WaitUntilRepositoriesDeleted(f.StashClient, f.DaemonSetRepos(&daemon))
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
		daemon = f.DaemonSet(pvc.Name)
		localRef = api.LocalTypedReference{
			Kind: apis.KindDaemonSet,
			Name: daemon.Name,
		}
	})

	var (
		shouldBackupNewDaemonSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating DaemonSet " + daemon.Name)
			_, err = f.CreateDaemonSet(daemon)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDaemonSet(daemon.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos := f.DaemonSetRepos(&daemon)
			Expect(repos).NotTo(BeEmpty())
			f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldBackupExistingDaemonSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating DaemonSet " + daemon.Name)
			_, err = f.CreateDaemonSet(daemon)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDaemonSet(daemon.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos := f.DaemonSetRepos(&daemon)
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

			By("Creating DaemonSet " + daemon.Name)
			_, err = f.CreateDaemonSet(daemon)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDaemonSet(daemon.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			By("Waiting to remove sidecar")
			f.EventuallyDaemonSet(daemon.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldStopBackupIfLabelChanged = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating DaemonSet " + daemon.Name)
			_, err = f.CreateDaemonSet(daemon)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDaemonSet(daemon.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Removing labels of DaemonSet " + daemon.Name)
			_, _, err = apps_util.PatchDaemonSet(f.KubeClient, &daemon, func(in *apps.DaemonSet) *apps.DaemonSet {
				in.Labels = map[string]string{
					"app": "unmatched",
				}
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar to be removed")
			f.EventuallyDaemonSet(daemon.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldStopBackupIfSelectorChanged = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating DaemonSet " + daemon.Name)
			_, err = f.CreateDaemonSet(daemon)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDaemonSet(daemon.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

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

			By("Waiting for sidecar to be removed")
			f.EventuallyDaemonSet(daemon.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldMutateAndBackupNewDaemonSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating DaemonSet " + daemon.Name)
			obj, err := f.CreateDaemonSet(daemon)
			Expect(err).NotTo(HaveOccurred())

			// sidecar should be added as soon as daemonset created, we don't need to wait for it
			By("Checking sidecar added")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos := f.DaemonSetRepos(&daemon)
			Expect(repos).NotTo(BeEmpty())
			f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldNotMutateNewDaemonSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating DaemonSet " + daemon.Name)
			obj, err := f.CreateDaemonSet(daemon)
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar not added")
			Expect(obj).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldRejectToCreateNewDaemonSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating first restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating second restic " + secondRestic.Name)
			err = f.CreateRestic(secondRestic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating DaemonSet " + daemon.Name)
			_, err := f.CreateDaemonSet(daemon)
			Expect(err).To(HaveOccurred())
		}

		shouldRemoveSidecarInstantly = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating DaemonSet " + daemon.Name)
			obj, err := f.CreateDaemonSet(daemon)
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar added")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Removing labels of DaemonSet " + daemon.Name)
			obj, _, err = apps_util.PatchDaemonSet(f.KubeClient, &daemon, func(in *apps.DaemonSet) *apps.DaemonSet {
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

			By("Creating DaemonSet " + daemon.Name)
			previousLabel := daemon.Labels
			daemon.Labels = map[string]string{
				"app": "unmatched",
			}
			obj, err := f.CreateDaemonSet(daemon)
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar not added")
			Expect(obj).ShouldNot(HaveSidecar(util.StashContainer))

			By("Adding label to match restic" + daemon.Name)
			obj, _, err = apps_util.PatchDaemonSet(f.KubeClient, &daemon, func(in *apps.DaemonSet) *apps.DaemonSet {
				in.Labels = previousLabel
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar added")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
		}
	)

	Describe("Creating restic for", func() {
		AfterEach(func() {
			f.DeleteDaemonSet(daemon.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should backup new DaemonSet`, shouldBackupNewDaemonSet)
			It(`should backup existing DaemonSet`, shouldBackupExistingDaemonSet)
		})

		Context(`"S3" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForS3Backend()
				restic = f.ResticForS3Backend()
			})
			It(`should backup new DaemonSet`, shouldBackupNewDaemonSet)
			It(`should backup existing DaemonSet`, shouldBackupExistingDaemonSet)
		})

		Context(`"DO" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForDOBackend()
				restic = f.ResticForDOBackend()
			})
			It(`should backup new DaemonSet`, shouldBackupNewDaemonSet)
			It(`should backup existing DaemonSet`, shouldBackupExistingDaemonSet)
		})

		Context(`"GCS" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForGCSBackend()
				restic = f.ResticForGCSBackend()
			})
			It(`should backup new DaemonSet`, shouldBackupNewDaemonSet)
			It(`should backup existing DaemonSet`, shouldBackupExistingDaemonSet)
		})

		Context(`"Azure" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForAzureBackend()
				restic = f.ResticForAzureBackend()
			})
			It(`should backup new DaemonSet`, shouldBackupNewDaemonSet)
			It(`should backup existing DaemonSet`, shouldBackupExistingDaemonSet)
		})

		Context(`"Swift" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForSwiftBackend()
				restic = f.ResticForSwiftBackend()
			})
			It(`should backup new DaemonSet`, shouldBackupNewDaemonSet)
			It(`should backup existing DaemonSet`, shouldBackupExistingDaemonSet)
		})

		Context(`"B2" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForB2Backend()
				restic = f.ResticForB2Backend()
			})
			It(`should backup new DaemonSet`, shouldBackupNewDaemonSet)
			It(`should backup existing DaemonSet`, shouldBackupExistingDaemonSet)
		})
	})

	Describe("Changing DaemonSet labels", func() {
		AfterEach(func() {
			f.DeleteDaemonSet(daemon.ObjectMeta)
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
			f.DeleteDaemonSet(daemon.ObjectMeta)
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
			f.DeleteDaemonSet(daemon.ObjectMeta)
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

	Describe("Stash Webhook for", func() {
		AfterEach(func() {
			f.DeleteDaemonSet(daemon.ObjectMeta)
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
			It("should mutate and backup new DaemonSet", shouldMutateAndBackupNewDaemonSet)
			It("should not mutate new DaemonSet if no restic select it", shouldNotMutateNewDaemonSet)
			It("should reject to create new DaemonSet if multiple restic select it", shouldRejectToCreateNewDaemonSet)
			It("should remove sidecar instantly if label change to match no restic", shouldRemoveSidecarInstantly)
			It("should add sidecar instantly if label change to match single restic", shouldAddSidecarInstantly)
		})
	})

	Describe("Offline backup for", func() {
		AfterEach(func() {
			f.DeleteDaemonSet(daemon.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.Type = api.BackupOffline
				restic.Spec.Schedule = "@every 3m"
			})
			It(`should backup new DaemonSet`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				cronJobName := util.ScaledownCronPrefix + restic.Name
				By("Checking cron job created: " + cronJobName)
				Eventually(func() error {
					_, err := f.KubeClient.BatchV1beta1().CronJobs(restic.Namespace).Get(cronJobName, metav1.GetOptions{})
					return err
				}).Should(BeNil())

				By("Creating DaemonSet " + daemon.Name)
				_, err = f.CreateDaemonSet(daemon)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for init-container")
				f.EventuallyDaemonSet(daemon.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

				By("Waiting for initial backup to complete")
				f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically("==", 1)))

				By("Ensuring initial backup is not taken by cronJob")
				backupCron, err := f.KubeClient.BatchV1beta1().CronJobs(restic.Namespace).Get(cronJobName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(backupCron.Status.LastScheduleTime).Should(BeNil())

				By("Waiting for 3 backup by cronJob to complete")
				start := time.Now()
				for i := 2; i <= 4; i++ {
					fmt.Printf("=============== Waiting for backup no: %d ============\n", i)
					f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically("==", i)))
				}
				elapsedTime := time.Since(start).Minutes()

				// backup is scheduled for every 3 minutes.
				// so 3 backup by cronJob should not take more than 9 minutes + some overhead.(let 1 minute overhead for each backup)
				Expect(elapsedTime).Should(BeNumerically("<=", 9+3))

				By("Waiting for backup event")
				repos := f.DaemonSetRepos(&daemon)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">", 1)))
			})
		})
	})

	Describe("Pause Restic to stop backup", func() {
		Context(`"Local" backend`, func() {
			AfterEach(func() {
				f.DeleteDaemonSet(daemon.ObjectMeta)
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

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating Daemonset " + daemon.Name)
				_, err = f.CreateDaemonSet(daemon)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDaemonSet(daemon.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.DaemonSetRepos(&daemon)
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
				repos = f.DaemonSetRepos(&daemon)
				Expect(repos).NotTo(BeEmpty())

				previousBackupCount := repos[0].Status.BackupCount

				By("Waiting 2 minutes")
				time.Sleep(2 * time.Minute)

				By("Checking that Backup count has not changed")
				repos = f.DaemonSetRepos(&daemon)
				Expect(repos).NotTo(BeEmpty())
				Expect(repos[0].Status.BackupCount).Should(BeNumerically("==", previousBackupCount))

				By(`Patching Restic with "paused: false"`)
				err = f.CreateOrPatchRestic(restic.ObjectMeta, func(in *api.Restic) *api.Restic {
					in.Spec.Paused = false
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">", previousBackupCount)))

				By("Waiting for backup event")
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">", previousBackupCount)))

			})

		})
	})

	Describe("Repository CRD", func() {
		Context(`"Local" backend`, func() {
			AfterEach(func() {
				f.DeleteDaemonSet(daemon.ObjectMeta)
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

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating Daemonset " + daemon.Name)
				_, err = f.CreateDaemonSet(daemon)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDaemonSet(daemon.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">", 1)))

				By("Waiting for backup event")
				repos := f.DaemonSetRepos(&daemon)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

			})

		})
	})

	Describe("Complete Recovery", func() {
		Context(`"Local" backend, single fileGroup`, func() {
			AfterEach(func() {
				f.CleanupRecoveredVolume(daemon.ObjectMeta)
				f.DeleteDaemonSet(daemon.ObjectMeta)
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

				By("Creating DaemonSet " + daemon.Name)
				_, err = f.CreateDaemonSet(daemon)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDaemonSet(daemon.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.DaemonSetRepos(&daemon)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(daemon.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic))
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting daemon")
				f.DeleteDaemonSet(daemon.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// wait until daemonset terminated
				err = framework.WaitUntilDaemonSetDeleted(f.KubeClient, daemon.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, daemon.Namespace, f.AppLabel())
				Expect(err).NotTo(HaveOccurred())

				nodeName := f.GetNodeName(daemon.ObjectMeta)
				recovery.Spec.Repository.Name = localRef.GetRepositoryCRDName("", nodeName)
				recovery.Spec.Repository.Namespace = f.Namespace()

				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying daemon with recovered volume")
				daemon.Spec.Template.Spec.Volumes = f.RecoveredVolume()
				_, err = f.CreateDaemonSet(daemon)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(daemon.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic)).Should(BeEquivalentTo(previousData))
			})
		})

		Context(`"Local" backend, multiple fileGroup`, func() {
			AfterEach(func() {
				f.CleanupRecoveredVolume(daemon.ObjectMeta)
				f.DeleteDaemonSet(daemon.ObjectMeta)
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

				daemon.Spec.Template.Spec.Volumes = f.HostPathVolumeWithMultipleDirectory()
				By("Creating DaemonSet " + daemon.Name)
				_, err = f.CreateDaemonSet(daemon)
				Expect(err).NotTo(HaveOccurred())
				f.WaitUntilDaemonPodReady(daemon.ObjectMeta)

				By("Creating demo data in hostPath")
				err = f.CreateDemoData(daemon.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDaemonSet(daemon.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.DaemonSetRepos(&daemon)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(daemon.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic))
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting daemon")
				f.DeleteDaemonSet(daemon.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// wait until daemonset terminated
				err = framework.WaitUntilDaemonSetDeleted(f.KubeClient, daemon.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, daemon.Namespace, f.AppLabel())
				Expect(err).NotTo(HaveOccurred())

				nodeName := f.GetNodeName(daemon.ObjectMeta)
				recovery.Spec.Repository.Name = localRef.GetRepositoryCRDName("", nodeName)
				recovery.Spec.Repository.Namespace = f.Namespace()

				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying daemon with recovered volume")
				daemon.Spec.Template.Spec.Volumes = f.RecoveredVolume()
				_, err = f.CreateDaemonSet(daemon)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(daemon.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic)).Should(BeEquivalentTo(previousData))
			})
		})
	})

	Describe("Recover from different namespace", func() {
		var (
			recoveryNamespace *core.Namespace
		)
		Context(`"Local" backend, single fileGroup`, func() {
			AfterEach(func() {
				f.CleanupRecoveredVolume(daemon.ObjectMeta)
				f.DeleteDaemonSet(daemon.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
				f.DeleteRecovery(recovery.ObjectMeta)
				f.DeleteNamespace(recoveryNamespace.Name)

				err := framework.WaitUntilRecoveryDeleted(f.StashClient, recovery.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

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

				By("Creating DaemonSet " + daemon.Name)
				_, err = f.CreateDaemonSet(daemon)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDaemonSet(daemon.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.DaemonSetRepos(&daemon)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(daemon.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic))
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting daemon")
				f.DeleteDaemonSet(daemon.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// wait until daemonset terminated
				err = framework.WaitUntilDaemonSetDeleted(f.KubeClient, daemon.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, daemon.Namespace, f.AppLabel())
				Expect(err).NotTo(HaveOccurred())

				nodeName := f.GetNodeName(daemon.ObjectMeta)
				recovery.Spec.Repository.Name = localRef.GetRepositoryCRDName("", nodeName)
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

				By("Re-deploying daemon with recovered volume")
				daemon.Namespace = recoveryNamespace.Name
				daemon.Spec.Template.Spec.Volumes = f.RecoveredVolume()
				_, err = f.CreateDaemonSet(daemon)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(daemon.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic)).Should(BeEquivalentTo(previousData))
			})
		})
	})
})
