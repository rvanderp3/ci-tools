package prowgen

import (
	"sort"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	prowconfig "k8s.io/test-infra/prow/config"

	ciop "github.com/openshift/ci-tools/pkg/api"
	"github.com/openshift/ci-tools/pkg/config"
	"github.com/openshift/ci-tools/pkg/testhelper"
)

func sorted(spec *corev1.PodSpec) *corev1.PodSpec {
	container := &spec.Containers[0]
	sort.Slice(spec.Volumes, func(i, j int) bool {
		return spec.Volumes[i].Name < spec.Volumes[j].Name
	})
	sort.Slice(container.Env, func(i, j int) bool {
		return container.Env[i].Name < container.Env[j].Name
	})
	sort.Slice(container.VolumeMounts, func(i, j int) bool {
		return container.VolumeMounts[i].Name < container.VolumeMounts[j].Name
	})

	canSortArgs := true
	for i := range container.Args {
		if !strings.HasPrefix(container.Args[i], "--") {
			canSortArgs = false
			break
		}
	}
	if canSortArgs {
		sort.Strings(container.Args)
	}
	return spec
}
func TestGeneratePresubmitForTest(t *testing.T) {
	tests := []struct {
		description string

		test              string
		repoInfo          *ProwgenInfo
		jobRelease        string
		clone             bool
		runIfChanged      string
		skipIfOnlyChanged string
		optional          bool
	}{{
		description: "presubmit for standard test",
		test:        "testname",
		repoInfo:    &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"}},
	},
		{
			description: "presubmit for a test in a variant config",
			test:        "testname",
			repoInfo:    &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch", Variant: "also"}},
		},
		{
			description: "presubmit with job release specified",
			test:        "testname",
			repoInfo:    &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"}},
			jobRelease:  "4.6",
		},
		{
			description: "presubmit with job release specified and clone",
			test:        "testname",
			repoInfo:    &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"}},
			jobRelease:  "4.6",
			clone:       true,
		},
		{
			description:  "presubmit with run_if_changed",
			test:         "testname",
			repoInfo:     &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"}},
			jobRelease:   "4.6",
			runIfChanged: "^README.md$",
		},
		{
			description:       "presubmit with skip_if_only_changed",
			test:              "testname",
			repoInfo:          &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"}},
			jobRelease:        "4.6",
			skipIfOnlyChanged: "^README.md$",
		},
		{
			description: "optional presubmit",
			test:        "testname",
			repoInfo:    &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"}},
			jobRelease:  "4.6",
			optional:    true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			// podSpec tested in generatePodSpec
			testhelper.CompareWithFixture(t, generatePresubmitForTest(tc.test, tc.repoInfo, nil, nil, tc.jobRelease, !tc.clone, tc.runIfChanged, tc.skipIfOnlyChanged, tc.optional, nil, false))
		})
	}
}

func TestGeneratePeriodicForTest(t *testing.T) {
	tests := []struct {
		description string

		test              string
		repoInfo          *ProwgenInfo
		jobRelease        string
		clone             bool
		cron              string
		interval          string
		releaseController bool
	}{
		{
			description: "periodic for standard test",
			test:        "testname",
			repoInfo:    &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"}},
			cron:        "@yearly",
		},
		{
			description: "periodic for a test in a variant config",
			test:        "testname",
			repoInfo:    &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch", Variant: "also"}},
			cron:        "@yearly",
		},
		{
			description: "periodic for specific release",
			test:        "testname",
			repoInfo:    &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"}},
			jobRelease:  "4.6",
			cron:        "@yearly",
		},
		{
			description: "periodic for specific release using interval",
			test:        "testname",
			repoInfo:    &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"}},
			jobRelease:  "4.6",
			interval:    "6h",
		},
		{
			description: "periodic for specific release and clone: true",
			test:        "testname",
			repoInfo:    &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"}},
			jobRelease:  "4.6",
			clone:       true,
			cron:        "@yearly",
		},
		{
			description:       "release controller job",
			test:              "testname",
			repoInfo:          &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"}},
			jobRelease:        "4.6",
			releaseController: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			// podSpec tested in generatePodSpec
			testhelper.CompareWithFixture(t, generatePeriodicForTest(tc.test, tc.repoInfo, nil, true, tc.cron, tc.interval, tc.releaseController, nil, tc.jobRelease, !tc.clone, nil, false))
		})
	}
}

func TestGeneratePostSubmitForTest(t *testing.T) {
	tests := []struct {
		name       string
		repoInfo   *ProwgenInfo
		jobRelease string
		clone      bool
	}{
		{
			name: "Lowercase org repo and branch",
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "organization",
				Repo:   "repository",
				Branch: "branch",
			}},
		},
		{
			name: "Uppercase org, repo and branch",
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "Organization",
				Repo:   "Repository",
				Branch: "Branch",
			}},
		},
		{
			name: "Uppercase org, repo and branch with clone: true",
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "Organization",
				Repo:   "Repository",
				Branch: "Branch",
			}},
			clone: true,
		},
		{
			name: "Lowercase org repo and branch with release",
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "organization",
				Repo:   "repository",
				Branch: "branch",
			}},
			jobRelease: "4.6",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// podSpec tested in generatePodSpec
			testhelper.CompareWithFixture(t, generatePostsubmitForTest(tc.name, tc.repoInfo, nil, nil, tc.jobRelease, !tc.clone, nil, false))
		})
	}
}

var (
	cron = "0 0 * * *"
)

func TestGenerateJobs(t *testing.T) {
	tests := []struct {
		id       string
		keep     bool
		config   *ciop.ReleaseBuildConfiguration
		repoInfo *ProwgenInfo
	}{
		{
			id: "two tests and empty Images so only two test presubmits are generated",
			config: &ciop.ReleaseBuildConfiguration{
				Tests: []ciop.TestStepConfiguration{
					{As: "derTest", ContainerTestConfiguration: &ciop.ContainerTestConfiguration{From: "from"}},
					{As: "leTest", ContainerTestConfiguration: &ciop.ContainerTestConfiguration{From: "from"}}},
			},
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "organization",
				Repo:   "repository",
				Branch: "branch",
			}},
		}, {
			id: "two tests and nonempty Images so two test presubmits and images pre/postsubmits are generated ",
			config: &ciop.ReleaseBuildConfiguration{
				Tests: []ciop.TestStepConfiguration{
					{As: "derTest", ContainerTestConfiguration: &ciop.ContainerTestConfiguration{From: "from"}},
					{As: "leTest", ContainerTestConfiguration: &ciop.ContainerTestConfiguration{From: "from"}}},
				Images:                 []ciop.ProjectDirectoryImageBuildStepConfiguration{{}},
				PromotionConfiguration: &ciop.PromotionConfiguration{},
			},
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "organization",
				Repo:   "repository",
				Branch: "branch",
			}},
		}, {
			id: "template test",
			config: &ciop.ReleaseBuildConfiguration{
				InputConfiguration: ciop.InputConfiguration{
					ReleaseTagConfiguration: &ciop.ReleaseTagConfiguration{Name: "origin-v4.0"}},
				Tests: []ciop.TestStepConfiguration{
					{
						As: "oTeste",
						OpenshiftAnsibleClusterTestConfiguration: &ciop.OpenshiftAnsibleClusterTestConfiguration{
							ClusterTestConfiguration: ciop.ClusterTestConfiguration{ClusterProfile: "gcp"},
						},
					},
				},
			},
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "organization",
				Repo:   "repository",
				Branch: "branch",
			}},
		}, {
			id: "Promotion configuration causes --promote job",
			config: &ciop.ReleaseBuildConfiguration{
				Tests:                  []ciop.TestStepConfiguration{},
				Images:                 []ciop.ProjectDirectoryImageBuildStepConfiguration{{}},
				PromotionConfiguration: &ciop.PromotionConfiguration{Namespace: "ci"},
			},
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "organization",
				Repo:   "repository",
				Branch: "branch",
			}},
		}, {
			id:   "Promotion configuration causes --promote job with unique targets",
			keep: true,
			config: &ciop.ReleaseBuildConfiguration{
				Tests: []ciop.TestStepConfiguration{},
				Images: []ciop.ProjectDirectoryImageBuildStepConfiguration{
					{To: "out-1", From: "base"},
					{To: "out-2", From: "base"},
				},
				PromotionConfiguration: &ciop.PromotionConfiguration{
					Namespace: "ci",
					AdditionalImages: map[string]string{
						"out": "out-1",
					},
				},
			},
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "organization",
				Repo:   "repository",
				Branch: "branch",
			}},
		}, {
			id: "no Promotion configuration has no branch job",
			config: &ciop.ReleaseBuildConfiguration{
				Tests:  []ciop.TestStepConfiguration{},
				Images: []ciop.ProjectDirectoryImageBuildStepConfiguration{{}},
				InputConfiguration: ciop.InputConfiguration{
					ReleaseTagConfiguration: &ciop.ReleaseTagConfiguration{Namespace: "openshift"},
				},
			},
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "organization",
				Repo:   "repository",
				Branch: "branch",
			}},
		}, {
			id: "operator section creates ci-index presubmit job",
			config: &ciop.ReleaseBuildConfiguration{
				Tests: []ciop.TestStepConfiguration{},
				Operator: &ciop.OperatorStepConfiguration{
					Bundles: []ciop.Bundle{{
						DockerfilePath: "bundle.Dockerfile",
						ContextDir:     "manifests",
					}},
				},
			},
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "organization",
				Repo:   "repository",
				Branch: "branch",
			}},
		}, {
			id: "two tests and empty Images with one test configured as a postsubmit",
			config: &ciop.ReleaseBuildConfiguration{
				Tests: []ciop.TestStepConfiguration{
					{As: "derTest", ContainerTestConfiguration: &ciop.ContainerTestConfiguration{From: "from"}},
					{As: "leTest", ContainerTestConfiguration: &ciop.ContainerTestConfiguration{From: "from"}, Postsubmit: true}},
			},
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "organization",
				Repo:   "repository",
				Branch: "branch",
			}},
		}, {
			id: "kvm label",
			config: &ciop.ReleaseBuildConfiguration{
				Resources: map[string]ciop.ResourceRequirements{
					"*": {Requests: ciop.ResourceList{"devices.kubevirt.io/kvm": "1"}},
				},
				Tests: []ciop.TestStepConfiguration{
					{As: "unit", ContainerTestConfiguration: &ciop.ContainerTestConfiguration{From: "bin"}},
				},
			},
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "organization",
				Repo:   "repository",
				Branch: "branch",
			}},
		},
		{
			id: "cluster label for presubmit",
			config: &ciop.ReleaseBuildConfiguration{
				Tests: []ciop.TestStepConfiguration{
					{As: "unit", Cluster: "build01", ContainerTestConfiguration: &ciop.ContainerTestConfiguration{From: "bin"}},
				},
			},
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "organization",
				Repo:   "repository",
				Branch: "branch",
			}},
		},
		{
			id: "cluster label for periodic",
			config: &ciop.ReleaseBuildConfiguration{
				Tests: []ciop.TestStepConfiguration{
					{As: "unit", Cron: &cron, Cluster: "build01", ContainerTestConfiguration: &ciop.ContainerTestConfiguration{From: "bin"}},
				},
			},
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "organization",
				Repo:   "repository",
				Branch: "branch",
			}},
		},
		{
			id: "cluster label for postsubmit",
			config: &ciop.ReleaseBuildConfiguration{
				Tests: []ciop.TestStepConfiguration{
					{As: "unit", Postsubmit: true, Cluster: "build01", ContainerTestConfiguration: &ciop.ContainerTestConfiguration{From: "bin"}},
				},
			},
			repoInfo: &ProwgenInfo{Metadata: ciop.Metadata{
				Org:    "organization",
				Repo:   "repository",
				Branch: "branch",
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			jobConfig := GenerateJobs(tc.config, tc.repoInfo)
			if !tc.keep {
				pruneForTests(jobConfig) // prune the fields that are tested in TestGeneratePre/PostsubmitForTest
			}
			testhelper.CompareWithFixture(t, sortPodspecsInJobsonfig(jobConfig))
		})
	}
}

func sortPodspecsInJobsonfig(jobConfig *prowconfig.JobConfig) *prowconfig.JobConfig {
	for repo := range jobConfig.PresubmitsStatic {
		for i := range jobConfig.PresubmitsStatic[repo] {
			if jobConfig.PresubmitsStatic[repo][i].Spec != nil {
				sorted(jobConfig.PresubmitsStatic[repo][i].Spec)
			}
		}
	}
	for repo := range jobConfig.PostsubmitsStatic {
		for i := range jobConfig.PostsubmitsStatic[repo] {
			if jobConfig.PostsubmitsStatic[repo][i].Spec != nil {
				sorted(jobConfig.PostsubmitsStatic[repo][i].Spec)
			}
		}
	}

	for i := range jobConfig.Periodics {
		if jobConfig.Periodics[i].Spec != nil {
			sorted(jobConfig.Periodics[i].Spec)
		}
	}

	return jobConfig
}

func TestGenerateJobBase(t *testing.T) {
	path := "/some/where"
	var testCases = []struct {
		testName    string
		name        string
		prefix      string
		info        *ProwgenInfo
		podSpec     *corev1.PodSpec
		rehearsable bool
		pathAlias   *string
		clone       bool
	}{
		{
			testName: "no special options",
			name:     "test",
			prefix:   "pull",
			info:     &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"}},
			podSpec:  &corev1.PodSpec{Containers: []corev1.Container{{Name: "test"}}},
		},
		{
			testName:    "rehearsable",
			name:        "test",
			prefix:      "pull",
			info:        &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"}},
			podSpec:     &corev1.PodSpec{Containers: []corev1.Container{{Name: "test"}}},
			rehearsable: true,
		},
		{
			testName: "config variant",
			name:     "test",
			prefix:   "pull",
			info:     &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch", Variant: "whatever"}},
			podSpec:  &corev1.PodSpec{Containers: []corev1.Container{{Name: "test"}}},
		},
		{
			testName:  "path alias",
			name:      "test",
			prefix:    "pull",
			info:      &ProwgenInfo{Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch", Variant: "whatever"}},
			podSpec:   &corev1.PodSpec{Containers: []corev1.Container{{Name: "test"}}},
			pathAlias: &path,
		},
		{
			testName: "hidden job for private repos",
			name:     "test",
			prefix:   "pull",
			info: &ProwgenInfo{
				Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"},
				Config:   config.Prowgen{Private: true},
			},
			podSpec: &corev1.PodSpec{Containers: []corev1.Container{{Name: "test"}}},
		},
		{
			testName: "expose job for private repos with public results",
			name:     "test",
			prefix:   "pull",
			info: &ProwgenInfo{
				Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"},
				Config:   config.Prowgen{Private: true, Expose: true},
			},
			podSpec: &corev1.PodSpec{Containers: []corev1.Container{{Name: "test"}}},
		},
		{
			testName: "expose option set but not private",
			name:     "test",
			prefix:   "pull",
			info: &ProwgenInfo{
				Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"},
				Config:   config.Prowgen{Private: false, Expose: true},
			},
			podSpec: &corev1.PodSpec{Containers: []corev1.Container{{Name: "test"}}},
		},
		{
			testName: "expose option set but not private with clone: true",
			name:     "test",
			prefix:   "pull",
			info: &ProwgenInfo{
				Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"},
				Config:   config.Prowgen{Private: false, Expose: true},
			},
			podSpec: &corev1.PodSpec{Containers: []corev1.Container{{Name: "test"}}},
			clone:   true,
		},
		{
			testName: "private jobs that clone should contain oauth_token_secret config",
			name:     "test",
			prefix:   "pull",
			info: &ProwgenInfo{
				Metadata: ciop.Metadata{Org: "org", Repo: "repo", Branch: "branch"},
				Config:   config.Prowgen{Private: true, Expose: true},
			},
			podSpec: &corev1.PodSpec{Containers: []corev1.Container{{Name: "test"}}},
			clone:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			testhelper.CompareWithFixture(t, generateJobBase(testCase.name, testCase.prefix, testCase.info, sorted(testCase.podSpec), testCase.rehearsable, testCase.pathAlias, "", !testCase.clone, nil, false))
		})
	}
}

func pruneForTests(jobConfig *prowconfig.JobConfig) {
	for repo := range jobConfig.PresubmitsStatic {
		for i := range jobConfig.PresubmitsStatic[repo] {
			jobConfig.PresubmitsStatic[repo][i].AlwaysRun = false
			jobConfig.PresubmitsStatic[repo][i].Context = ""
			jobConfig.PresubmitsStatic[repo][i].Trigger = ""
			jobConfig.PresubmitsStatic[repo][i].RerunCommand = ""
			jobConfig.PresubmitsStatic[repo][i].Agent = ""
			jobConfig.PresubmitsStatic[repo][i].Spec = nil
			jobConfig.PresubmitsStatic[repo][i].Brancher = prowconfig.Brancher{}
			jobConfig.PresubmitsStatic[repo][i].UtilityConfig = prowconfig.UtilityConfig{}
		}
	}
	for repo := range jobConfig.PostsubmitsStatic {
		for i := range jobConfig.PostsubmitsStatic[repo] {
			jobConfig.PostsubmitsStatic[repo][i].Agent = ""
			jobConfig.PostsubmitsStatic[repo][i].Spec = nil
			jobConfig.PostsubmitsStatic[repo][i].Brancher = prowconfig.Brancher{}
			jobConfig.PostsubmitsStatic[repo][i].UtilityConfig = prowconfig.UtilityConfig{}
		}
	}
}
