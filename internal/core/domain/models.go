package domain

type Permission string

const (
	PermissionOwner        Permission = "Owner"
	PermissionCollaborator Permission = "Collaborator"
)

const (
	AccessKeyMask   = "(hidden)"
	ReleaseRollback = "Rollback"
)

type Account struct {
	ID              string   `json:"-"`
	Email           string   `json:"email"`
	Name            string   `json:"name"`
	LinkedProviders []string `json:"linkedProviders"`
	CreatedAt       int64    `json:"-"`
}

type AccessKey struct {
	ID           string `json:"-"`
	AccountID    string `json:"-"`
	Name         string `json:"name,omitempty"`
	FriendlyName string `json:"friendlyName,omitempty"`
	Description  string `json:"description,omitempty"`
	CreatedBy    string `json:"createdBy,omitempty"`
	CreatedTime  int64  `json:"createdTime,omitempty"`
	Expires      int64  `json:"expires"`
	IsSession    bool   `json:"isSession,omitempty"`
}

type AccessKeyRequest struct {
	Name         string `json:"name,omitempty"`
	FriendlyName string `json:"friendlyName,omitempty"`
	Description  string `json:"description,omitempty"`
	CreatedBy    string `json:"createdBy,omitempty"`
	TTL          int64  `json:"ttl,omitempty"`
}

type CollaboratorProperties struct {
	IsCurrentAccount bool       `json:"isCurrentAccount,omitempty"`
	Permission       Permission `json:"permission"`
}

type App struct {
	ID            string                            `json:"-"`
	Name          string                            `json:"name"`
	CreatedAt     int64                             `json:"-"`
	Deployments   []string                          `json:"deployments,omitempty"`
	Collaborators map[string]CollaboratorProperties `json:"collaborators,omitempty"`
}

type AppCreationRequest struct {
	Name                         string `json:"name"`
	ManuallyProvisionDeployments bool   `json:"manuallyProvisionDeployments,omitempty"`
}

type AppPatchRequest struct {
	Name string `json:"name"`
}

type Deployment struct {
	ID        string   `json:"-"`
	AppID     string   `json:"-"`
	Name      string   `json:"name"`
	Key       string   `json:"key"`
	CreatedAt int64    `json:"-"`
	Package   *Package `json:"package,omitempty"`
}

type DeploymentRequest struct {
	Name string `json:"name"`
	Key  string `json:"key,omitempty"`
}

type DeploymentPatchRequest struct {
	Name string `json:"name"`
}

type Package struct {
	ID                 string `json:"-"`
	DeploymentID       string `json:"-"`
	Label              string `json:"label,omitempty"`
	AppVersion         string `json:"appVersion,omitempty"`
	Description        string `json:"description,omitempty"`
	IsDisabled         bool   `json:"isDisabled,omitempty"`
	IsMandatory        bool   `json:"isMandatory,omitempty"`
	PackageHash        string `json:"packageHash,omitempty"`
	BlobURL            string `json:"blobUrl,omitempty"`
	ManifestBlobURL    string `json:"-"`
	Rollout            *int   `json:"rollout,omitempty"`
	Size               int64  `json:"size,omitempty"`
	UploadTime         int64  `json:"uploadTime,omitempty"`
	ReleaseMethod      string `json:"releaseMethod,omitempty"`
	OriginalLabel      string `json:"originalLabel,omitempty"`
	OriginalDeployment string `json:"originalDeployment,omitempty"`
	ReleasedBy         string `json:"releasedBy,omitempty"`
	Ordinal            int    `json:"-"`
}

type UpdateMetrics struct {
	Active     int64 `json:"active"`
	Downloaded int64 `json:"downloaded,omitempty"`
	Failed     int64 `json:"failed,omitempty"`
	Installed  int64 `json:"installed,omitempty"`
}

type UpdateCheckRequest struct {
	DeploymentKey  string
	AppVersion     string
	PackageHash    string
	IsCompanion    bool
	Label          string
	ClientUniqueID string
}

type UpdateCheckResponse struct {
	Label                  string `json:"label,omitempty"`
	AppVersion             string `json:"appVersion,omitempty"`
	Description            string `json:"description,omitempty"`
	IsDisabled             bool   `json:"isDisabled,omitempty"`
	IsMandatory            bool   `json:"isMandatory,omitempty"`
	PackageHash            string `json:"packageHash,omitempty"`
	DownloadURL            string `json:"downloadURL,omitempty"`
	PackageSize            int64  `json:"packageSize,omitempty"`
	IsAvailable            bool   `json:"isAvailable"`
	ShouldRunBinaryVersion bool   `json:"shouldRunBinaryVersion,omitempty"`
	UpdateAppVersion       bool   `json:"updateAppVersion,omitempty"`
	TargetBinaryRange      string `json:"target_binary_range,omitempty"`
}

type DeploymentStatusReport struct {
	AppVersion                string `json:"appVersion"`
	ClientUniqueID            string `json:"clientUniqueId"`
	DeploymentKey             string `json:"deploymentKey"`
	PreviousDeploymentKey     string `json:"previousDeploymentKey"`
	PreviousLabelOrAppVersion string `json:"previousLabelOrAppVersion"`
	Label                     string `json:"label"`
	Status                    string `json:"status"`
}

type DownloadReport struct {
	ClientUniqueID string `json:"clientUniqueId"`
	DeploymentKey  string `json:"deploymentKey"`
	Label          string `json:"label"`
}
