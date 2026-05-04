// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package userdata

import (
	_ "embed"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-playground/validator/v10"
	"github.com/podplane/podplane/internal/deps"
)

//go:embed user-data.sh
var userdataTemplate string

// TemplateVars are the template variables consumed by user-data.sh.
type TemplateVars struct {
	// Manifest is the resolved vmconfig manifest for this VM.
	Manifest *deps.Manifest `validate:"required"`

	// DepsMirrorURL, when set, overrides all upstream dependency URLs with
	// mirror-relative paths (<base>/<name>/<version>/<filename>). Used for
	// local development and air-gapped installs.
	DepsMirrorURL string `validate:"omitempty,url"`

	// Env is rendered one-to-one into /opt/podplane/etc/user-data.env.
	Env EnvVars

	// NstanceRegistrationNonceJWT is written directly to nonce.jwt. It is not
	// rendered into user-data.env because it is a sensitive one-time credential.
	NstanceRegistrationNonceJWT string
}

// EnvVars are the environment variables rendered into user-data.env.
type EnvVars struct {
	SSHAuthorizedKey string // SSH_AUTHORIZED_KEY
	InstanceID       string `validate:"required"` // INSTANCE_ID
	ClusterID        string `validate:"required"` // CLUSTER_ID

	// ProviderKind identifies the environment the VM is running in.
	// One of: "local", "aws", "google", "proxmox".
	ProviderKind         string `validate:"required,oneof=local aws google proxmox"` // PROVIDER_KIND
	ProviderRegion       string // PROVIDER_REGION
	ProviderZone         string // PROVIDER_ZONE
	ProviderInstanceType string // PROVIDER_INSTANCE_TYPE
	AWSAccountID         string // AWS_ACCOUNT_ID
	GoogleProjectID      string // GOOGLE_PROJECT_ID

	OIDCIssuer   string `validate:"omitempty,url"` // OIDC_ISSUER
	OIDCCustomCA string // OIDC_CUSTOM_CA
	OIDCCAFile   string // OIDC_CA_FILE

	KubeLogLevel          string `validate:"required,uintstr"` // KUBE_LOG_LEVEL
	KubeAPIPublicHostname string // KUBE_API_PUBLIC_HOSTNAME
	KubeAPIPort           string `validate:"required,portstr"` // KUBE_API_PORT
	KubeAPIEtcdServers    string // KUBE_API_ETCD_SERVERS

	NstanceCACert                 string // NSTANCE_CA_CERT
	NstanceServerRegistrationAddr string // NSTANCE_SERVER_REGISTRATION_ADDR
	NstanceServerAgentAddr        string // NSTANCE_SERVER_AGENT_ADDR

	NetsyBucket          string `validate:"omitempty"`     // NETSY_BUCKET
	NetsyEndpoint        string `validate:"omitempty,url"` // NETSY_ENDPOINT
	NetsyRegion          string // NETSY_REGION
	NetsyAccessKeyID     string // NETSY_ACCESS_KEY_ID
	NetsySecretAccessKey string // NETSY_SECRET_ACCESS_KEY

	TelemetryBucket          string `validate:"omitempty"`     // TELEMETRY_BUCKET
	TelemetryEndpoint        string `validate:"omitempty,url"` // TELEMETRY_ENDPOINT
	TelemetryRegion          string // TELEMETRY_REGION
	TelemetryLogServices     string `validate:"omitempty,service_list"` // TELEMETRY_LOG_SERVICES
	TelemetryLogCloudinit    string `validate:"required,boolstr"`       // TELEMETRY_LOG_CLOUDINIT
	TelemetryAccessKeyID     string // TELEMETRY_ACCESS_KEY_ID
	TelemetrySecretAccessKey string // TELEMETRY_SECRET_ACCESS_KEY

	RegistryEnabled         string `validate:"required,boolstr"`           // REGISTRY_ENABLED
	RegistryHostname        string `validate:"omitempty,hostname_rfc1123"` // REGISTRY_HOSTNAME
	RegistryBucket          string `validate:"omitempty"`                  // REGISTRY_BUCKET
	RegistryEndpoint        string `validate:"omitempty,url"`              // REGISTRY_ENDPOINT
	RegistryRegion          string // REGISTRY_REGION
	RegistryAccessKeyID     string // REGISTRY_ACCESS_KEY_ID
	RegistrySecretAccessKey string // REGISTRY_SECRET_ACCESS_KEY
	AWSS3UsePathStyle       string `validate:"omitempty,boolstr"` // AWS_S3_USE_PATH_STYLE
}

// ManifestFilter selects cached dependencies that apply to this VM's provider.
func (v *TemplateVars) ManifestFilter() deps.ItemFilter {
	return deps.ItemFilter{Providers: []string{v.Env.ProviderKind}, CachedOnly: true}
}

// SetObjectStorageEndpoint sets all component object storage endpoints to the
// same value. Use direct field assignment when components use different stores.
func (v *EnvVars) SetObjectStorageEndpoint(endpoint string) {
	v.NetsyEndpoint = endpoint
	v.TelemetryEndpoint = endpoint
	v.RegistryEndpoint = endpoint
}

// SetObjectStorageRegion sets all component object storage regions to the same
// value. Use direct field assignment when components use different stores.
func (v *EnvVars) SetObjectStorageRegion(region string) {
	v.NetsyRegion = region
	v.TelemetryRegion = region
	v.RegistryRegion = region
}

// SetObjectStorageCredentials sets all component object storage credentials to
// the same values. Use direct field assignment when components use different
// credentials.
func (v *EnvVars) SetObjectStorageCredentials(accessKeyID, secretAccessKey string) {
	v.NetsyAccessKeyID = accessKeyID
	v.NetsySecretAccessKey = secretAccessKey
	v.TelemetryAccessKeyID = accessKeyID
	v.TelemetrySecretAccessKey = secretAccessKey
	v.RegistryAccessKeyID = accessKeyID
	v.RegistrySecretAccessKey = secretAccessKey
}

// ApplyDefaults populates derived values (e.g. cluster-prefixed bucket names)
// when the caller has not set them explicitly.
func (v *TemplateVars) ApplyDefaults() {
	env := &v.Env
	if env.KubeLogLevel == "" {
		env.KubeLogLevel = "2"
	}
	if env.KubeAPIPort == "" {
		env.KubeAPIPort = "6443"
	}
	if env.TelemetryLogCloudinit == "" {
		env.TelemetryLogCloudinit = "true"
	}
	if env.RegistryEnabled == "" {
		env.RegistryEnabled = "true"
	}
	if env.ClusterID == "" {
		return
	}
	if env.NetsyBucket == "" {
		env.NetsyBucket = fmt.Sprintf("%s-netsy", env.ClusterID)
	}
	if env.RegistryBucket == "" {
		env.RegistryBucket = fmt.Sprintf("%s-registry", env.ClusterID)
	}
	if env.TelemetryBucket == "" {
		env.TelemetryBucket = fmt.Sprintf("%s-telemetry", env.ClusterID)
	}
}

// validate is package-scoped so struct tags are cached across calls.
var validate = validator.New(validator.WithRequiredStructEnabled())

var serviceListRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*(,[a-z0-9][a-z0-9-]*)*$`)

func init() {
	if err := validate.RegisterValidation("uintstr", validateUintString); err != nil {
		panic(err)
	}
	if err := validate.RegisterValidation("portstr", validatePortString); err != nil {
		panic(err)
	}
	if err := validate.RegisterValidation("boolstr", validateBoolString); err != nil {
		panic(err)
	}
	if err := validate.RegisterValidation("service_list", validateServiceList); err != nil {
		panic(err)
	}
	validate.RegisterStructValidation(validateEnvVars, EnvVars{})
}

func validateUintString(fl validator.FieldLevel) bool {
	_, err := strconv.ParseUint(fl.Field().String(), 10, 64)
	return err == nil
}

func validatePortString(fl validator.FieldLevel) bool {
	port, err := strconv.ParseUint(fl.Field().String(), 10, 16)
	return err == nil && port > 0
}

func validateBoolString(fl validator.FieldLevel) bool {
	switch fl.Field().String() {
	case "true", "false":
		return true
	default:
		return false
	}
}

func validateServiceList(fl validator.FieldLevel) bool {
	return serviceListRE.MatchString(fl.Field().String())
}

func validateEnvVars(sl validator.StructLevel) {
	env, ok := sl.Current().Interface().(EnvVars)
	if !ok {
		return
	}
	value := reflect.ValueOf(env)
	typeOf := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() != reflect.String {
			continue
		}
		if strings.ContainsAny(field.String(), "'\n\r") {
			fieldName := typeOf.Field(i).Name
			sl.ReportError(field.Interface(), fieldName, fieldName, "envstr", "")
		}
	}
	if env.RegistryEnabled == "true" && env.RegistryHostname == "" {
		sl.ReportError(env.RegistryHostname, "RegistryHostname", "RegistryHostname", "required_if_registry_enabled", "")
	}
}

// Validate checks the TemplateVars are populated correctly enough to render.
func (v *TemplateVars) Validate() error {
	if err := validate.Struct(v); err != nil {
		if verrs, ok := err.(validator.ValidationErrors); ok {
			msg := ""
			for _, e := range verrs {
				msg += fmt.Sprintf("\n%s failed validation on '%s' validator.", e.Field(), e.Tag())
			}
			return fmt.Errorf("%s", msg)
		}
		return err
	}
	return nil
}

// Render produces the rendered user-data script.
func (v *TemplateVars) Render() (string, error) {
	v.ApplyDefaults()
	if err := v.Validate(); err != nil {
		return "", err
	}
	tmpl, err := template.New("userdata").Parse(userdataTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse userdata template: %w", err)
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, v); err != nil {
		return "", fmt.Errorf("failed to render userdata template: %w", err)
	}
	return sb.String(), nil
}
