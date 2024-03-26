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

func getHelmReleaseByShortName(shortName string, helmClient helm.Client) (release.Release, bool) {
	releases, err := helmClient.ListDeployedReleases()
	if err != nil {
		log.Fatalf("Error: Could not List deployed helm releases : %v", err)
		return release.Release{}, false
	}
	if len(releases) == 0 {
		log.Fatal("Error: no deployed Helm releases found in source cluster.")
		return release.Release{}, false
	}
	filteredReleases := []release.Release{}
	for _, release := range releases {
		fmt.Printf("found release: %s\n", release.Name)
		if strings.Contains(release.Chart.Name(), shortName) {
			filteredReleases = append(filteredReleases, *release)
		}
	}
	if len(filteredReleases) == 0 {
		return release.Release{}, false
	}
	if len(filteredReleases) > 1 {
		log.Println("Warning: found multiple helm releases in cluster with \"velero\" in name :")
		for _, release := range filteredReleases {
			log.Println(release.Name)
		}
		return release.Release{}, false
	}
	return filteredReleases[0], true
}

func cloneVeleroHelmChart(destinationHelmClient helm.Client, destinationHelmValues map[string]interface{}, sourceVeleroRelease release.Release, destinationReleaseNamespace string) {
	chartRepo := repo.Entry{
		Name:                  "velero",
		URL:                   "https://vmware-tanzu.github.io/helm-charts",
		InsecureSkipTLSverify: true,
	}
	// Add a chart-repository to the client.
	if err := destinationHelmClient.AddOrUpdateChartRepo(chartRepo); err != nil {
		log.Fatalf("Error: could not add %s chart Repo to helm client %v ", chartRepo.URL, err)
	}
	destinationRealeaseValuesYAML, err := mapToYAML(destinationHelmValues)
	if err != nil {
		log.Fatalf("Error: could not Parse source Release YAML Values.")
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
	if _, err := destinationHelmClient.InstallOrUpgradeChart(context.Background(), &destinationChartSpec, &helm.GenericHelmOptions{}); err != nil {
		log.Fatalf("Error: could not install or update velero helm chart on destination kubernetes cluster %v", err)
	}
}
