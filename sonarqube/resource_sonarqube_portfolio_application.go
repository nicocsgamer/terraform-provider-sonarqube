package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Returns the resource represented by this file.
func resourceSonarqubePortfolioApplication() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a SonarQube Portfolio Application resource. Links an Application to a Portfolio via api/views/add_application.",
		Create:      resourceSonarqubePortfolioApplicationCreate,
		Read:        resourceSonarqubePortfolioApplicationRead,
		Delete:      resourceSonarqubePortfolioApplicationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"portfolio_key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The key of the Portfolio to link the Application to.",
			},
			"application_key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The key of the Application to link.",
			},
		},
	}
}

func resourceSonarqubePortfolioApplicationCreate(d *schema.ResourceData, m interface{}) error {
	if err := checkPortfolioSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	portfolioKey := d.Get("portfolio_key").(string)
	applicationKey := d.Get("application_key").(string)

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/add_application"
	sonarQubeURL.RawQuery = url.Values{
		"portfolio":   []string{portfolioKey},
		"application": []string{applicationKey},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubePortfolioApplicationCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	d.SetId(fmt.Sprintf("%s/%s", portfolioKey, applicationKey))

	if err := portfolioRefresh(portfolioKey, m); err != nil {
		return err
	}

	return resourceSonarqubePortfolioApplicationRead(d, m)
}

func resourceSonarqubePortfolioApplicationRead(d *schema.ResourceData, m interface{}) error {
	if err := checkPortfolioSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	parts := strings.SplitN(d.Id(), "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("resourceSonarqubePortfolioApplicationRead: invalid ID format %q, expected portfolio_key/application_key", d.Id())
	}
	portfolioKey := parts[0]
	applicationKey := parts[1]

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/show"
	sonarQubeURL.RawQuery = url.Values{
		"key": []string{portfolioKey},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
		http.StatusOK,
		"resourceSonarqubePortfolioApplicationRead",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	type portfolioShowResponse struct {
		Applications []struct {
			Key string `json:"key"`
		} `json:"applications"`
	}
	showResponse := portfolioShowResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&showResponse); err != nil {
		return fmt.Errorf("resourceSonarqubePortfolioApplicationRead: failed to decode response: %+v", err)
	}

	for _, app := range showResponse.Applications {
		if app.Key == applicationKey {
			if err := d.Set("portfolio_key", portfolioKey); err != nil {
				return err
			}
			return d.Set("application_key", applicationKey)
		}
	}

	// Application no longer linked — remove from state
	d.SetId("")
	return nil
}

func resourceSonarqubePortfolioApplicationDelete(d *schema.ResourceData, m interface{}) error {
	if err := checkPortfolioSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/remove_application"
	sonarQubeURL.RawQuery = url.Values{
		"portfolio":   []string{d.Get("portfolio_key").(string)},
		"application": []string{d.Get("application_key").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubePortfolioApplicationDelete",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
