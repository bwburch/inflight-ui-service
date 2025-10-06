-- Enhance quick_templates to store full workbench configuration state
ALTER TABLE quick_templates RENAME COLUMN template_data TO configuration_data;

COMMENT ON COLUMN quick_templates.configuration_data IS 'Complete workbench state: {llm_provider_id, prompt_version_id, proposed_changes[]}';

-- Add index for faster queries
CREATE INDEX idx_templates_created ON quick_templates(created_at DESC);
