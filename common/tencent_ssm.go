package common

import (
	"fmt"
	"os"
	"strings"

	tencentcommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	ssm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssm/v20190923"
)

// InitTencentCloudSecrets loads connection strings from Tencent Cloud Secrets
// Manager before the database and Redis clients are initialized. Authentication
// uses the SDK provider chain, so CVM instance roles and TKE OIDC can be used
// without putting a SecretId or SecretKey in the container environment.
func InitTencentCloudSecrets() error {
	if !GetEnvOrDefaultBool("TENCENTCLOUD_SSM_ENABLED", false) {
		return nil
	}

	region := strings.TrimSpace(os.Getenv("TENCENTCLOUD_REGION"))
	if region == "" {
		region = strings.TrimSpace(os.Getenv("TENCENTCLOUD_SSM_REGION"))
	}
	if region == "" {
		return fmt.Errorf("TENCENTCLOUD_SSM_REGION is required when TENCENTCLOUD_SSM_ENABLED=true")
	}

	credential, err := tencentcommon.DefaultProviderChain().GetCredential()
	if err != nil {
		return fmt.Errorf("get Tencent Cloud credential: %w", err)
	}
	client, err := ssm.NewClient(credential, region, profile.NewClientProfile())
	if err != nil {
		return fmt.Errorf("create Tencent Cloud SSM client: %w", err)
	}

	secrets := []struct {
		envName    string
		secretName string
	}{
		{"SQL_DSN", os.Getenv("TENCENTCLOUD_SSM_SQL_DSN_SECRET")},
		{"LOG_SQL_DSN", os.Getenv("TENCENTCLOUD_SSM_LOG_SQL_DSN_SECRET")},
		{"REDIS_CONN_STRING", os.Getenv("TENCENTCLOUD_SSM_REDIS_SECRET")},
	}
	if bundleName := strings.TrimSpace(os.Getenv("TENCENTCLOUD_SSM_BUNDLE_SECRET")); bundleName != "" {
		bundle, err := getTencentSecret(client, bundleName)
		if err != nil {
			return fmt.Errorf("load Tencent Cloud secret %q: %w", bundleName, err)
		}
		values := map[string]string{}
		if err := Unmarshal([]byte(bundle), &values); err != nil {
			return fmt.Errorf("parse Tencent Cloud secret %q as JSON bundle: %w", bundleName, err)
		}
		for envName, value := range values {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("Tencent Cloud secret %q contains an empty value for %s", bundleName, envName)
			}
			if err := os.Setenv(envName, value); err != nil {
				return fmt.Errorf("set %s from Tencent Cloud SSM: %w", envName, err)
			}
		}
	}
	for _, item := range secrets {
		if strings.TrimSpace(item.secretName) == "" {
			continue
		}
		value, err := getTencentSecret(client, item.secretName)
		if err != nil {
			return fmt.Errorf("load Tencent Cloud secret %q for %s: %w", item.secretName, item.envName, err)
		}
		if err := os.Setenv(item.envName, value); err != nil {
			return fmt.Errorf("set %s from Tencent Cloud SSM: %w", item.envName, err)
		}
	}
	return nil
}

func getTencentSecret(client *ssm.Client, secretName string) (string, error) {
	request := ssm.NewGetSecretValueRequest()
	request.SecretName = tencentcommon.StringPtr(secretName)
	request.VersionId = tencentcommon.StringPtr("SSM_Current")
	response, err := client.GetSecretValue(request)
	if err != nil {
		return "", err
	}
	if response == nil || response.Response == nil || response.Response.SecretString == nil {
		return "", fmt.Errorf("secret has no SecretString value")
	}
	value := strings.TrimSpace(*response.Response.SecretString)
	if value == "" {
		return "", fmt.Errorf("secret value is empty")
	}
	return value, nil
}
