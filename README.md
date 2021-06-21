# aws-iam-topology
AWS IAM topology project for multi account users.

# new4j
Run graphdb container
> podman run -d --rm -p7474:7474 -p7687:7687 -e NEO4J_AUTH=neo4j/s3cr3t --name graphdb neo4j

connect container service via your browser.(recommended chrome)
http://localhost:7474

Stop and delete graphdb container
> podman stop graphdb
