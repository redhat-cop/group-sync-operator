package constants

const (
	AnnotationBase    = "group-sync-operator.redhat-cop.io"
	SyncTimestamp     = AnnotationBase + "/sync-time"
	SyncSourceURL     = AnnotationBase + "/sync.source.url"
	SyncSourceHost    = AnnotationBase + "/sync.source.host"
	SyncSourceUID     = AnnotationBase + "/sync.source.uid"
	SyncProvider      = AnnotationBase + "/sync-provider"
	HierarchyChildren = "hierarchy_children"
	HierarchyParent   = "hierarchy_parent"
	HierarchyParents  = "hierarchy_parents"
	ISO8601Layout     = "2006-01-02T15:04:05Z0700"
)
