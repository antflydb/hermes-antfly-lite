"""Dashboard configuration metadata for the Antfly Lite memory provider."""

from plugins.memory.config_schema import (
    KIND_BOOL,
    KIND_NUMBER,
    KIND_SELECT,
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
    ),
)
