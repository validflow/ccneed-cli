package resources

// KnowledgeBase 表示可选择性、采集状态及已发布数据，不公开内部 script_run。
type KnowledgeBase struct {
	ID                 int64    `json:"id"`
	BusinessID         *string  `json:"biz_id"`
	Name               string   `json:"name"`
	Status             *string  `json:"status"`
	Platforms          []string `json:"platforms"`
	PostCount          int64    `json:"post_count"`
	PublishedPostCount int64    `json:"published_post_count"`
	ActiveRunStatus    *string  `json:"active_run_status"`
	CollectedCount     int64    `json:"active_run_collected_count"`
	CanSelect          bool     `json:"can_select"`
	DisabledReason     *string  `json:"disabled_reason"`
	FailureReason      *string  `json:"failure_reason"`
	DeletedAt          *string  `json:"deleted_at"`
}

// KnowledgeListing 返回当前账号知识库摘要，不提供采集配额修改。
type KnowledgeListing struct {
	KnowledgeBases []KnowledgeBase `json:"knowledge_bases"`
}

// KBDistribution 只携带分类标签和聚合数量，不包含单条内容。
type KBDistribution struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

// KnowledgeBaseStats 是当前账号的已发布 KB 概览，不提供帖子明细。
type KnowledgeBaseStats struct {
	KBID               int64            `json:"kb_id"`
	AvailablePostCount int64            `json:"available_post_count"`
	Platforms          []KBDistribution `json:"platforms"`
	Sentiments         []KBDistribution `json:"sentiments"`
	TopTopics          []KBDistribution `json:"top_topics"`
}

// UserFile 表示独立上传后可用于绑定的文件，不含存储 object key。
type UserFile struct {
	ID          string  `json:"library_file_id"`
	Name        string  `json:"display_name"`
	Size        int64   `json:"size_bytes"`
	ContentType *string `json:"content_type"`
	Status      string  `json:"status"`
}

// FileListing 保留服务列表上限，便于调用方识别可能截断的数据。
type FileListing struct {
	Files           []UserFile `json:"files"`
	Limit           int        `json:"limit"`
	HasMorePossible bool       `json:"has_more_possible"`
}

// Bindings 是服务器所有的资源窗口和 CAS 版本，不是本地缓存。
type Bindings struct {
	ThreadID string   `json:"thread_id"`
	KBIDs    []int64  `json:"kb_ids"`
	FileIDs  []string `json:"library_file_ids"`
	Version  string   `json:"metadata_version"`
	Window   string   `json:"resource_window_state"`
}

// Appended 是追加回执，不包含解绑或替换能力。
type Appended struct {
	Bindings
	AddedKBIDs   []int64  `json:"added_kb_ids"`
	AddedFileIDs []string `json:"added_library_file_ids"`
}
