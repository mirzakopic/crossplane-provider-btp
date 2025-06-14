package adapters

import (
	"fmt"
	"os"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/sap/crossplane-provider-btp/internal/crossplaneimport/client"
	cpimportconfig "github.com/sap/crossplane-provider-btp/internal/crossplaneimport/config"
	"github.com/sap/crossplane-provider-btp/internal/crossplaneimport/resource"
	"gopkg.in/yaml.v2"
)

// types for the config file
type Config struct {
	Resources         []Resource               `yaml:"resources"`
	ProviderConfigRef client.ProviderConfigRef `yaml:"providerConfigRef"`
}

type Resource struct {
	Subaccount  Subaccount  `yaml:"subaccount"`
	Entitlement Entitlement `yaml:"entitlement"`
	// add more resources here
}

type Subaccount struct {
	DisplayName        string              `yaml:"displayName"`
	Subdomain          string              `yaml:"subdomain"`
	Region             string              `yaml:"region"`
	Description        string              `yaml:"description"`
	BetaEnabled        *bool               `yaml:"betaEnabled,omitempty"`
	Labels             map[string][]string `yaml:"labels,omitempty"`
	SubaccountAdmins   []string            `yaml:"subaccountAdmins,omitempty"`
	UsedForProduction  string              `yaml:"usedForProduction,omitempty"`
	GlobalAccountGuid  string              `yaml:"globalAccountGuid,omitempty"`
	DirectoryGuid      string              `yaml:"directoryGuid,omitempty"`
	ManagementPolicies []ManagementPolicy  `yaml:"managementPolicies"`
}

type Entitlement struct {
	ServiceName                 string             `yaml:"serviceName"`
	ServicePlanName             string             `yaml:"servicePlanName"`
	ServicePlanUniqueIdentifier string             `yaml:"servicePlanUniqueIdentifier,omitempty"`
	SubaccountGuid              string             `yaml:"subaccountGuid"`
	Enable                      *bool              `yaml:"enable,omitempty"`
	Amount                      *int               `yaml:"amount,omitempty"`
	ManagementPolicies          []ManagementPolicy `yaml:"managementPolicies"`
}

type ManagementPolicy string

// BTPResourceFilter implements the ResourceFilter interface
type BTPResourceFilter struct {
	Type               string
	Subaccount         *SubaccountFilter
	Entitlement        *EntitlementFilter
	ManagementPolicies []v1.ManagementAction
}

func (f *BTPResourceFilter) GetResourceType() string {
	return f.Type
}

func (f *BTPResourceFilter) GetFilterCriteria() map[string]string {
	criteria := make(map[string]string)

	if f.Subaccount != nil {
		if f.Subaccount.DisplayName != "" {
			criteria["displayName"] = f.Subaccount.DisplayName
		}
		if f.Subaccount.Subdomain != nil && *f.Subaccount.Subdomain != "" {
			criteria["subdomain"] = *f.Subaccount.Subdomain
		}
		if f.Subaccount.Region != nil && *f.Subaccount.Region != "" {
			criteria["region"] = *f.Subaccount.Region
		}
		if f.Subaccount.Description != nil && *f.Subaccount.Description != "" {
			criteria["description"] = *f.Subaccount.Description
		}
	}

	if f.Entitlement != nil {
		if f.Entitlement.ServiceName != "" {
			criteria["serviceName"] = f.Entitlement.ServiceName
		}
		if f.Entitlement.ServicePlanName != "" {
			criteria["servicePlanName"] = f.Entitlement.ServicePlanName
		}
		if f.Entitlement.SubaccountGuid != "" {
			criteria["subaccountGuid"] = f.Entitlement.SubaccountGuid
		}
	}

	return criteria
}

func (f *BTPResourceFilter) GetManagementPolicies() []v1.ManagementAction {
	return f.ManagementPolicies
}

type SubaccountFilter struct {
	DisplayName       string
	Subdomain         *string
	Region            *string
	Description       *string
	BetaEnabled       *bool
	Labels            map[string][]string
	SubaccountAdmins  []string
	UsedForProduction *string
	GlobalAccountGuid *string
	DirectoryGuid     *string
}

type EntitlementFilter struct {
	ServiceName                 string
	ServicePlanName             string
	ServicePlanUniqueIdentifier string
	SubaccountGuid              string
	Enable                      *bool
	Amount                      *int
}

// BTPConfig implements the ProviderConfig interface
type BTPConfig struct {
	Resources         []Resource
	ProviderConfigRef client.ProviderConfigRef
}

func (c *BTPConfig) GetProviderConfigRef() client.ProviderConfigRef {
	return c.ProviderConfigRef
}

func (c *BTPConfig) Validate() bool {
	for _, resource := range c.Resources {
		// check for valid subaccount configuration - DisplayName is the primary identifier
		if resource.Subaccount.DisplayName != "" && resource.Subaccount.ManagementPolicies == nil {
			fmt.Println("Subaccount configuration is missing management policies")
			return false
		}
		// check for valid entitlement configuration - all three fields are required
		if (resource.Entitlement.ServiceName != "" || resource.Entitlement.ServicePlanName != "" ||
			resource.Entitlement.SubaccountGuid != "") &&
			resource.Entitlement.ManagementPolicies == nil {
			fmt.Println("Entitlement configuration is missing management policies")
			return false
		}
	}

	// check for empty provider config ref
	if c.ProviderConfigRef.Name == "" {
		return false
	}

	return true
}

// Helper function to convert string to *string if not empty
func stringToPointer(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// BTPConfigParser implements the ConfigParser interface
//
// DEPRECATED: This parser uses the old configuration format. For new implementations,
// use the BTPConfigParser in internal/crossplaneimport/config/parser.go which supports
// the new ImportConfig structure and dynamic filter validation.
type BTPConfigParser struct{}

func (p *BTPConfigParser) ParseConfig(configPath string) (cpimportconfig.ProviderConfig, []resource.ResourceFilter, error) {
	// Read config file
	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil, nil, err
	}

	// Parse YAML
	var config Config
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, nil, err
	}

	// Convert to BTPConfig
	btpConfig := &BTPConfig{
		Resources: config.Resources,
		ProviderConfigRef: client.ProviderConfigRef{
			Name: config.ProviderConfigRef.Name,
		},
	}

	// Convert to resource filters
	var filters []resource.ResourceFilter
	for _, res := range config.Resources {
		// For Subaccounts, DisplayName is the primary identifier
		if res.Subaccount.DisplayName != "" {
			var policies []v1.ManagementAction
			for _, policy := range res.Subaccount.ManagementPolicies {
				policies = append(policies, v1.ManagementAction(policy))
			}

			filters = append(filters, &BTPResourceFilter{
				Type: "Subaccount",
				Subaccount: &SubaccountFilter{
					DisplayName:       res.Subaccount.DisplayName,
					Subdomain:         stringToPointer(res.Subaccount.Subdomain),
					Region:            stringToPointer(res.Subaccount.Region),
					Description:       stringToPointer(res.Subaccount.Description),
					BetaEnabled:       res.Subaccount.BetaEnabled,
					Labels:            res.Subaccount.Labels,
					SubaccountAdmins:  res.Subaccount.SubaccountAdmins,
					UsedForProduction: stringToPointer(res.Subaccount.UsedForProduction),
					GlobalAccountGuid: stringToPointer(res.Subaccount.GlobalAccountGuid),
					DirectoryGuid:     stringToPointer(res.Subaccount.DirectoryGuid),
				},
				ManagementPolicies: policies,
			})
		}

		// For Entitlements, all three fields remain required as per requirements
		if res.Entitlement.ServiceName != "" || res.Entitlement.ServicePlanName != "" ||
			res.Entitlement.SubaccountGuid != "" {
			var policies []v1.ManagementAction
			for _, policy := range res.Entitlement.ManagementPolicies {
				policies = append(policies, v1.ManagementAction(policy))
			}

			filters = append(filters, &BTPResourceFilter{
				Type: "Entitlement",
				Entitlement: &EntitlementFilter{
					ServiceName:                 res.Entitlement.ServiceName,
					ServicePlanName:             res.Entitlement.ServicePlanName,
					ServicePlanUniqueIdentifier: res.Entitlement.ServicePlanUniqueIdentifier,
					SubaccountGuid:              res.Entitlement.SubaccountGuid,
					Enable:                      res.Entitlement.Enable,
					Amount:                      res.Entitlement.Amount,
				},
				ManagementPolicies: policies,
			})
		}
	}

	return btpConfig, filters, nil
}

// LoadAndValidateNewConfig is a convenience function that uses the new BTPConfigParser
// from the config package to load and validate configuration with dynamic filter validation.
//
// This function demonstrates how to migrate from the old parser to the new one.
//
// Parameters:
//   - configPath: Path to the configuration file
//
// Returns:
//   - *cpimportconfig.ImportConfig: The parsed and validated configuration using the new format
//   - error: Any parsing or validation errors
//
// Example usage:
//
//	import (
//	    "github.com/sap/crossplane-provider-btp/internal/crossplaneimport/config"
//	    "github.com/sap/crossplane-provider-btp/internal/crossplaneimport/importer"
//	)
//
//	registry := importer.NewRegistryAdapter()
//	config, err := adapters.LoadAndValidateNewConfig("config.yaml", registry)
//	if err != nil {
//	    log.Fatalf("Configuration validation failed: %v", err)
//	}
func LoadAndValidateNewConfig(configPath string, registry cpimportconfig.AdapterRegistry) (*cpimportconfig.ImportConfig, error) {
	parser := cpimportconfig.NewBTPConfigParser(registry)
	return parser.LoadAndValidateCLIConfig(configPath)
}
