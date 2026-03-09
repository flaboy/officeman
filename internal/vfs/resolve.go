package vfs

import (
	"strings"

	"github.com/github-flaboy/officeman/internal/api"
)

type Resolver interface {
	ResolveFile(ctx api.VFSContext, filePath string) (ResolvedFile, *ResolveError)
}

func ResolveFile(ctx api.VFSContext, filePath string) (ResolvedFile, *ResolveError) {
	if err := ValidateVirtualPath(filePath); err != nil {
		return ResolvedFile{}, err
	}

	mountPath, mount, ok := findLongestMount(filePath, ctx.Mounts)
	if !ok {
		return ResolvedFile{}, resolveErr("VFS_PATH_NOT_MOUNTED", "filePath is not covered by any mount")
	}

	s3Set, ok := ctx.S3Sets[mount.Bucket]
	if !ok {
		return ResolvedFile{}, resolveErr("VFS_S3_SET_NOT_FOUND", "mount bucket alias is not configured in vfs.s3_sets")
	}

	resolvedPrefix, err := resolveTemplate(mount.Path, ctx.TemplateVars)
	if err != nil {
		return ResolvedFile{}, err
	}
	resolvedPrefix = normalizeS3Prefix(resolvedPrefix)

	relativePath := strings.TrimPrefix(filePath, mountPath)

	return ResolvedFile{
		MountPath:    mountPath,
		Permission:   mount.Permission,
		BucketAlias:  mount.Bucket,
		BucketName:   s3Set.Bucket,
		S3Prefix:     resolvedPrefix,
		RelativePath: relativePath,
		S3Key:        resolvedPrefix + relativePath,
	}, nil
}

func ValidateVirtualPath(filePath string) *ResolveError {
	if filePath == "" || !strings.HasPrefix(filePath, "/") {
		return resolveErr("VFS_INVALID_PATH", "filePath must be an absolute vfs path")
	}

	segments := strings.Split(filePath, "/")
	for _, segment := range segments {
		if segment == ".." {
			return resolveErr("VFS_PATH_TRAVERSAL", "filePath cannot contain traversal segments")
		}
	}

	return nil
}

func normalizeS3Prefix(prefix string) string {
	out := strings.TrimSpace(prefix)
	out = strings.TrimPrefix(out, "/")
	if out != "" && !strings.HasSuffix(out, "/") {
		out += "/"
	}
	return out
}

func resolveTemplate(tpl string, vars map[string]string) (string, *ResolveError) {
	out := tpl
	start := strings.Index(out, "{")
	for start >= 0 {
		end := strings.Index(out[start:], "}")
		if end < 0 {
			break
		}
		end += start
		name := out[start+1 : end]
		value := strings.TrimSpace(vars[name])
		if value == "" {
			return "", resolveErr("VFS_MISSING_TEMPLATE_VAR", "missing template variable: "+name)
		}
		out = out[:start] + value + out[end+1:]
		start = strings.Index(out, "{")
	}
	return out, nil
}

func findLongestMount(filePath string, mounts map[string]api.VFSMount) (string, api.VFSMount, bool) {
	var (
		bestPath string
		best     api.VFSMount
		found    bool
	)
	for path, mount := range mounts {
		if !strings.HasPrefix(filePath, path) {
			continue
		}
		if !found || len(path) > len(bestPath) {
			bestPath = path
			best = mount
			found = true
		}
	}
	return bestPath, best, found
}

func resolveErr(code, message string) *ResolveError {
	return &ResolveError{Code: code, Message: message}
}
