package options

import "github.com/hydn-co/mesh-sdk/pkg/connectorutil"

func (o *AccountEntityCollectorOptions) Validate() error {
	return requireBaseURL("account collector options", o.BaseURL)
}

func (o *GroupEntityCollectorOptions) Validate() error {
	return requireBaseURL("group collector options", o.BaseURL)
}

func (o *OwnerEntityCollectorOptions) Validate() error {
	return requireBaseURL("owner collector options", o.BaseURL)
}

func (o *ApplicationRoleEntityCollectorOptions) Validate() error {
	return requireBaseURL("application role collector options", o.BaseURL)
}

func requireBaseURL(subject, baseURL string) error {
	return connectorutil.RequireStrings(
		subject,
		connectorutil.RequiredString{Name: "base_url", Value: baseURL},
	)
}
