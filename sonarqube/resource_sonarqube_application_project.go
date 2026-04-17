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
func resourceSonarqubeApplicationProject() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a SonarQube Application Project resource. Links a Project to an Application via api/applications/add_project.",
		Create:      resourceSonarqubeApplicationProjectCreate,
		Read:        resourceSonarqubeApplicationProjectRead,
		Delete:      resourceSonarqubeApplicationProjectDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"application_key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The key of the Application to link the Project to.",
			},
			"project_key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The key of the Project to link.",
			},
		},
	}
}

func resourceSonarqubeApplicationProjectCreate(d *schema.ResourceData, m interface{}) error {
	if err := checkApplicationSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	applicationKey := d.Get("application_key").(string)
	projectKey := d.Get("project_key").(string)

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/applications/add_project"
	sonarQubeURL.RawQuery = url.Values{
		"application": []string{applicationKey},
		"project":     []string{projectKey},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubeApplicationProjectCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	d.SetId(fmt.Sprintf("%s/%s", applicationKey, projectKey))
	return resourceSonarqubeApplicationProjectRead(d, m)
}

func resourceSonarqubeApplicationProjectRead(d *schema.ResourceData, m interface{}) error {
	if err := checkApplicationSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	parts := strings.SplitN(d.Id(), "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("resourceSonarqubeApplicationProjectRead: invalid ID format %q, expected application_key/project_key", d.Id())
	}
	applicationKey := parts[0]
	projectKey := parts[1]

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/applications/show"
	sonarQubeURL.RawQuery = url.Values{
		"application": []string{applicationKey},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
		http.StatusOK,
		"resourceSonarqubeApplicationProjectRead",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	type applicationShowResponse struct {
		Application struct {
			Projects []struct {
				Key string `json:"key"`
			} `json:"projects"`
		} `json:"application"`
	}
	showResponse := applicationShowResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&showResponse); err != nil {
		return fmt.Errorf("resourceSonarqubeApplicationProjectRead: failed to decode response: %+v", err)
	}

	for _, project := range showResponse.Application.Projects {
		if project.Key == projectKey {
			if err := d.Set("application_key", applicationKey); err != nil {
				return err
			}
			return d.Set("project_key", projectKey)
		}
	}

	// Project no longer linked — remove from state
	d.SetId("")
	return nil
}

func resourceSonarqubeApplicationProjectDelete(d *schema.ResourceData, m interface{}) error {
	if err := checkApplicationSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/applications/remove_project"
	sonarQubeURL.RawQuery = url.Values{
		"application": []string{d.Get("application_key").(string)},
		"project":     []string{d.Get("project_key").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubeApplicationProjectDelete",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
