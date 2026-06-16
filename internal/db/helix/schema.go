package helix

import (
	"context"

	helixsdk "github.com/helixdb/helix-db/sdks/go"
)

// InitSchema creates all required indexes idempotently.
// It is safe to call multiple times — each index creation uses
// CreateIndexIfNotExists internally.
func (c *Client) InitSchema(ctx context.Context) error {
	if c == nil || c.inner == nil {
		return ErrConnection
	}

	req := c.buildSchemaIndexes()
	return c.inner.Exec(ctx, req, nil, helixsdk.WriterOnly(), helixsdk.AwaitDurability(true))
}

// buildSchemaIndexes constructs the WriteQuery that defines all indexes
// for the Zyro domain model.
func (c *Client) buildSchemaIndexes() helixsdk.Request {
	q := helixsdk.WriteQuery("create_schema_indexes")

	// -- Equality indexes (unique where applicable) --

	q = q.VarAs("idx_dev_name",
		helixsdk.G().CreateIndexIfNotExists(
			helixsdk.NodeUniqueEqualityIndex("Developer", "name"),
		),
	)
	q = q.VarAs("idx_proj_name",
		helixsdk.G().CreateIndexIfNotExists(
			helixsdk.NodeEqualityIndex("Project", "name"),
		),
	)
	q = q.VarAs("idx_proj_tenant",
		helixsdk.G().CreateIndexIfNotExists(
			helixsdk.NodeEqualityIndex("Project", "project_id"),
		),
	)
	q = q.VarAs("idx_doc_topic",
		helixsdk.G().CreateIndexIfNotExists(
			helixsdk.NodeEqualityIndex("Document", "topic_key"),
		),
	)
	q = q.VarAs("idx_skill_name",
		helixsdk.G().CreateIndexIfNotExists(
			helixsdk.NodeUniqueEqualityIndex("Skill", "name"),
		),
	)
	q = q.VarAs("idx_codenode_path",
		helixsdk.G().CreateIndexIfNotExists(
			helixsdk.NodeEqualityIndex("CodeNode", "path"),
		),
	)

	// -- Vector indexes (1536-dim cosine, project-scoped) --

	q = q.VarAs("idx_pattern_emb",
		helixsdk.G().CreateVectorIndexNodes("Pattern", "embedding", "project_id"),
	)
	q = q.VarAs("idx_doc_emb",
		helixsdk.G().CreateVectorIndexNodes("Document", "embedding", "project_id"),
	)
	q = q.VarAs("idx_skill_emb",
		helixsdk.G().CreateVectorIndexNodes("Skill", "embedding"),
	)

	// -- Text indexes (BM25) --

	q = q.VarAs("idx_doc_content",
		helixsdk.G().CreateTextIndexNodes("Document", "content", "project_id"),
	)

	// -- Range indexes --

	q = q.VarAs("idx_task_created",
		helixsdk.G().CreateIndexIfNotExists(
			helixsdk.NodeRangeIndex("Task", "created_at"),
		),
	)

	return q.Returning()
}
