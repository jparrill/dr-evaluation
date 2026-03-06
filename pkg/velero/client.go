package velero

import (
	"context"
	"fmt"
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	backupGVR = schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "backups",
	}
	restoreGVR = schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "restores",
	}
)

type Client struct {
	dynClient dynamic.Interface
	namespace string
}

func NewClient(kubeconfigPath, namespace string) (*Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("building kubeconfig: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	return &Client{
		dynClient: dynClient,
		namespace: namespace,
	}, nil
}

func (c *Client) FetchBackups(ctx context.Context) ([]BackupInfo, error) {
	list, err := c.dynClient.Resource(backupGVR).Namespace(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing backups: %w", err)
	}

	var backups []BackupInfo
	for _, item := range list.Items {
		b := parseBackup(item)
		backups = append(backups, b)
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreationTimestamp.Before(backups[j].CreationTimestamp)
	})

	return backups, nil
}

func (c *Client) FetchRestores(ctx context.Context) ([]RestoreInfo, error) {
	list, err := c.dynClient.Resource(restoreGVR).Namespace(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing restores: %w", err)
	}

	var restores []RestoreInfo
	for _, item := range list.Items {
		r := parseRestore(item)
		restores = append(restores, r)
	}

	sort.Slice(restores, func(i, j int) bool {
		return restores[i].CreationTimestamp.Before(restores[j].CreationTimestamp)
	})

	return restores, nil
}

func parseBackup(obj unstructured.Unstructured) BackupInfo {
	name := obj.GetName()
	ct := obj.GetCreationTimestamp().Time

	status, _ := getNestedMap(obj.Object, "status")
	spec, _ := getNestedMap(obj.Object, "spec")

	phase := getStringField(status, "phase")
	startTS := parseTimestamp(getStringField(status, "startTimestamp"))
	compTS := parseTimestamp(getStringField(status, "completionTimestamp"))

	progress, _ := getNestedMap(status, "progress")
	itemsBackedUp := getIntField(progress, "itemsBackedUp")
	totalItems := getIntField(progress, "totalItems")
	warnings := getIntField(status, "warnings")
	errors := getIntField(status, "errors")

	ttl := getStringField(spec, "ttl")
	storageLocation := getStringField(spec, "storageLocation")
	includedNamespaces := getStringSlice(spec, "includedNamespaces")

	var dur time.Duration
	if !startTS.IsZero() && !compTS.IsZero() {
		dur = compTS.Sub(startTS)
	}

	return BackupInfo{
		Name:                name,
		Type:                ClassifyBackup(name),
		CreationTimestamp:   ct,
		StartTimestamp:      startTS,
		CompletionTimestamp: compTS,
		Duration:            dur,
		Phase:               phase,
		ItemsBackedUp:       itemsBackedUp,
		TotalItems:          totalItems,
		Warnings:            warnings,
		Errors:              errors,
		TTL:                 ttl,
		IncludedNamespaces:  includedNamespaces,
		StorageLocation:     storageLocation,
	}
}

func parseRestore(obj unstructured.Unstructured) RestoreInfo {
	name := obj.GetName()
	ct := obj.GetCreationTimestamp().Time

	status, _ := getNestedMap(obj.Object, "status")
	spec, _ := getNestedMap(obj.Object, "spec")

	phase := getStringField(status, "phase")
	startTS := parseTimestamp(getStringField(status, "startTimestamp"))
	compTS := parseTimestamp(getStringField(status, "completionTimestamp"))

	progress, _ := getNestedMap(status, "progress")
	itemsRestored := getIntField(progress, "itemsRestored")
	totalItems := getIntField(progress, "totalItems")
	warnings := getIntField(status, "warnings")
	errors := getIntField(status, "errors")
	backupName := getStringField(spec, "backupName")

	var dur time.Duration
	if !startTS.IsZero() && !compTS.IsZero() {
		dur = compTS.Sub(startTS)
	}

	return RestoreInfo{
		Name:                name,
		BackupName:          backupName,
		CreationTimestamp:   ct,
		StartTimestamp:      startTS,
		CompletionTimestamp: compTS,
		Duration:            dur,
		Phase:               phase,
		ItemsRestored:       itemsRestored,
		TotalItems:          totalItems,
		Warnings:            warnings,
		Errors:              errors,
	}
}

func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func getNestedMap(obj map[string]interface{}, key string) (map[string]interface{}, bool) {
	val, ok := obj[key]
	if !ok {
		return nil, false
	}
	m, ok := val.(map[string]interface{})
	return m, ok
}

func getStringField(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	val, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := val.(string)
	if !ok {
		return ""
	}
	return s
}

func getIntField(m map[string]interface{}, key string) int {
	if m == nil {
		return 0
	}
	val, ok := m[key]
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case int64:
		return int(v)
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func getStringSlice(m map[string]interface{}, key string) []string {
	if m == nil {
		return nil
	}
	val, ok := m[key]
	if !ok {
		return nil
	}
	slice, ok := val.([]interface{})
	if !ok {
		return nil
	}
	var result []string
	for _, v := range slice {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
