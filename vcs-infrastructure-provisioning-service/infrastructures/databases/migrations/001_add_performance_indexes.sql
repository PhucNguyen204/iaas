-- Performance indexes for high-traffic queries

CREATE INDEX IF NOT EXISTS idx_infra_user_type ON infrastructures(user_id, type);
CREATE INDEX IF NOT EXISTS idx_infra_user_status ON infrastructures(user_id, status);

CREATE INDEX IF NOT EXISTS idx_stack_user_env ON stacks(user_id, environment);
CREATE INDEX IF NOT EXISTS idx_stack_tenant_proj ON stacks(tenant_id, project_id);

CREATE INDEX IF NOT EXISTS idx_cluster_nodes_health ON cluster_nodes(cluster_id, is_healthy);
CREATE INDEX IF NOT EXISTS idx_cluster_nodes_role ON cluster_nodes(cluster_id, role);

CREATE INDEX IF NOT EXISTS idx_stacks_tags_gin ON stacks USING GIN (tags jsonb_path_ops);

CREATE INDEX IF NOT EXISTS idx_docker_services_status ON docker_services(infrastructure_id, status);
CREATE INDEX IF NOT EXISTS idx_nginx_instances_infra ON nginx_instances(infrastructure_id);

CREATE INDEX IF NOT EXISTS idx_postgres_databases_instance ON postgres_databases(instance_id, status);
CREATE INDEX IF NOT EXISTS idx_postgres_databases_tenant ON postgres_databases(tenant_id, project_id);
