package sonarqube

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// Application represents a SonarQube Application
type Application struct {
	Key        string `json:"key"`
	Name       string `json:"name"`
	Desc       string `json:"description"`
	Visibility string `json:"visibility"`
}

// ApplicationResponse wraps the API response for create/show
type ApplicationResponse struct {
	Application Application `json:"application"`
}

// Returns the resource represented by this file.
func resourceSonarqubeApplication() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a SonarQube Application resource. Applications are only available in Enterprise and Data Center editions.",
		Create:      resourceSonarqubeApplicationCreate,
		Read:        resourceSonarqubeApplicationRead,
		Update:      resourceSonarqubeApplicationUpdate,
		Delete:      resourceSonarqubeApplicationDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubeApplicationImport,
		},
		Schema: map[string]*schema.Schema{
			"key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The key of the Application to create.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    false,
				Description: "The name of the Application to create.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				ForceNew:    false,
				Description: "A description of the Application.",
			},
			"visibility": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "public",
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"public", "private"}, false),
				Description:  "Whether the Application should be visible to everyone, or only specific users/groups. Defaults to `public`.",
			},
		},
	}
}

func checkApplicationSupport(conf *ProviderConfiguration) error {
	edition := strings.ToLower(conf.sonarQubeEdition)
	if edition != "enterprise" && edition != "data center" {
		return fmt.Errorf("applications are only supported in the Enterprise and Datacenter editions of SonarQube. You are using: SonarQube %s version %s", conf.sonarQubeEdition, conf.sonarQubeVersion)
	}
	return nil
}

func resourceSonarqubeApplicationCreate(d *schema.ResourceData, m interface{}) error {
	if err := checkApplicationSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/applications/create"

	sonarQubeURL.RawQuery = url.Values{
		"key":        []string{d.Get("key").(string)},
		"name":       []string{d.Get("name").(string)},
		"visibility": []string{d.Get("visibility").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusOK,
		"resourceSonarqubeApplicationCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	applicationResponse := ApplicationResponse{}
	err = json.NewDecoder(resp.Body).Decode(&applicationResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeApplicationCreate: Failed to decode json into struct: %+v", err)
	}

	d.SetId(applicationResponse.Application.Key)

	// Set description if provided (separate API call on update endpoint)
	if desc := d.Get("description").(string); desc != "" {
		if err := applicationUpdate(d.Id(), d.Get("name").(string), desc, m); err != nil {
			return err
		}
	}

	return resourceSonarqubeApplicationRead(d, m)
}

func resourceSonarqubeApplicationRead(d *schema.ResourceData, m interface{}) error {
	if err := checkApplicationSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/applications/show"
	sonarQubeURL.RawQuery = url.Values{
		"application": []string{d.Id()},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
		http.StatusOK,
		"resourceSonarqubeApplicationRead",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	applicationResponse := ApplicationResponse{}
	err = json.NewDecoder(resp.Body).Decode(&applicationResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeApplicationRead: Failed to decode json into struct: %+v", err)
	}

	return updateResourceDataFromApplicationResponse(d, &applicationResponse.Application)
}

func resourceSonarqubeApplicationUpdate(d *schema.ResourceData, m interface{}) error {
	if err := checkApplicationSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	if d.HasChanges("name", "description") {
		if err := applicationUpdate(d.Id(), d.Get("name").(string), d.Get("description").(string), m); err != nil {
			return err
		}
	}

	return resourceSonarqubeApplicationRead(d, m)
}

func resourceSonarqubeApplicationDelete(d *schema.ResourceData, m interface{}) error {
	if err := checkApplicationSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/applications/delete"
	sonarQubeURL.RawQuery = url.Values{
		"application": []string{d.Id()},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubeApplicationDelete",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubeApplicationImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := resourceSonarqubeApplicationRead(d, m); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func applicationUpdate(key, name, description string, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/applications/update"
	sonarQubeURL.RawQuery = url.Values{
		"application": []string{key},
		"name":        []string{name},
		"description": []string{description},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"applicationUpdate",
	)
	if err != nil {
		return fmt.Errorf("applicationUpdate: Failed to update application: %+v", err)
	}
	defer resp.Body.Close()

	return nil
}

func updateResourceDataFromApplicationResponse(d *schema.ResourceData, app *Application) error {
	d.SetId(app.Key)
	errs := []error{
		d.Set("key", app.Key),
		d.Set("name", app.Name),
		d.Set("description", app.Desc),
		d.Set("visibility", app.Visibility),
	}
	return errors.Join(errs...)
}
