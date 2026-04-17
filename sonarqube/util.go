package sonarqube

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
)

// portfolioRefresh triggers a SonarQube portfolio rebuild via api/views/refresh.
func portfolioRefresh(portfolioKey string, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/refresh"
	sonarQubeURL.RawQuery = url.Values{
		"key": []string{portfolioKey},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"portfolioRefresh",
	)
	if err != nil {
		return fmt.Errorf("portfolioRefresh: failed to refresh portfolio %q: %+v", portfolioKey, err)
	}
	defer resp.Body.Close()
	return nil
}

// Checks if two string slices are equal, optionally ignoring ordering
func stringSlicesEqual(a, b []string, ignoreOrder bool) bool {
	if ignoreOrder {
		sort.Slice(a, func(i, j int) bool {
			return a[i] < a[j]
		})
		sort.Slice(b, func(i, j int) bool {
			return b[i] < b[j]
		})
	}

	return reflect.DeepEqual(a, b)
}
