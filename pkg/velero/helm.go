package velero

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	helm "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
)

// getHelmReleaseByShortName retrieves a Helm release by its short name.
// It returns the release and a boolean indicating if the release is found.
func getHelmReleaseByShortName(shortName string, helmClient helm.Client) (release.Release, bool) {
	// List deployed Helm releases
	releases, err := helmClient.ListDeployedReleases()
	if err != nil {
		log.Fatalf("Error: Could not list deployed Helm releases: %v", err)
		return release.Release{}, false
	}
	if len(releases) == 0 {
		log.Fatal("Error: No deployed Helm releases found in the source cluster.")
		return release.Release{}, false
	}

	// Filter releases by shortName
	filteredReleases := []release.Release{}
	for _, release := range releases {
		fmt.Printf("Found release: %s\n", release.Name)
		if strings.Contains(release.Chart.Name(), shortName) {
			filteredReleases = append(filteredReleases, *release)
		}
	}

	// Handle filtered releases
	if len(filteredReleases) == 0 {
		return release.Release{}, false
	}
	if len(filteredReleases) > 1 {
		log.Println("Warning: Found multiple Helm releases in the cluster with \"velero\" in name:")
		for _, release := range filteredReleases {
			log.Println(release.Name)
		}
		return release.Release{}, false
	}
	return filteredReleases[0], true
}

// cloneVeleroHelmChart clones the Velero Helm chart to the destination Kubernetes cluster.
func cloneVeleroHelmChart(destinationHelmClient helm.Client, destinationHelmValues map[string]interface{}, sourceVeleroRelease release.Release, destinationReleaseNamespace string) {
	// Define the chart repository
	chartRepo := repo.Entry{
		Name:                  "velero",
		URL:                   "https://vmware-tanzu.github.io/helm-charts",
		InsecureSkipTLSverify: true,
	}
	// Add or update the chart repository to the Helm client
	if err := destinationHelmClient.AddOrUpdateChartRepo(chartRepo); err != nil {
		log.Fatalf("Error: Could not add %s chart repository to the Helm client: %v", chartRepo.URL, err)
	}

	// Convert destination Helm values to YAML
	destinationRealeaseValuesYAML, err := mapToYAML(destinationHelmValues)
	if err != nil {
		log.Fatalf("Error: Could not parse source release YAML values.")
	}

	// Define the chart to be installed
	destinationChartSpec := helm.ChartSpec{
		GenerateName:    true,
		ChartName:       fmt.Sprintf("%s/%s", "velero", sourceVeleroRelease.Chart.Name()),
		Namespace:       destinationReleaseNamespace,
		CreateNamespace: true,
		Version:         sourceVeleroRelease.Chart.Metadata.Version,
		UpgradeCRDs:     true,
		Wait:            true,
		WaitForJobs:     true,
		Timeout:         15 * time.Minute,
		ValuesYaml:      destinationRealeaseValuesYAML,
	}

	// Install or upgrade the Helm chart on the destination Kubernetes cluster
	if _, err := destinationHelmClient.InstallOrUpgradeChart(context.Background(), &destinationChartSpec, &helm.GenericHelmOptions{}); err != nil {
		log.Fatalf("Error: Could not install or update Velero Helm chart on the destination Kubernetes cluster: %v", err)
	}
}
