INSERT INTO workspaces (id, name, kind)
VALUES ('ws_personal_dev', 'Development Workspace', 'personal')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, issuer, subject, email, display_name)
VALUES ('usr_dev', 'development', 'developer', 'developer@localhost', 'Development User')
ON CONFLICT (id) DO NOTHING;

INSERT INTO workspace_memberships (workspace_id, user_id, role)
VALUES ('ws_personal_dev', 'usr_dev', 'owner')
ON CONFLICT (workspace_id, user_id) DO NOTHING;

