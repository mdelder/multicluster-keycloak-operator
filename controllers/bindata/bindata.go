// Code generated for package bindata by go-bindata DO NOT EDIT. (@generated)
// sources:
// controllers/manifests/clusterrole.yaml
// controllers/manifests/clusterrolebinding.yaml
// controllers/manifests/oauthclient.yaml
// controllers/manifests/secret.yaml
package bindata

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _controllersManifestsClusterroleYaml = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: open-cluster-management:klusterlet-work-sa:agent:oauth-edit
rules:
- apiGroups: ["config.openshift.io"]
  resources: ["oauths"]
  verbs: ["create", "get", "list", "update", "delete"]`)

func controllersManifestsClusterroleYamlBytes() ([]byte, error) {
	return _controllersManifestsClusterroleYaml, nil
}

func controllersManifestsClusterroleYaml() (*asset, error) {
	bytes, err := controllersManifestsClusterroleYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "controllers/manifests/clusterrole.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _controllersManifestsClusterrolebindingYaml = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: open-cluster-management:klusterlet-work-sa:agent:oauth-edit"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: open-cluster-management:klusterlet-work-sa:agent:oauth-edit

subjects:
- kind: ServiceAccount
  name: klusterlet-work-sa
  namespace: open-cluster-management-agent`)

func controllersManifestsClusterrolebindingYamlBytes() ([]byte, error) {
	return _controllersManifestsClusterrolebindingYaml, nil
}

func controllersManifestsClusterrolebindingYaml() (*asset, error) {
	bytes, err := controllersManifestsClusterrolebindingYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "controllers/manifests/clusterrolebinding.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _controllersManifestsOauthclientYaml = []byte(`apiVersion: config.openshift.io/v1
kind: OAuth
metadata:
  name: cluster
spec:
  identityProviders:
    name: oidcidp
    mappingMethod: claim
    type: OpenID
    openID:
      clientID: {{ .ClientID }}
      clientSecret:
        name: {{ .ClientSecret }}
      ca:
        name: ca-config-map
      claims:
        preferredUsername: ["email"]
        name: ["name"]
        email: ["email"]
      issuer: {{ .Issuer }}`)

func controllersManifestsOauthclientYamlBytes() ([]byte, error) {
	return _controllersManifestsOauthclientYaml, nil
}

func controllersManifestsOauthclientYaml() (*asset, error) {
	bytes, err := controllersManifestsOauthclientYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "controllers/manifests/oauthclient.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _controllersManifestsSecretYaml = []byte(`apiVersion: v1
kind: Secret
metadata:
  name: {{ .OAuthConfigSecretName }}
  namespace: openshift-config
  labels:
    app: sso
data:
  clientID: {{ .EncodedClientID }}
  clientSecret: {{ .EncodedClientSecret }}
type: Opaque`)

func controllersManifestsSecretYamlBytes() ([]byte, error) {
	return _controllersManifestsSecretYaml, nil
}

func controllersManifestsSecretYaml() (*asset, error) {
	bytes, err := controllersManifestsSecretYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "controllers/manifests/secret.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"controllers/manifests/clusterrole.yaml":        controllersManifestsClusterroleYaml,
	"controllers/manifests/clusterrolebinding.yaml": controllersManifestsClusterrolebindingYaml,
	"controllers/manifests/oauthclient.yaml":        controllersManifestsOauthclientYaml,
	"controllers/manifests/secret.yaml":             controllersManifestsSecretYaml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"controllers": {nil, map[string]*bintree{
		"manifests": {nil, map[string]*bintree{
			"clusterrole.yaml":        {controllersManifestsClusterroleYaml, map[string]*bintree{}},
			"clusterrolebinding.yaml": {controllersManifestsClusterrolebindingYaml, map[string]*bintree{}},
			"oauthclient.yaml":        {controllersManifestsOauthclientYaml, map[string]*bintree{}},
			"secret.yaml":             {controllersManifestsSecretYaml, map[string]*bintree{}},
		}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
