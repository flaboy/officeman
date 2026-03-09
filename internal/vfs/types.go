package vfs

type ResolvedFile struct {
	MountPath    string
	Permission   string
	BucketAlias  string
	BucketName   string
	S3Prefix     string
	RelativePath string
	S3Key        string
}

type ResolveError struct {
	Code    string
	Message string
}

func (e *ResolveError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}
