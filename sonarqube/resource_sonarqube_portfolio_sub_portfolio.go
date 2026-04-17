package sonarqube

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Returns the resource represented by this file.
func resourceSonarqubePortfolioSubPortfolio() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a SonarQube Portfolio Sub-Portfolio resource. Nests a Portfolio inside another Portfolio via api/views/add_local_view.",
		Create:      resourceSonarqubePortfolioSubPortfolioCreate,
		Read:        resourceSonarqubePortfolioSubPortfolioRead,
		Delete:      resourceSonarqubePortfolioSubPortfolioDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"parent_key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The key of the parent Portfolio (e.g. the DG portfolio).",
			},
			"child_key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The key of the Portfolio to nest as a sub-portfolio (e.g. the IS portfolio).",
			},
		},
	}
}

func resourceSonarqubePortfolioSubPortfolioCreate(d *schema.ResourceData, m interface{}) error {
	if err := checkPortfolioSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	parentKey := d.Get("parent_key").(string)
	childKey := d.Get("child_key").(string)

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/add_local_view"
	sonarQubeURL.RawQuery = url.Values{
		"key":     []string{parentKey},
		"ref_key": []string{childKey},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubePortfolioSubPortfolioCreate",
	)
	if err != nil && !strings.Contains(err.Error(), "already added") {
		return err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	d.SetId(fmt.Sprintf("%s/%s", parentKey, childKey))

	if err := portfolioRefresh(parentKey, m); err != nil {
		return err
	}

	return resourceSonarqubePortfolioSubPortfolioRead(d, m)
}

func resourceSonarqubePortfolioSubPortfolioRead(d *schema.ResourceData, m interface{}) error {
	if err := checkPortfolioSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	parts := strings.SplitN(d.Id(), "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("resourceSonarqubePortfolioSubPortfolioRead: invalid ID format %q, expected parent_key/child_key", d.Id())
	}
	parentKey := parts[0]
	childKey := parts[1]

	// Verify the parent portfolio still exists; if not, remove from state.
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/show"
	sonarQubeURL.RawQuery = url.Values{
		"key": []string{parentKey},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
		http.StatusOK,
		"resourceSonarqubePortfolioSubPortfolioRead",
	)
	if err != nil {
		d.SetId("")
		return nil
	}
	defer resp.Body.Close()

	if err := d.Set("parent_key", parentKey); err != nil {
		return err
	}
	return d.Set("child_key", childKey)
}

func resourceSonarqubePortfolioSubPortfolioDelete(d *schema.ResourceData, m interface{}) error {
	if err := checkPortfolioSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	parentKey := d.Get("parent_key").(string)
	childKey := d.Get("child_key").(string)

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/remove_local_view"
	sonarQubeURL.RawQuery = url.Values{
		"key":     []string{parentKey},
		"ref_key": []string{childKey},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubePortfolioSubPortfolioDelete",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return portfolioRefresh(parentKey, m)
}
