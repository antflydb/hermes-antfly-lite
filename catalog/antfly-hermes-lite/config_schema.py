"""Dashboard configuration metadata for the Antfly Lite memory provider."""

from plugins.memory.config_schema import (
    KIND_BOOL,
    KIND_NUMBER,
    KIND_SELECT,
    KIND_TEXT,
    ProviderConfigSchema,
    ProviderField,
    ProviderFieldOption,
    STORAGE_FLAT_JSON,
)


CONFIG_SCHEMA = ProviderConfigSchema(
    name="antfly-hermes-lite",
    label="Antfly Lite",
    storage=STORAGE_FLAT_JSON,
    fields=(
        ProviderField(
            key="auto_capture",
            label="Capture completed turns",
            kind=KIND_BOOL,
            default="true",
            description="Store completed primary-agent turns as episodic memory.",
            inline=True,
        ),
        ProviderField(
            key="auto_recall",
            label="Automatic recall",
            kind=KIND_BOOL,
            default="true",
            description="Prefetch relevant memories before each turn.",
            inline=True,
        ),
        ProviderField(
            key="max_recall_results",
            label="Recall limit",
            kind=KIND_NUMBER,
            default="6",
            description="Maximum memories included as background evidence per turn (1–20).",
        ),
        ProviderField(
            key="default_scope",
            label="Default scope",
            kind=KIND_SELECT,
            default="user",
            description="Isolation boundary used when a memory is stored.",
            options=tuple(
                ProviderFieldOption(value=value, label=label)
                for value, label in (
                    ("user", "User"),
                    ("workspace", "Workspace"),
                    ("agent", "Agent"),
                    ("session", "Session"),
                    ("profile", "Profile"),
                )
            ),
        ),
        ProviderField(
            key="auto_extract_facts",
            label="Extract declarative facts",
            kind=KIND_BOOL,
            default="true",
            description="Store safe, short user declarations as separately retrievable facts.",
        ),
        ProviderField(
            key="semantic_recall",
            label="Semantic recall",
            kind=KIND_BOOL,
            default="false",
            description="Use Antfly Lite hybrid lexical and dense-vector retrieval.",
        ),
        ProviderField(
            key="embedding_url",
            label="Embedding URL",
            kind=KIND_TEXT,
            default="",
            placeholder="http://127.0.0.1:8082/ai/v1/embeddings",
            description="OpenAI-compatible Antfly Inference embeddings endpoint.",
            group="Semantic recall",
        ),
        ProviderField(
            key="embedding_model",
            label="Embedding model",
            kind=KIND_TEXT,
            default="",
            placeholder="bge-base-en-v1.5",
            description="Model identifier sent to the embeddings endpoint.",
            group="Semantic recall",
        ),
        ProviderField(
            key="embedding_space",
            label="Vector space",
            kind=KIND_TEXT,
            default="",
            placeholder="support-memory-v1",
            description="Stable identity pinned to the database; defaults to the model identifier.",
            group="Semantic recall",
        ),
        ProviderField(
            key="embedding_timeout_seconds",
            label="Embedding timeout",
            kind=KIND_NUMBER,
            default="3",
            description="Fail open to lexical retrieval after this many seconds.",
            group="Semantic recall",
        ),
    ),
)
